/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"crypto/sha256"
	"encoding/hex"
)

const clusterRBACScopeLabel = "lightspeed.openstack.org/rbac-scope"

// clusterRBACScopeID is a stable, DNS-label-safe identity shared with the
// telemetry operator. Keep its SHA-256 prefix length and inputs in sync there.
func clusterRBACScopeID(namespace string) string {
	sum := sha256.Sum256([]byte(namespace))
	return hex.EncodeToString(sum[:])[:12]
}

func sarClusterRoleName(namespace string) string {
	return "lightspeed-sar-role-" + clusterRBACScopeID(namespace)
}

func sarClusterRoleBindingName(namespace string) string {
	return "lightspeed-sar-binding-" + clusterRBACScopeID(namespace)
}

// AccessClusterRoleName returns the role name that telemetry derives from
// MetricStorage.spec.openStackLightspeedNamespace.
func AccessClusterRoleName(namespace string) string {
	return "lightspeed-access-" + clusterRBACScopeID(namespace)
}

func clusterRBACLabels(namespace string) map[string]string {
	return map[string]string{clusterRBACScopeLabel: clusterRBACScopeID(namespace)}
}
