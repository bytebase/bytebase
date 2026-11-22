package redshift

import (
	"context"
	"testing"

	redshiftdriver "github.com/bytebase/omni/redshift/driver"
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

// TestRedshiftConfig maps a data source onto the contract's Config, which is the
// part of Open that does not reach a server.
func TestRedshiftConfig(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dataStore *storepb.DataSource
		ctx       db.ConnectionContext
		want      struct {
			database  string
			readOnly  bool
			datashare bool
			tls       bool
		}
		wantErr string
	}{
		{
			name:      "the context's database wins",
			dataStore: &storepb.DataSource{Username: "dba", Database: "from-source"},
			ctx:       db.ConnectionContext{DatabaseName: "from-context"},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{database: "from-context"},
		},
		{
			name:      "the data source's database is the next choice",
			dataStore: &storepb.DataSource{Username: "dba", Database: "from-source"},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{database: "from-source"},
		},
		{
			name:      "no database leaves the contract's default",
			dataStore: &storepb.DataSource{Username: "dba"},
		},
		{
			name:      "read only and datashare are carried",
			dataStore: &storepb.DataSource{Username: "dba"},
			ctx:       db.ConnectionContext{ReadOnly: true, DataShare: true},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{readOnly: true, datashare: true},
		},
		{
			name:      "use-ssl alone leaves libpq's negotiation alone",
			dataStore: &storepb.DataSource{Username: "dba", UseSsl: true},
		},
		{
			name:      "a user is required",
			dataStore: &storepb.DataSource{},
			wantErr:   "user must be set",
		},
		{
			name:      "a certificate requires its key",
			dataStore: &storepb.DataSource{Username: "dba", SslCert: "cert"},
			wantErr:   "ssl-cert and ssl-key must be both set or unset",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := redshiftConfig(db.ConnectionConfig{DataSource: tc.dataStore, ConnectionContext: tc.ctx})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want.database, cfg.Database)
			require.Equal(t, tc.want.readOnly, cfg.ReadOnly)
			require.Equal(t, tc.want.datashare, cfg.Params[redshiftdriver.ParamDatashare] == "true")
			require.Equal(t, tc.want.tls, cfg.TLS != nil)
		})
	}
}
