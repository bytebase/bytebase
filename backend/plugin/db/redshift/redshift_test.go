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
		host, port, wantErr string
	}{
		{"redshift.example.com", "5439 host=other.example.com", "invalid port"},
		{"  ", "5439", "host must be set"},
	} {
		driver, err := newDriver().Open(context.Background(), storepb.Engine_REDSHIFT, db.ConnectionConfig{
			DataSource: &storepb.DataSource{Username: "dba", Host: tc.host, Port: tc.port},
		})
		if driver != nil {
			_ = driver.Close(context.Background())
		}
		require.ErrorContains(t, err, tc.wantErr, tc.port)
	}
}
