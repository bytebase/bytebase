//nolint:revive
package util

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// ResolveDatabaseName returns the connection context database name when set,
// falling back to the data source's configured database.
func ResolveDatabaseName(config db.ConnectionConfig) string {
	if config.ConnectionContext.DatabaseName != "" {
		return config.ConnectionContext.DatabaseName
	}
	return config.DataSource.GetDatabase()
}

// ApplicationName returns the application_name for a connection:
// "bytebase-taskrun-<uid>" for task runs and "bytebase" otherwise.
func ApplicationName(connectionCtx db.ConnectionContext) string {
	if connectionCtx.TaskRunUID != nil {
		return fmt.Sprintf("bytebase-taskrun-%d", *connectionCtx.TaskRunUID)
	}
	return "bytebase"
}

// CloneParams returns a shallow copy of extra connection parameters, or an
// empty non-nil map when extra is empty.
func CloneParams(extra map[string]string) map[string]string {
	params := make(map[string]string, len(extra))
	for key, value := range extra {
		params[key] = value
	}
	return params
}

// PositiveMaxResultBytes clamps maxBytes <= 0 to 1 so drivers that treat <= 0
// as unbounded still enforce Bytebase's zero-byte cap ("no rows").
func PositiveMaxResultBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return 1
	}
	return maxBytes
}

// ValidateTLSCertKeyPair verifies that client certificate and key are either
// both set or both unset.
func ValidateTLSCertKeyPair(ds *storepb.DataSource) error {
	if (ds.GetSslCert() == "") != (ds.GetSslKey() == "") {
		return errors.New("ssl-cert and ssl-key must be both set or unset")
	}
	return nil
}

// StaticTLSFunc resolves TLS material eagerly and returns a constant callback
// for driver.Config.TLS when SSL is enabled, or nil when SSL is disabled.
func StaticTLSFunc(ds *storepb.DataSource) (func(string) (*tls.Config, error), error) {
	tlsConfig, err := GetTLSConfig(ds)
	if err != nil {
		return nil, err
	}
	if tlsConfig == nil {
		return nil, nil
	}
	return func(string) (*tls.Config, error) {
		return tlsConfig, nil
	}, nil
}

// StaticHostTLSFunc resolves TLS material eagerly via GetTLSConfig and returns
// a driver.Config.TLS callback that clones the config with ServerName set to
// the dialed host, or nil when SSL is disabled.
func StaticHostTLSFunc(ds *storepb.DataSource) (func(string) (*tls.Config, error), error) {
	tlsConfig, err := GetTLSConfig(ds)
	if err != nil || tlsConfig == nil {
		return nil, err
	}
	return func(host string) (*tls.Config, error) {
		byHost := tlsConfig.Clone()
		byHost.ServerName = host
		return byHost, nil
	}, nil
}

// LazyTLSFunc returns a driver.Config.TLS callback that builds TLS material
// via GetTLSConfig on dial when SSL is enabled, or nil when SSL is disabled.
func LazyTLSFunc(ds *storepb.DataSource) func(string) (*tls.Config, error) {
	if !ds.GetUseSsl() {
		return nil
	}
	return func(string) (*tls.Config, error) {
		return GetTLSConfig(ds)
	}
}

// HostTLSFunc returns a driver.Config.TLS callback that binds SNI and
// certificate hostname verification to each dialed host when SSL is enabled.
func HostTLSFunc(ds *storepb.DataSource) func(string) (*tls.Config, error) {
	if !ds.GetUseSsl() {
		return nil
	}
	return func(host string) (*tls.Config, error) {
		return getTLSConfigForHost(ds, host)
	}
}

// SQLStaticTLSFunc resolves TLS material eagerly via StaticTLSFunc and wraps
// any error with the "sql: tls config error" prefix expected by SQL adapters.
func SQLStaticTLSFunc(ds *storepb.DataSource) (func(string) (*tls.Config, error), error) {
	tlsFn, err := StaticTLSFunc(ds)
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	return tlsFn, nil
}

// PGTLSFunc returns a driver.Config.TLS callback that applies PostgreSQL SNI
// and per-host certificate verification when SSL is enabled, or nil otherwise.
func PGTLSFunc(ds *storepb.DataSource) func(string) (*tls.Config, error) {
	if !ds.GetUseSsl() {
		return nil
	}
	return func(host string) (*tls.Config, error) {
		return getPGTLSConfigForHost(ds, host)
	}
}

// MySQLFamilyDriverConfig builds an omni/driver.Config for MySQL-protocol
// drivers that use SSH tunneling and eagerly resolved TLS material (Doris and
// StarRocks).
func MySQLFamilyDriverConfig(config db.ConnectionConfig, adapter *OmniAdapter) (omnidriver.Config, error) {
	ds := config.DataSource
	cfg := omnidriver.Config{
		Host:            ds.GetHost(),
		Port:            ds.GetPort(),
		Database:        config.ConnectionContext.DatabaseName,
		User:            ds.GetUsername(),
		Password:        config.Password,
		ApplicationName: "bytebase",
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	if err := SetupSSHDial(ds, &cfg, adapter, false); err != nil {
		return cfg, err
	}
	tlsFn, err := SQLStaticTLSFunc(ds)
	if err != nil {
		return cfg, err
	}
	cfg.TLS = tlsFn
	return cfg, nil
}

// SetupSSHDial opens an SSH tunnel when ds.GetSshHost() is non-empty, registers
// the ssh.Client on adapter for Close, and sets cfg.Dial. When deadlineAware is
// true, dials use dialSSHTunnel (required by drivers like pgx that interrupt
// blocked reads via deadlines); otherwise dials use sshClient.Dial directly.
func SetupSSHDial(ds *storepb.DataSource, cfg *omnidriver.Config, adapter *OmniAdapter, deadlineAware bool) error {
	if ds.GetSshHost() == "" {
		return nil
	}
	sshClient, err := GetSSHClient(ds)
	if err != nil {
		return err
	}
	adapter.addCloser(sshClient)
	if deadlineAware {
		cfg.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialSSHTunnel(ctx, sshClient, network, address)
		}
	} else {
		cfg.Dial = func(_ context.Context, network, address string) (net.Conn, error) {
			return sshClient.Dial(network, address)
		}
	}
	return nil
}

// SetupSSHDialWithWrap opens an SSH tunnel like SetupSSHDial and wraps each
// dialed connection with wrap when non-nil (for clients requiring custom
// RemoteAddr or no-op deadlines).
func SetupSSHDialWithWrap(
	ds *storepb.DataSource,
	cfg *omnidriver.Config,
	adapter *OmniAdapter,
	deadlineAware bool,
	wrap func(conn net.Conn, address string) net.Conn,
) error {
	if err := SetupSSHDial(ds, cfg, adapter, deadlineAware); err != nil {
		return err
	}
	if cfg.Dial != nil && wrap != nil {
		baseDial := cfg.Dial
		cfg.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := baseDial(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return wrap(conn, address), nil
		}
	}
	return nil
}

// SetupCloudSQLDial creates a Cloud SQL IAM dialer, registers it on adapter
// for Close, sets cfg.Dial to dial config.DataSource.GetHost(), and returns the
// dialer.
func SetupCloudSQLDial(ctx context.Context, config db.ConnectionConfig, cfg *omnidriver.Config, adapter *OmniAdapter) (*cloudsqlconn.Dialer, error) {
	dialer, err := getGCPConnectionConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "load gcp config failed")
	}
	adapter.addCloser(dialer)
	instance := config.DataSource.GetHost()
	cfg.Dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, instance)
	}
	return dialer, nil
}

// ConfigureRDSMySQLIAM resolves an AWS RDS IAM token and TLS config and applies
// them along with allowCleartextPasswords=true to cfg.
func ConfigureRDSMySQLIAM(ctx context.Context, config db.ConnectionConfig, cfg *omnidriver.Config) error {
	token, err := RDSAuthToken(ctx, config)
	if err != nil {
		return err
	}
	tlsConfig, err := RDSTLSConfig(ctx, config.DataSource)
	if err != nil {
		return err
	}
	cfg.Password = token
	cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["allowCleartextPasswords"] = "true"
	return nil
}
