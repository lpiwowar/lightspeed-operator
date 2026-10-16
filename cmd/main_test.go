package main

import (
	"os"
	"strings"
	"testing"
)

func TestGetWatchNamespaces(t *testing.T) {
	t.Setenv("WATCH_NAMESPACE", "legacy-value")
	for _, tt := range []struct {
		name    string
		value   *string
		want    []string
		wantErr string
	}{
		{name: "missing", wantErr: "WATCH_NAMESPACES must be set"},
		{name: "all namespaces", value: ptr(""), want: nil},
		{name: "single namespace", value: ptr("target"), want: []string{"target"}},
		{name: "multiple namespaces", value: ptr("one,two"), wantErr: "MultiNamespace installations are unsupported"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WATCH_NAMESPACES", "")
			if tt.value == nil {
				t.Setenv("WATCH_NAMESPACES", "")
				// Setenv cannot make a variable absent, so exercise the missing case separately.
				if err := unsetenvForTest("WATCH_NAMESPACES"); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Setenv("WATCH_NAMESPACES", *tt.value)
			}
			got, err := getWatchNamespaces()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("got %v, %v; want %v, nil", got, err, tt.want)
			}
		})
	}
}

func ptr(value string) *string { return &value }

func unsetenvForTest(key string) error { return os.Unsetenv(key) }
