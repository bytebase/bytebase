//nolint:revive
package util

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestConfigHelpers(t *testing.T) {
	require.Equal(t, "ctx_db", ResolveDatabaseName(db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Database: "ds_db"},
		ConnectionContext: db.ConnectionContext{DatabaseName: "ctx_db"},
	}))
	require.Equal(t, "ds_db", ResolveDatabaseName(db.ConnectionConfig{
		DataSource: &storepb.DataSource{Database: "ds_db"},
	}))

	uid := int64(42)
	require.Equal(t, "bytebase", ApplicationName(db.ConnectionContext{}))
	require.Equal(t, "bytebase-taskrun-42", ApplicationName(db.ConnectionContext{TaskRunUID: &uid}))

	orig := map[string]string{"a": "1"}
	cloned := CloneParams(orig)
	cloned["b"] = "2"
	require.NotContains(t, orig, "b")

	require.Equal(t, int64(1), PositiveMaxResultBytes(0))
	require.Equal(t, int64(1), PositiveMaxResultBytes(-5))
	require.Equal(t, int64(512), PositiveMaxResultBytes(512))

	require.NoError(t, ValidateTLSCertKeyPair(&storepb.DataSource{}))
	require.NoError(t, ValidateTLSCertKeyPair(&storepb.DataSource{SslCert: "c", SslKey: "k"}))
	require.EqualError(t, ValidateTLSCertKeyPair(&storepb.DataSource{SslCert: "c"}), "ssl-cert and ssl-key must be both set or unset")
	require.EqualError(t, ValidateTLSCertKeyPair(&storepb.DataSource{SslKey: "k"}), "ssl-cert and ssl-key must be both set or unset")

	fn, err := StaticTLSFunc(&storepb.DataSource{})
	require.NoError(t, err)
	require.Nil(t, fn)

	_, err = StaticTLSFunc(&storepb.DataSource{UseSsl: true, SslCert: "c"})
	require.Error(t, err)

	require.Nil(t, LazyTLSFunc(&storepb.DataSource{}))
	require.NotNil(t, LazyTLSFunc(&storepb.DataSource{UseSsl: true}))

	require.Nil(t, HostTLSFunc(&storepb.DataSource{}))
	hostFn := HostTLSFunc(&storepb.DataSource{UseSsl: true})
	require.NotNil(t, hostFn)
	cfg, err := hostFn("db.example.com")
	require.NoError(t, err)
	require.Equal(t, "db.example.com", cfg.ServerName)
}
