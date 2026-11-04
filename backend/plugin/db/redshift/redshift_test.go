package redshift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestOpenKeepsHostAndPortWhole(t *testing.T) {
	for _, tc := range []struct {
		name, host, port, wantErr string
	}{
		{"a port naming another host", "redshift.example.com", "5439 host=other.example.com", "invalid port"},
		{"a blank host", "  ", "5439", "host must be set"},
		{"a blank port", "redshift.example.com", "", "port must be set"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := newDriver().Open(context.Background(), storepb.Engine_REDSHIFT, db.ConnectionConfig{
				DataSource: &storepb.DataSource{Username: "dba", Host: tc.host, Port: tc.port},
			})
			if driver != nil {
				_ = driver.Close(context.Background())
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
