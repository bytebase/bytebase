package drivers

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

// TestOceanBaseConfig maps a data source onto the driver contract: the login, the
// database, the application name the /*app=*/ comment carries, and the extra
// parameters the MySQL family passes through.
func TestOceanBaseConfig(t *testing.T) {
	cfg, err := oceanbaseConfig(context.Background(), newDriver(oceanbaseSpec), db.ConnectionConfig{
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
	})
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

// TestOceanBaseConfigRefusesAllowAllFiles keeps the one data source parameter the
// MySQL family refuses, in any spelling.
func TestOceanBaseConfigRefusesAllowAllFiles(t *testing.T) {
	for _, key := range []string{"allowAllFiles", "allowallfiles", " AllowAllFiles "} {
		_, err := oceanbaseConfig(context.Background(), newDriver(oceanbaseSpec), db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username:                  "root@test",
				Host:                      "127.0.0.1",
				Port:                      "2881",
				ExtraConnectionParameters: map[string]string{key: "true"},
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
		})
		require.ErrorContains(t, err, "is not allowed for security reasons", key)
	}
}

// TestOceanBaseConfigTLS maps the data source's TLS material onto Config.TLS,
// which the driver requires once it is set.
func TestOceanBaseConfigTLS(t *testing.T) {
	cfg, err := oceanbaseConfig(context.Background(), newDriver(oceanbaseSpec), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
			UseSsl:   true,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlsConfig, err := cfg.TLS(cfg.Host)
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify, "verification is off unless the data source asks for it")

	// Without use_ssl the driver negotiates the server's default instead.
	cfg, err = oceanbaseConfig(context.Background(), newDriver(oceanbaseSpec), db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "root@test",
			Host:     "127.0.0.1",
			Port:     "2881",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "app"},
	})
	require.NoError(t, err)
	require.Nil(t, cfg.TLS)
}

// TestOceanBaseRDSTLSConfig builds the TLS config an RDS IAM connection always needs:
// an operator-supplied CA wins over the downloaded AWS bundle, and verification
// is the chain-and-hostname callback.
func TestOceanBaseRDSTLSConfig(t *testing.T) {
	dataSource := &storepb.DataSource{
		Username:             "root@test",
		Host:                 "rds.example.com",
		Port:                 "3306",
		VerifyTlsCertificate: true,
		SslCa:                testOceanBaseCACertPEM,
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
const testOceanBaseCACertPEM = `-----BEGIN CERTIFICATE-----
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

// TestOceanBaseExplainFormat maps the API's plan format onto the driver's.
func TestOceanBaseExplainFormat(t *testing.T) {
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
	d := &Driver{spec: oceanbaseSpec, OmniAdapter: util.OmniAdapter{Driver: impl, WithInstanceMetadata: true}}

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

// TestOceanBaseProbeRefusesPreExecutionsOutsideATransaction pins the contract's rule:
// Autocommit exists for a statement the engine refuses inside a transaction, so
// pre-executions, which need one, are refused rather than run beside it.
func TestOceanBaseProbeRefusesPreExecutionsOutsideATransaction(t *testing.T) {
	_, err := db.ProberOf(newDriver(oceanbaseSpec)).Probe(context.Background(), db.ProbeContext{
		Autocommit:    true,
		PreExecutions: []string{"SET NAMES utf8mb4"},
	}, "SELECT 1")
	require.ErrorContains(t, err, "cannot run outside a transaction")
}

type recordingOceanBaseProbeSession struct {
	statements   []string
	statementErr error
	closed       bool
}

func (s *recordingOceanBaseProbeSession) Query(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
	s.statements = append(s.statements, script)
	switch script {
	case "START TRANSACTION", "ROLLBACK":
		return []*odriver.Result{{}}, nil
	case "SET @probe_value = 7":
		return []*odriver.Result{{}}, nil
	default:
		if s.statementErr != nil {
			//nolint:nilerr // a refused statement's error rides on its result.
			return []*odriver.Result{{Err: s.statementErr}}, nil
		}
		return []*odriver.Result{{
			Columns: []odriver.Column{{Name: "Query Plan", TypeName: "VARCHAR"}},
			Rows:    [][]odriver.Value{{{Kind: odriver.String, String: `{"EST.ROWS":2}`}}},
		}}, nil
	}
}

func (s *recordingOceanBaseProbeSession) Close() error {
	s.closed = true
	return nil
}

type fakeOceanBaseProbeDriver struct {
	session      *recordingOceanBaseProbeSession
	affectedRows int64
	affectedErr  error
	columnExtra  string
	columnErr    error
	gotCountStmt string
	gotExtraArgs [3]string
}

func (*fakeOceanBaseProbeDriver) Ping(context.Context) error { return nil }
func (*fakeOceanBaseProbeDriver) Close() error               { return nil }
func (f *fakeOceanBaseProbeDriver) Session(context.Context) (odriver.Session, error) {
	return f.session, nil
}

func (f *fakeOceanBaseProbeDriver) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	f.gotCountStmt = statement
	return f.affectedRows, f.affectedErr
}

func (f *fakeOceanBaseProbeDriver) ColumnExtra(_ context.Context, database, table, column string) (string, error) {
	f.gotExtraArgs = [3]string{database, table, column}
	return f.columnExtra, f.columnErr
}

func TestOceanBaseProbeTransactionAndAutocommit(t *testing.T) {
	ctx := context.Background()

	t.Run("transactional probe wraps pre-executions and rolls back", func(t *testing.T) {
		sess := &recordingOceanBaseProbeSession{}
		d := &Driver{spec: oceanbaseSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeOceanBaseProbeDriver{session: sess}}}

		res, err := db.ProberOf(d).Probe(ctx, db.ProbeContext{
			PreExecutions: []string{"", "SET @probe_value = 7"},
		}, "EXPLAIN format=json UPDATE t SET name = 'x'")
		require.NoError(t, err)
		require.True(t, sess.closed)
		require.Equal(t, []string{
			"START TRANSACTION",
			"SET @probe_value = 7",
			"EXPLAIN format=json UPDATE t SET name = 'x'",
			"ROLLBACK",
		}, sess.statements)
		require.Equal(t, []string{"Query Plan"}, res.Columns)
		plan, err := res.Plan()
		require.NoError(t, err)
		require.Equal(t, `{"EST.ROWS":2}`, plan)
	})

	t.Run("failed probe statement still rolls back", func(t *testing.T) {
		boom := errors.New("table doesn't exist")
		sess := &recordingOceanBaseProbeSession{statementErr: boom}
		d := &Driver{spec: oceanbaseSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeOceanBaseProbeDriver{session: sess}}}

		_, err := db.ProberOf(d).Probe(ctx, db.ProbeContext{}, "EXPLAIN UPDATE missing SET a = 1")
		require.ErrorIs(t, err, boom)
		require.Equal(t, []string{
			"START TRANSACTION",
			"EXPLAIN UPDATE missing SET a = 1",
			"ROLLBACK",
		}, sess.statements)
	})

	t.Run("autocommit runs the statement alone", func(t *testing.T) {
		sess := &recordingOceanBaseProbeSession{}
		d := &Driver{spec: oceanbaseSpec, OmniAdapter: util.OmniAdapter{Driver: &fakeOceanBaseProbeDriver{session: sess}}}

		_, err := db.ProberOf(d).Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN SELECT 1")
		require.NoError(t, err)
		require.Equal(t, []string{"EXPLAIN SELECT 1"}, sess.statements)
	})
}

func TestOceanBaseProbeCell(t *testing.T) {
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123000000, time.UTC)
	require.Equal(t, "raw", oceanbaseProbeCell(odriver.Value{Kind: odriver.Bytes, Bytes: []byte("raw")}))
	require.Equal(t, int32(7), oceanbaseProbeCell(odriver.Value{Kind: odriver.Int32, Int64: 7}))
	require.Equal(t, uint32(9), oceanbaseProbeCell(odriver.Value{Kind: odriver.Uint32, Uint64: 9}))
	require.Nil(t, oceanbaseProbeCell(odriver.Value{Kind: odriver.Structured, Structured: map[string]any{"a": 1}}))
	require.Equal(t, ts.Format(time.RFC3339Nano), oceanbaseProbeCell(odriver.Value{Kind: odriver.Timestamp, Time: ts}))
	require.Equal(t, int64(42), oceanbaseProbeCell(odriver.Value{Kind: odriver.Int64, Int64: 42}))
}

func TestOceanBaseRowEstimatorAndColumnExtraDelegation(t *testing.T) {
	ctx := context.Background()
	impl := &fakeOceanBaseProbeDriver{
		affectedRows: 2,
		columnExtra:  "STORED GENERATED",
	}
	d := &Driver{spec: oceanbaseSpec, OmniAdapter: util.OmniAdapter{Driver: impl}}

	estimator, ok := db.As[db.RowEstimator](d)
	require.True(t, ok)
	rows, err := estimator.CountAffectedRows(ctx, "UPDATE t SET name = 'x'")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, "UPDATE t SET name = 'x'", impl.gotCountStmt)

	lookup, ok := db.As[db.ColumnExtraLookup](d)
	require.True(t, ok)
	extra, err := lookup.ColumnExtra(ctx, "app", "t", "generated_col")
	require.NoError(t, err)
	require.Equal(t, "STORED GENERATED", extra)
	require.Equal(t, [3]string{"app", "t", "generated_col"}, impl.gotExtraArgs)
}
