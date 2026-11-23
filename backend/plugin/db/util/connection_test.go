//nolint:revive
package util

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

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
