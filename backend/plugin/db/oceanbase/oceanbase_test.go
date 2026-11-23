package oceanbase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestDriverConfig maps a data source onto the driver contract: the login, the
// database, the application name the /*app=*/ comment carries, and the extra
// parameters the MySQL family passes through.
func TestDriverConfig(t *testing.T) {
	cfg, err := driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:                  "root@test",
			Host:                      "127.0.0.1",
			Port:                      "2881",
			ExtraConnectionParameters: map[string]string{"charset": "utf8mb4", "loc": "UTC"},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "app",
			EngineVersion: "4.3.5",
		},
	}, &Driver{})
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.Host)
	require.Equal(t, "2881", cfg.Port)
	require.Equal(t, "app", cfg.Database)
	require.Equal(t, "root@test", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, "bytebase", cfg.ApplicationName)
	require.Equal(t, "4.3.5", cfg.EngineVersion)
	require.Equal(t, map[string]string{"charset": "utf8mb4", "loc": "UTC"}, cfg.Params)
	require.Nil(t, cfg.Dial)
	require.Nil(t, cfg.TLS)
}

// TestDriverConfigRefusesAllowAllFiles keeps the one data source parameter the
// MySQL family refuses, in any spelling.
func TestDriverConfigRefusesAllowAllFiles(t *testing.T) {
	for _, key := range []string{"allowAllFiles", "allowallfiles", " AllowAllFiles "} {
		_, err := driverConfig(context.Background(), db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username:                  "root@test",
				Host:                      "127.0.0.1",
				Port:                      "2881",
				ExtraConnectionParameters: map[string]string{key: "true"},
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
		}, &Driver{})
		require.ErrorContains(t, err, "is not allowed for security reasons", key)
	}
}

// TestDriverConfigTLS maps the data source's TLS material onto Config.TLS,
// which the driver requires once it is set.
func TestDriverConfigTLS(t *testing.T) {
	cfg, err := driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
			UseSsl:   true,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	}, &Driver{})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlsConfig, err := cfg.TLS(cfg.Host)
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify, "verification is off unless the data source asks for it")

	// Without use_ssl the driver negotiates the server's default instead.
	cfg, err = driverConfig(context.Background(), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	}, &Driver{})
	require.NoError(t, err)
	require.Nil(t, cfg.TLS)
}

// TestRDSTLSConfig builds the TLS config an RDS IAM connection always needs:
// an operator-supplied CA wins over the downloaded AWS bundle, and verification
// is the chain-and-hostname callback.
func TestRDSTLSConfig(t *testing.T) {
	dataSource := &storepb.DataSource{
		Username:             "root@test",
		Host:                 "rds.example.com",
		Port:                 "3306",
		VerifyTlsCertificate: true,
		SslCa:                testCACertPEM,
	}
	tlsConfig, err := util.RDSTLSConfig(context.Background(), dataSource)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig.RootCAs)
	require.NotNil(t, tlsConfig.VerifyPeerCertificate)

	// Verification off still encrypts, and consults no root pool.
	dataSource.VerifyTlsCertificate = false
	tlsConfig, err = util.RDSTLSConfig(context.Background(), dataSource)
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.RootCAs)
	require.Nil(t, tlsConfig.VerifyPeerCertificate)

	// A CA the operator supplied but the server cannot parse is an error rather
	// than a silent fallback to the downloaded bundle.
	dataSource.VerifyTlsCertificate = true
	dataSource.SslCa = "not a certificate"
	_, err = util.RDSRootCAs(context.Background(), dataSource)
	require.ErrorContains(t, err, "failed to parse ssl_ca certificates")
}

// A self-signed certificate's PEM is enough to prove the CA is preferred over
// the downloaded bundle: nothing here verifies a server against it.
const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

// TestExplainFormat maps the API's plan format onto the driver's.
func TestExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", util.OmniExplainFormat(v1pb.QueryOption_JSON, "", v1pb.QueryOption_JSON))
	require.Equal(t, "", util.OmniExplainFormat(v1pb.QueryOption_TEXT, "", v1pb.QueryOption_JSON))
	require.Equal(t, "", util.OmniExplainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "", v1pb.QueryOption_JSON))
}

type fakeOceanBaseSession struct {
	gotOpts odriver.QueryOptions
	results []*odriver.Result
	err     error
	closed  bool
}

func (s *fakeOceanBaseSession) Query(_ context.Context, _ string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotOpts = opts
	return s.results, s.err
}

func (s *fakeOceanBaseSession) Close() error {
	s.closed = true
	return nil
}

type fakeOceanBaseDriver struct {
	sessions      []*fakeOceanBaseSession
	sessionCalls  int
	executeScript string
	executeOpts   odriver.ExecuteOptions
	executeFn     func(script string, opts odriver.ExecuteOptions) (int64, error)
	instance      *odriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
	dumpDDL       string
}

func (*fakeOceanBaseDriver) Ping(context.Context) error { return nil }
func (*fakeOceanBaseDriver) Close() error               { return nil }

func (f *fakeOceanBaseDriver) Session(context.Context) (odriver.Session, error) {
	idx := f.sessionCalls
	f.sessionCalls++
	if idx < len(f.sessions) {
		return f.sessions[idx], nil
	}
	return &fakeOceanBaseSession{}, nil
}

func (f *fakeOceanBaseDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeOceanBaseDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeOceanBaseDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (f *fakeOceanBaseDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func TestOceanBaseAdapterSeam(t *testing.T) {
	ctx := context.Background()
	s1 := &fakeOceanBaseSession{
		results: []*odriver.Result{{
			Statement: "SELECT 1",
			Columns:   []odriver.Column{{Name: "1", TypeName: "BIGINT"}},
			Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
			Plan:      &odriver.Plan{Format: "JSON"},
		}},
	}
	s2 := &fakeOceanBaseSession{err: errors.New("broken session")}
	var logs []*storepb.TaskRunLog
	impl := &fakeOceanBaseDriver{
		sessions: []*fakeOceanBaseSession{s1, s2},
		executeFn: func(script string, opts odriver.ExecuteOptions) (int64, error) {
			opts.Log.TransactionControl(odriver.Begin, nil)
			opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(2, []int64{2}, nil)
			opts.Log.TransactionControl(odriver.Commit, nil)
			return 2, nil
		},
		instance: &odriver.Instance{
			Version:   "4.3.5",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "app"}},
		},
		schema:  &metadatapb.DatabaseSchemaMetadata{Name: "app"},
		dumpDDL: "CREATE TABLE `t` (`id` int);\n",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl, WithInstanceMetadata: true}}

	// Held session is reused across QueryConn calls and forwards options + masking.
	res, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{
		Limit:                10,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Nil(t, res[0].QueryPlan, "explain requests leave QueryPlan nil for the API layer")
	require.Equal(t, odriver.QueryOptions{
		Limit:          10,
		Explain:        true,
		ExplainFormat:  "JSON",
		MaxResultBytes: 1024,
		MaskingEnabled: true,
	}, s1.gotOpts)
	require.Same(t, s1, d.Session())

	_, err = d.QueryConn(ctx, "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.Equal(t, 1, impl.sessionCalls)

	// Canceled context discards the held session.
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = d.QueryConn(canceledCtx, "SELECT 1", db.QueryContext{})
	require.NoError(t, err)
	require.True(t, s1.closed)
	require.Nil(t, d.Session())

	// Session error discards the session.
	_, err = d.QueryConn(ctx, "SELECT 1", db.QueryContext{})
	require.ErrorContains(t, err, "broken session")
	require.True(t, s2.closed)
	require.Nil(t, d.Session())

	// Execute parses transaction directive and isolation level and logs steps.
	affected, err := d.Execute(ctx, "-- txn-mode=on\n-- txn-isolation=read committed\nINSERT INTO t VALUES (1);", db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)
	require.Equal(t, "INSERT INTO t VALUES (1);", impl.executeScript)
	require.Equal(t, odriver.Transactional, impl.executeOpts.Transaction)
	require.Equal(t, odriver.ReadCommitted, impl.executeOpts.Isolation)
	require.Len(t, logs, 4)

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "4.3.5", inst.Version)

	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "app", schema.GetName())

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, schema))
	require.Contains(t, out.String(), "CREATE TABLE `t`")
}
