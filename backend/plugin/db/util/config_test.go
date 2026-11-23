//nolint:revive
package util

import (
	"context"
	"net"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
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

	sqlTLSFn, err := SQLStaticTLSFunc(&storepb.DataSource{})
	require.NoError(t, err)
	require.Nil(t, sqlTLSFn)

	_, err = SQLStaticTLSFunc(&storepb.DataSource{UseSsl: true, SslCert: "c"})
	require.ErrorContains(t, err, "sql: tls config error")

	require.Nil(t, PGTLSFunc(&storepb.DataSource{}))
	pgTLSFn := PGTLSFunc(&storepb.DataSource{UseSsl: true})
	require.NotNil(t, pgTLSFn)
	pgTLSCfg, err := pgTLSFn("pg.example.com")
	require.NoError(t, err)
	require.Equal(t, "pg.example.com", pgTLSCfg.ServerName)

	var adapter OmniAdapter
	mysqlCfg, err := MySQLFamilyDriverConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "9030",
			Username: "root",
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "analytics",
			EngineVersion: "3.0.0",
		},
	}, &adapter)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", mysqlCfg.Host)
	require.Equal(t, "9030", mysqlCfg.Port)
	require.Equal(t, "analytics", mysqlCfg.Database)
	require.Equal(t, "root", mysqlCfg.User)
	require.Equal(t, "secret", mysqlCfg.Password)
	require.Equal(t, "bytebase", mysqlCfg.ApplicationName)
	require.Equal(t, "3.0.0", mysqlCfg.EngineVersion)
	require.Nil(t, mysqlCfg.TLS)

	_, err = MySQLFamilyDriverConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{UseSsl: true, SslCert: "c"},
	}, &adapter)
	require.ErrorContains(t, err, "sql: tls config error")
}

func TestStaticHostTLSFunc(t *testing.T) {
	fn, err := StaticHostTLSFunc(&storepb.DataSource{})
	require.NoError(t, err)
	require.Nil(t, fn)

	_, err = StaticHostTLSFunc(&storepb.DataSource{UseSsl: true, SslCert: "c"})
	require.Error(t, err)

	fn, err = StaticHostTLSFunc(&storepb.DataSource{Host: "primary.example.com", UseSsl: true})
	require.NoError(t, err)
	require.NotNil(t, fn)

	cfgA, err := fn("node-a.example.com")
	require.NoError(t, err)
	cfgB, err := fn("node-b.example.com")
	require.NoError(t, err)
	require.Equal(t, "node-a.example.com", cfgA.ServerName)
	require.Equal(t, "node-b.example.com", cfgB.ServerName)
	require.NotSame(t, cfgA, cfgB)
}

func TestSetupSSHDialWithWrap(t *testing.T) {
	var adapter OmniAdapter
	var cfg omnidriver.Config
	require.NoError(t, SetupSSHDialWithWrap(&storepb.DataSource{}, &cfg, &adapter, true, nil))
	require.Nil(t, cfg.Dial)

	err := SetupSSHDialWithWrap(&storepb.DataSource{SshHost: "127.0.0.1", SshPort: "1"}, &cfg, &adapter, true, nil)
	require.ErrorContains(t, err, "127.0.0.1:1")
	require.Nil(t, cfg.Dial)

	sshServer := testcontainer.NewSSHServer(t)
	ds := &storepb.DataSource{
		SshHost:       sshServer.GetHost(),
		SshPort:       sshServer.GetPort(),
		SshUser:       sshServer.GetUser(),
		SshPrivateKey: sshServer.GetPrivateKey(),
	}
	targetAddr := net.JoinHostPort(testcontainer.NewTCPServer(t, func(conn net.Conn) {
		_ = conn.Close()
	}))

	var wrappedAddr string
	require.NoError(t, SetupSSHDialWithWrap(ds, &cfg, &adapter, false, func(conn net.Conn, address string) net.Conn {
		wrappedAddr = address
		return &NoDeadlineConn{Conn: conn}
	}))
	t.Cleanup(func() { _ = adapter.Close(context.Background()) })
	require.NotNil(t, cfg.Dial)

	conn, err := cfg.Dial(context.Background(), "tcp", targetAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, ok := conn.(*NoDeadlineConn)
	require.True(t, ok)
	require.Equal(t, targetAddr, wrappedAddr)
}
