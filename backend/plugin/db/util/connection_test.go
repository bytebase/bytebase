//nolint:revive
package util

import (
	"strings"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGCPCredentialOption(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name: "service_account",
			json: `{"type": "service_account", "project_id": "test"}`,
		},
		{
			name: "external_account",
			json: `{"type": "external_account", "audience": "test"}`,
		},
		{
			name: "impersonated_service_account",
			json: `{"type": "impersonated_service_account"}`,
		},
		{
			name: "authorized_user",
			json: `{"type": "authorized_user"}`,
		},
		{
			name:    "invalid JSON",
			json:    `not json`,
			wantErr: "failed to parse GCP credential JSON",
		},
		{
			name:    "missing type field",
			json:    `{"project_id": "test"}`,
			wantErr: `missing "type" field`,
		},
		{
			name:    "unsupported type",
			json:    `{"type": "unknown_type"}`,
			wantErr: `unsupported GCP credential type: "unknown_type"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GCPCredentialOption([]byte(tc.json))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil ClientOption")
			}
		})
	}
}

func TestCloudSQLDialOptions(t *testing.T) {
	tests := []struct {
		name    string
		ipType  storepb.DataSource_CloudSQLIPType
		wantLen int
	}{
		{"unspecified defaults to public", storepb.DataSource_CLOUD_SQL_IP_TYPE_UNSPECIFIED, 0},
		{"public", storepb.DataSource_PUBLIC, 0},
		{"private", storepb.DataSource_PRIVATE, 1},
		{"psc", storepb.DataSource_PSC, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cloudSQLDialOptions(&storepb.DataSource{CloudSqlIpType: tc.ipType}); len(got) != tc.wantLen {
				t.Errorf("cloudSQLDialOptions(%v) = %d options, want %d", tc.ipType, len(got), tc.wantLen)
			}
		})
	}
}
