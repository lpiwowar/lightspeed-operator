#!/bin/bash

# govulncheck-wrapper.sh - Run govulncheck while ignoring specified vulnerabilities
#
# Usage: ./hack/govulncheck-wrapper.sh [--config FILE] [--verbose]
#
# Configuration file format (YAML):
#   govulncheck_ignore_list_expiry_date: "2000-01-01"
#   ignored_vulnerabilities:
#     - id: GO-2024-12345
#       module: github.com/example/module
#       reason: "Acceptable risk in our context"
#

set -euo pipefail

# Global variables
CONFIG_FILE=".govulncheck-ignore.yaml"
GOVULNCHECK_BIN="${GOVULNCHECK_BIN:-./bin/govulncheck}"
VERBOSE=1
VULN_JSON=""

print_usage() {
  cat << 'EOF'
Usage: govulncheck-wrapper.sh [options]

Options:
  --config FILE     Path to YAML config file (default: .govulncheck-ignore.yaml)
  --verbose         Enable verbose output
  -h, --help        Show this help message

Configuration file format (YAML):
  govulncheck_ignore_list_expiry_date: "2000-01-01"
  ignored_vulnerabilities:
    - id: GO-2024-12345
      module: github.com/example/module
      reason: "Acceptable risk in our context"

EOF
}

log_info() {
  if [[ $VERBOSE -eq 1 ]]; then
    echo "[INFO] $*"
  fi
}

log_error() {
  echo "[ERROR] $*" >&2
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --config)
        CONFIG_FILE="$2"
        shift 2
        ;;
      --verbose)
        VERBOSE=1
        shift
        ;;
      -h|--help)
        print_usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        print_usage
        exit 1
        ;;
    esac
  done
}

check_dependencies() {
  if ! command -v "${GOVULNCHECK_BIN}" &> /dev/null; then
    log_error "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
    exit 1
  fi

  if ! command -v jq &> /dev/null; then
    log_error "jq not found. Install with your package manager (e.g., apt install jq)"
    exit 1
  fi

  if ! command -v yq &> /dev/null; then
    log_error "yq not found. Install with: go install github.com/mikefarah/yq/v4@latest"
    exit 1
  fi
}

validate_config() {
  if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
  fi

  log_info "Using config file: $CONFIG_FILE"

  local expiry_date
  expiry_date=$(yq -r '.govulncheck_ignore_list_expiry_date // ""' "$CONFIG_FILE")

  if [[ -z "$expiry_date" ]]; then
    log_error "Config is missing required field: govulncheck_ignore_list_expiry_date"
    exit 1
  fi

  local expiry_epoch now_epoch max_epoch
  expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null) || {
    log_error "Cannot parse govulncheck_ignore_list_expiry_date: $expiry_date"
    exit 1
  }
  now_epoch=$(date +%s)
  max_epoch=$(date -d "+3 months" +%s)

  if [[ $expiry_epoch -lt $now_epoch ]]; then
    log_error "govulncheck ignore list has expired ($expiry_date). Update the list and set a new expiry date in .govulncheck-ignore.yaml."
    exit 1
  fi

  if [[ $expiry_epoch -gt $max_epoch ]]; then
    log_error "govulncheck_ignore_list_expiry_date ($expiry_date) is more than 3 months from today. Maximum allowed: $(date -d '+3 months' +%Y-%m-%d)."
    exit 1
  fi

  log_info "Ignore list expiry date: $expiry_date (valid)"
}

run_govulncheck() {
  log_info "Running govulncheck..."

  local govulncheck_exit
  set +e
  VULN_JSON=$(${GOVULNCHECK_BIN} -tags gcs -json ./... 2>&1)
  govulncheck_exit=$?
  set -e

  # With -json, govulncheck exits 0 even when vulnerabilities are reported; a non-zero
  # exit means the scan did not finish (toolchain mismatch, package load errors, etc.).
  # Findings are determined from the JSON stream below, not from govulncheck_exit.
  if [[ $govulncheck_exit -ne 0 ]]; then
    log_error "govulncheck failed (exit $govulncheck_exit)"
    echo "$VULN_JSON" >&2
    exit 1
  fi

  # JSON mode may emit partial records before appending plain-text errors.
  if grep -qE '^govulncheck:' <<< "$VULN_JSON"; then
    log_error "govulncheck reported errors"
    echo "$VULN_JSON" >&2
    exit 1
  fi

  # govulncheck -json emits a stream of pretty-printed JSON objects (not one value per line).
  if [[ -n "$VULN_JSON" ]]; then
    local jq_stderr
    if ! jq_stderr=$(echo "$VULN_JSON" | jq -e -n 'inputs' 2>&1 >/dev/null); then
      log_error "govulncheck output is not valid JSON (govulncheck exit $govulncheck_exit)"
      log_error "$jq_stderr"
      log_error "raw govulncheck output:"
      echo "$VULN_JSON" >&2
      exit 1
    fi
  fi
}

check_vulnerabilities() {
  # Extract findings from the JSON stream (one pretty-printed object per value)
  # Each finding has: osv (ID), fixed_version (optional), trace[0].module
  # Only consider vulnerabilities where our code actually calls the vulnerable function
  # (trace length > 1 means there's a call path from our code to the vulnerable function)
  local findings
  findings=$(echo "$VULN_JSON" | jq -c 'select(.finding) | select(.finding.trace | length > 1) | {id: .finding.osv, module: .finding.trace[0].module, fixed: .finding.fixed_version}')

  if [[ -z "$findings" ]]; then
    log_info "No vulnerabilities found"
    return 0
  fi

  # Get unique vulnerabilities (same ID+module can appear multiple times with different traces)
  local unique_vulns
  unique_vulns=$(echo "$findings" | jq -c -s 'unique_by(.id + .module)' | jq -c '.[]')

  # Parse ignored vulnerabilities from config into a format we can match
  local ignored_list
  ignored_list=$(yq -r '.ignored_vulnerabilities[] | "\(.id)|\(.module)"' "$CONFIG_FILE" 2>/dev/null || true)

  log_info "Ignored vulnerabilities in config: $(echo "$ignored_list" | grep -c . || echo 0)"

  local ignored_count=0
  local unignored_count=0
  local unignored_vulns=""

  while IFS= read -r vuln; do
    [[ -z "$vuln" ]] && continue

    local vuln_id module fixed
    vuln_id=$(echo "$vuln" | jq -r '.id')
    module=$(echo "$vuln" | jq -r '.module')
    fixed=$(echo "$vuln" | jq -r '.fixed // "N/A"')

    if grep -qxF "${vuln_id}|${module}" <<< "$ignored_list"; then
      ((ignored_count++)) || true
      log_info "Ignored: $vuln_id in $module (fixed: $fixed)"
    else
      ((unignored_count++)) || true
      unignored_vulns="${unignored_vulns}  - $vuln_id in $module (fixed: $fixed)\n"
      log_error "Found: $vuln_id in $module (fixed: $fixed)"
    fi
  done <<< "$unique_vulns"

  log_info "Found $unignored_count unignored vulnerabilities, $ignored_count ignored"

  if [[ $unignored_count -gt 0 ]]; then
    log_error "Unignored vulnerabilities found:"
    echo -e "$unignored_vulns" >&2
    exit 1
  fi
}

main() {
  parse_args "$@"
  check_dependencies
  validate_config
  run_govulncheck
  check_vulnerabilities
}

main "$@"
