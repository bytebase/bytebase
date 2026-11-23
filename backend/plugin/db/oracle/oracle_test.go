package oracle

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"maps"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func TestParseVersion(t *testing.T) {
	type testData struct {
		Banner string
		First  int
		Second int
	}
	tests := []testData{
		{
			Banner: "12.1.0.2.0",
			First:  12,
			Second: 1,
		},
		{
			Banner: "12.1.0.",
			First:  12,
			Second: 1,
		},
	}

	for _, test := range tests {
		v, err := plsql.ParseVersion(test.Banner)
		require.NoError(t, err)
		require.Equal(t, test.First, v.First)
		require.Equal(t, test.Second, v.Second)
	}
}

func TestOracleSplitKeepsTrailingFragmentForDatabaseExecution(t *testing.T) {
	commands, err := plsql.SplitSQL("DROP TABLESPACE xxx; CASCADE")
	require.NoError(t, err)
	commands = base.FilterEmptyStatements(commands)
	require.Len(t, commands, 2)
	require.Equal(t, "DROP TABLESPACE xxx", commands[0].Text)
	require.Equal(t, " CASCADE", commands[1].Text)
}

func TestOracleSplitAllowsParserUnsupportedDDL(t *testing.T) {
	statement := `SET DEFINE OFF
CREATE VECTOR INDEX vec_idx ON docs (embedding);
CREATE JSON RELATIONAL DUALITY VIEW emp_dv AS SELECT employee_id FROM employees;
CREATE INDEX IDX_SALES_MONTH_YEAR ON SALES_DATA(EXTRACT(YEAR FROM SALE_DATE), EXTRACT(MONTH FROM SALE_DATE));
CREATE SEQUENCE order_seq START WITH 1 INCREMENT BY 1;
CREATE TABLE employees (salary NUMBER CHECK (salary > 0 OR salary IS NULL));
CREATE PACKAGE pkg IS
  PROCEDURE p;
END pkg;
CREATE PACKAGE BODY pkg IS
  PROCEDURE p IS
  BEGIN
    NULL;
  END p;
END pkg;
CREATE FUNCTION calc_bonus(p_start_date DATE)
RETURN DATE
IS
  v_current_date DATE := p_start_date;
BEGIN
  RETURN v_current_date;
END calc_bonus;
CREATE PROCEDURE update_salary(p_employee_id NUMBER)
IS
BEGIN
  UPDATE employees SET salary = salary + 1 WHERE id = p_employee_id;
END update_salary;`
	commands, err := plsql.SplitSQL(statement)
	require.NoError(t, err)
	commands = base.FilterEmptyStatements(commands)
	require.Len(t, commands, 9)
}

// TestConnectionParams pins the data source fields that reach go-ora through the
// contract's Params, and that an extra parameter wins in any spelling: go-ora
// upper-cases option keys, so two spellings of one option would leave the winner
// to map iteration order.
func TestConnectionParams(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		sid             string
		extraParameters map[string]string
		want            map[string]string
	}{
		{name: "no fields", want: map[string]string{}},
		{name: "service name field", serviceName: "ORCLPDB1", want: map[string]string{"SERVICE_NAME": "ORCLPDB1"}},
		{name: "SID field", sid: "ORCL", want: map[string]string{"SID": "ORCL"}},
		{
			name:            "extra parameters in their own spelling",
			serviceName:     "ORCLPDB1",
			extraParameters: map[string]string{"WALLET": "/opt/oracle/wallet", "prefetch_rows": "500"},
			want:            map[string]string{"SERVICE_NAME": "ORCLPDB1", "WALLET": "/opt/oracle/wallet", "prefetch_rows": "500"},
		},
		{
			name:            "an extra SID replaces the field",
			sid:             "ORCL",
			want:            map[string]string{"sid": "STANDBY"},
			extraParameters: map[string]string{"sid": "STANDBY"},
		},
		{
			name:            "an extra service name replaces the field",
			serviceName:     "ORCLPDB1",
			extraParameters: map[string]string{"service_name": "OTHER"},
			want:            map[string]string{"service_name": "OTHER"},
		},
		{
			name:            "an extra SERVICE_NAME alone names the service",
			extraParameters: map[string]string{"SERVICE_NAME": "OTHER"},
			want:            map[string]string{"SERVICE_NAME": "OTHER"},
		},
		{
			name:            "an explicit dial timeout is kept",
			extraParameters: map[string]string{"CONNECT TIMEOUT": "5"},
			want:            map[string]string{"CONNECT TIMEOUT": "5"},
		},
		{
			name:            "another spelling of the dial timeout is kept",
			extraParameters: map[string]string{"connection timeout": "5"},
			want:            map[string]string{"connection timeout": "5"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extraParameters := maps.Clone(test.extraParameters)
			dataSource := &storepb.DataSource{
				ServiceName:               test.serviceName,
				Sid:                       test.sid,
				ExtraConnectionParameters: extraParameters,
			}
			// The dial timeout is the one parameter the adapter supplies: go-ora
			// otherwise leaves it to the transport, which stalls a connect check
			// behind an unreachable primary.
			want := maps.Clone(test.want)
			hasDialTimeout := false
			for key := range want {
				option := strings.ToUpper(key)
				if option == "CONNECTION TIMEOUT" || option == "CONNECT TIMEOUT" {
					hasDialTimeout = true
				}
			}
			if !hasDialTimeout {
				want["CONNECTION TIMEOUT"] = "10"
			}
			require.Equal(t, want, connectionParams(dataSource))
			require.Equal(t, test.extraParameters, dataSource.GetExtraConnectionParameters(), "the data source must not be changed")
		})
	}
}

// TestQueryOptionsCarriesTheRequest pins the mapping QueryConn hands the driver,
// masking included: the rows of a statement whose span cannot be traced must not
// reach the caller.
func TestQueryOptionsCarriesTheRequest(t *testing.T) {
	options := queryOptions(db.QueryContext{
		Schema:               "APP",
		Limit:                10,
		Explain:              true,
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.Equal(t, "APP", options.Schema)
	require.Equal(t, 10, options.Limit)
	require.True(t, options.Explain)
	require.Equal(t, "JSON", options.ExplainFormat)
	require.Equal(t, int64(1024), options.MaxResultBytes)
	require.True(t, options.MaskingEnabled, "masking reaches the driver")
}

func TestOpenAppliesTLSSettings(t *testing.T) {
	pki := newTestPKI(t)
	serverConfig := &tls.Config{Certificates: []tls.Certificate{pki.server}}
	mutualConfig := &tls.Config{
		Certificates: []tls.Certificate{pki.server},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pki.caPool,
	}
	verified := func(port string) *storepb.DataSource {
		dataSource := tlsTestDataSource(port)
		dataSource.UseSsl = true
		dataSource.VerifyTlsCertificate = true
		dataSource.SslCa = pki.caPEM
		return dataSource
	}

	t.Run("TLS mode Disabled connects without TLS", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		ping(t, tlsTestDataSource(port))
		require.ErrorContains(t, nextHandshake(t, handshakes).err, "first record does not look like a TLS handshake")
	})

	t.Run("TLS mode TLS without verification", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := tlsTestDataSource(port)
		dataSource.UseSsl = true
		ping(t, dataSource)
		require.NoError(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Verify accepts the signing CA through a name in the certificate", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		ping(t, verified(port))
		require.NoError(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Verify rejects a name the certificate doesn't list", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := verified(port)
		dataSource.Host = "127.0.0.1"
		require.Contains(t, ping(t, dataSource), "x509: cannot validate certificate for 127.0.0.1")
		require.Error(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Verify rejects an unrelated CA", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := verified(port)
		dataSource.SslCa = newTestPKI(t).caPEM
		require.Contains(t, ping(t, dataSource), "x509: certificate signed by unknown authority")
		require.Error(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Verify with system trust rejects a private CA", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := verified(port)
		dataSource.SslCa = ""
		// The message comes from the platform's verifier, so it differs between macOS and Linux.
		require.Contains(t, ping(t, dataSource), "x509:")
		require.Error(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Verify checks a SERVER address against its own name", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		// Hostname refuses connections, and the certificate doesn't name it.
		dataSource := verified(closedPort(t))
		dataSource.Host = "127.0.0.1"
		dataSource.ExtraConnectionParameters = map[string]string{"SERVER": "localhost:" + port}
		ping(t, dataSource)
		require.NoError(t, nextHandshake(t, handshakes).err)
	})

	t.Run("Mutual TLS presents the client certificate", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, mutualConfig)
		dataSource := verified(port)
		dataSource.SslCert = pki.clientCertPEM
		dataSource.SslKey = pki.clientKeyPEM
		ping(t, dataSource)
		result := nextHandshake(t, handshakes)
		require.NoError(t, result.err)
		require.Equal(t, "bytebase", result.peer)
	})

	t.Run("TLS mode TLS presents no client certificate", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, mutualConfig)
		ping(t, verified(port))
		require.ErrorContains(t, nextHandshake(t, handshakes).err, "client didn't provide a certificate")
	})

	t.Run("an SSL extra parameter can't turn TLS off", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := tlsTestDataSource(port)
		dataSource.UseSsl = true
		dataSource.ExtraConnectionParameters = map[string]string{"ssl": "FALSE"}
		ping(t, dataSource)
		require.NoError(t, nextHandshake(t, handshakes).err)
	})

	t.Run("an SSL VERIFY extra parameter can't turn verification off", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		dataSource := verified(port)
		dataSource.Host = "127.0.0.1"
		dataSource.ExtraConnectionParameters = map[string]string{"SSL VERIFY": "FALSE"}
		require.Contains(t, ping(t, dataSource), "x509: cannot validate certificate for 127.0.0.1")
		require.Error(t, nextHandshake(t, handshakes).err)
	})

	t.Run("the TLS callback names no server, so go-ora binds each dial", func(t *testing.T) {
		config, err := getTLSConfig(verified("1521"))
		require.NoError(t, err)
		// go-ora sets ServerName on the configuration it is handed, just before the
		// handshake; a name set here would check every server against one host.
		require.Empty(t, config.ServerName)
	})
}

func TestWithoutSQLPlusCommands(t *testing.T) {
	script := "SET DEFINE OFF\nSELECT 1 AS A FROM DUAL"
	blanked := withoutSQLPlusCommands(script)
	require.Len(t, blanked, len(script), "byte offsets of subsequent statements must be preserved")
	require.Equal(t, "SELECT 1 AS A FROM DUAL", strings.TrimLeft(blanked, " "))
	require.Equal(t, strings.Index(script, "SELECT"), strings.Index(blanked, "SELECT"))

	plain := "SELECT 1 FROM DUAL"
	require.Equal(t, plain, withoutSQLPlusCommands(plain))
}

type fakeOracleDriver struct {
	sessionOpens int
	sessions     []*captureOracleSession
	sessionErr   error

	execScript string
	execOpts   omnidriver.ExecuteOptions
	execFn     func(context.Context, string, omnidriver.ExecuteOptions) (int64, error)

	instance    *omnidriver.Instance
	schema      *metadatapb.DatabaseSchemaMetadata
	dumpDDL     string
	estimate    int64
	estimateErr error
	estimateSQL string
	ownsLink    bool
	ownsLinkErr error
}

func (*fakeOracleDriver) Ping(context.Context) error { return nil }
func (*fakeOracleDriver) Close() error               { return nil }

func (f *fakeOracleDriver) Session(context.Context) (omnidriver.Session, error) {
	f.sessionOpens++
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}
	if len(f.sessions) > 0 {
		s := f.sessions[0]
		f.sessions = f.sessions[1:]
		return s, nil
	}
	return &captureOracleSession{}, nil
}

func (f *fakeOracleDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeOracleDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (f *fakeOracleDriver) Dump(_ context.Context, out io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(out, f.dumpDDL)
	return err
}

func (f *fakeOracleDriver) Execute(ctx context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if f.execFn != nil {
		return f.execFn(ctx, script, opts)
	}
	return 1, nil
}

func (f *fakeOracleDriver) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	f.estimateSQL = statement
	return f.estimate, f.estimateErr
}

func (f *fakeOracleDriver) OwnsPrivateDatabaseLink(context.Context) (bool, error) {
	return f.ownsLink, f.ownsLinkErr
}

type captureOracleSession struct {
	gotStatements []string
	gotStatement  string
	gotOpts       omnidriver.QueryOptions
	queryFn       func(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error)
	closed        bool
}

func (s *captureOracleSession) Query(ctx context.Context, stmt string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotStatements = append(s.gotStatements, stmt)
	s.gotStatement = stmt
	s.gotOpts = opts
	if s.queryFn != nil {
		return s.queryFn(ctx, stmt, opts)
	}
	return []*omnidriver.Result{{Statement: stmt}}, nil
}

func (s *captureOracleSession) Close() error {
	s.closed = true
	return nil
}

func TestExecuteAndQueryConnSQLPlusSeam(t *testing.T) {
	ctx := context.Background()
	fake := &fakeOracleDriver{
		execFn: func(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(1, []int64{1}, nil)
			return 1, nil
		},
	}
	session := &captureOracleSession{}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
	d.SetSession(session)

	_, err := d.Execute(ctx, "CREATE TABLE T (ID NUMBER)", db.ExecuteOptions{CreateDatabase: true})
	require.ErrorContains(t, err, "create database is not supported for Oracle")

	var logTypes []storepb.TaskRunLog_Type
	rows, err := d.Execute(ctx, "-- txn-mode = off\nSET DEFINE OFF\nINSERT INTO T VALUES (1)", db.ExecuteOptions{
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logTypes = append(logTypes, entry.Type)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, omnidriver.AutoCommit, fake.execOpts.Transaction)
	require.Empty(t, fake.execOpts.Isolation)
	require.Equal(t, 2, fake.execOpts.MaxRetries)
	require.NotContains(t, fake.execScript, "SET DEFINE OFF")
	require.Contains(t, fake.execScript, "INSERT INTO T VALUES (1)")
	require.Equal(t, []storepb.TaskRunLog_Type{
		storepb.TaskRunLog_COMMAND_EXECUTE,
		storepb.TaskRunLog_COMMAND_RESPONSE,
	}, logTypes)

	_, err = d.QueryConn(ctx, "SET DEFINE OFF\nSELECT 1 FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.NotContains(t, session.gotStatement, "SET DEFINE OFF")
	require.Contains(t, session.gotStatement, "SELECT 1 FROM DUAL")
}

func TestQueryConnAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("reuses one session across calls and converts values", func(t *testing.T) {
		affected := int64(2)
		sess := &captureOracleSession{
			queryFn: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				if strings.HasPrefix(stmt, "ALTER SESSION") {
					return []*omnidriver.Result{{Statement: stmt}}, nil
				}
				return []*omnidriver.Result{
					{
						Statement: stmt,
						Columns:   []omnidriver.Column{{Name: "D", TypeName: "VARCHAR2"}, {Name: "N", TypeName: "NUMBER"}},
						Rows: [][]omnidriver.Value{{
							{Kind: omnidriver.String, String: "2026/01/02"},
							{Kind: omnidriver.Null},
						}},
					},
					{
						Statement:    "UPDATE T SET A = 1",
						AffectedRows: &affected,
					},
				}, nil
			},
		}
		fake := &fakeOracleDriver{sessions: []*captureOracleSession{sess}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
		qctx := db.QueryContext{Schema: "APP", Limit: 5, MaximumSQLResultSize: 1024}

		_, err := d.QueryConn(ctx, "ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY/MM/DD'", qctx)
		require.NoError(t, err)
		results, err := d.QueryConn(ctx, "SELECT TO_CHAR(DATE '2026-01-02') AS D, NULL AS N FROM DUAL", qctx)
		require.NoError(t, err)

		require.Equal(t, 1, fake.sessionOpens)
		require.Same(t, sess, d.Session())
		require.Equal(t, "APP", sess.gotOpts.Schema)
		require.Equal(t, 5, sess.gotOpts.Limit)
		require.Len(t, results, 2)
		require.Equal(t, "2026/01/02", results[0].Rows[0].Values[0].GetStringValue())
		require.IsType(t, (*v1pb.RowValue_NullValue)(nil), results[0].Rows[0].Values[1].Kind)
		require.Equal(t, []string{"Affected Rows"}, results[1].ColumnNames)
		require.Equal(t, int64(2), results[1].Rows[0].Values[0].GetInt64Value())
	})

	t.Run("wraps session open errors", func(t *testing.T) {
		fake := &fakeOracleDriver{sessionErr: errors.New("ORA-01435")}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
		_, err := d.QueryConn(ctx, "SELECT 1 FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
		require.ErrorContains(t, err, "failed to open a session")
		require.ErrorContains(t, err, "ORA-01435")
	})

	t.Run("discards session after canceled context", func(t *testing.T) {
		sess1 := &captureOracleSession{
			queryFn: func(ctx context.Context, _ string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				return nil, ctx.Err()
			},
		}
		sess2 := &captureOracleSession{}
		fake := &fakeOracleDriver{sessions: []*captureOracleSession{sess1, sess2}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := d.QueryConn(canceledCtx, "SELECT 1 FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, sess1.closed)
		require.Nil(t, d.Session())

		_, err = d.QueryConn(ctx, "SELECT 1 FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Equal(t, 2, fake.sessionOpens)
		require.Same(t, sess2, d.Session())
	})
}

func TestProbeAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("runs autocommit probe on fresh session and maps cells", func(t *testing.T) {
		probeSess := &captureOracleSession{
			queryFn: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				return []*omnidriver.Result{{
					Statement: stmt,
					Columns:   []omnidriver.Column{{Name: "OP", TypeName: "VARCHAR2"}, {Name: "META", TypeName: "JSON"}},
					Rows: [][]omnidriver.Value{{
						{Kind: omnidriver.String, String: "SELECT STATEMENT"},
						{Kind: omnidriver.Structured, String: `{"k":"v"}`},
					}},
				}}, nil
			},
		}
		fake := &fakeOracleDriver{sessions: []*captureOracleSession{probeSess}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		res, err := d.Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT 1 FROM DUAL")
		require.NoError(t, err)
		require.Equal(t, []string{"EXPLAIN PLAN FOR SELECT 1 FROM DUAL"}, probeSess.gotStatements)
		require.True(t, probeSess.closed)
		require.Equal(t, []string{"OP", "META"}, res.Columns)
		require.Equal(t, [][]any{{"SELECT STATEMENT", nil}}, res.Rows)
	})

	t.Run("propagates statement error from probe", func(t *testing.T) {
		probeSess := &captureOracleSession{
			queryFn: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				return []*omnidriver.Result{{Statement: stmt, Err: errors.New("ORA-00942: table or view does not exist")}}, nil
			},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeOracleDriver{sessions: []*captureOracleSession{probeSess}}}}

		_, err := d.Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT * FROM NO_SUCH_TABLE")
		require.ErrorContains(t, err, "ORA-00942")
		require.True(t, probeSess.closed)
	})

	t.Run("refuses non-autocommit probes without opening a session", func(t *testing.T) {
		fake := &fakeOracleDriver{}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

		_, err := d.Probe(ctx, db.ProbeContext{PreExecutions: []string{"ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY'"}}, "SELECT 1 FROM DUAL")
		require.ErrorContains(t, err, "runs alone")
		require.Zero(t, fake.sessionOpens)
	})
}

func TestSyncDumpLinksAndRowEstimatorSeam(t *testing.T) {
	ctx := context.Background()
	schema := &metadatapb.DatabaseSchemaMetadata{
		Name:    "APP",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "APP", Tables: []*metadatapb.TableMetadata{{Name: "T"}}}},
	}
	fake := &fakeOracleDriver{
		instance: &omnidriver.Instance{Version: "23.26.3.0.0", Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "APP"}}},
		schema:   schema,
		dumpDDL:  `CREATE TABLE "T" ("ID" NUMBER);`,
		estimate: 42,
		ownsLink: true,
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "23.26.3.0.0", inst.Version)
	require.Len(t, inst.Databases, 1)

	gotSchema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "APP", gotSchema.Name)

	var dump bytes.Buffer
	require.NoError(t, d.Dump(ctx, &dump, gotSchema))
	require.Contains(t, dump.String(), `CREATE TABLE "T"`)

	rows, err := d.CountAffectedRows(ctx, "DELETE FROM T WHERE ID = 1")
	require.NoError(t, err)
	require.Equal(t, int64(42), rows)
	require.Equal(t, "DELETE FROM T WHERE ID = 1", fake.estimateSQL)

	owns, err := d.OwnsPrivateDatabaseLink(ctx)
	require.NoError(t, err)
	require.True(t, owns)

	fake.ownsLink = false
	owns, err = d.OwnsPrivateDatabaseLink(ctx)
	require.NoError(t, err)
	require.False(t, owns)

	fake.ownsLinkErr = errors.New("ORA-00942")
	_, err = d.OwnsPrivateDatabaseLink(ctx)
	require.ErrorContains(t, err, "ORA-00942")
}

func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	return port
}

// testPKI holds a CA, a server certificate it signs for localhost only, and a client
// certificate it signs.
type testPKI struct {
	caPEM         string
	caPool        *x509.CertPool
	server        tls.Certificate
	clientCertPEM string
	clientKeyPEM  string
}

func newTestPKI(t *testing.T) *testPKI {
	t.Helper()
	ca, caKey, caPEM, _ := issueCertificate(t, &x509.Certificate{
		Subject:               pkix.Name{CommonName: "Bytebase Test CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}, nil, nil)
	_, _, serverCertPEM, serverKeyPEM := issueCertificate(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "localhost"},
		DNSNames:    []string{"localhost"},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, ca, caKey)
	_, _, clientCertPEM, clientKeyPEM := issueCertificate(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "bytebase"},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, ca, caKey)
	server, err := tls.X509KeyPair([]byte(serverCertPEM), []byte(serverKeyPEM))
	require.NoError(t, err)
	caPool := x509.NewCertPool()
	caPool.AddCert(ca)
	return &testPKI{
		caPEM:         caPEM,
		caPool:        caPool,
		server:        server,
		clientCertPEM: clientCertPEM,
		clientKeyPEM:  clientKeyPEM,
	}
}

// issueCertificate signs template with parentKey, or self-signs it when parent is nil. It
// returns the certificate, its key, and both in PEM.
func issueCertificate(t *testing.T, template, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (*x509.Certificate, *ecdsa.PrivateKey, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template.SerialNumber, err = rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)
	template.NotBefore = time.Now().Add(-time.Hour)
	template.NotAfter = time.Now().Add(time.Hour)
	if parent == nil {
		parent, parentKey = template, key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certificate, key, string(certificatePEM), string(keyPEM)
}

type handshake struct {
	err error
	// peer is the common name of the client certificate, if one was presented.
	peer string
}

// startTCPSListener stands in for an Oracle TCPS port. It reports each TLS handshake and then
// closes the connection, so the driver fails right after the handshake instead of signing in.
func startTCPSListener(t *testing.T, config *tls.Config) (string, <-chan handshake) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	handshakes := make(chan handshake, 32)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				tlsConn := tls.Server(conn, config)
				defer tlsConn.Close()
				_ = tlsConn.SetDeadline(time.Now().Add(10 * time.Second))
				result := handshake{err: tlsConn.Handshake()}
				if certificates := tlsConn.ConnectionState().PeerCertificates; len(certificates) > 0 {
					result.peer = certificates[0].Subject.CommonName
				}
				select {
				case handshakes <- result:
				default:
				}
			}()
		}
	}()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	return port, handshakes
}

func nextHandshake(t *testing.T, handshakes <-chan handshake) handshake {
	t.Helper()
	select {
	case result := <-handshakes:
		return result
	case <-time.After(20 * time.Second):
		require.FailNow(t, "no connection reached the listener")
		return handshake{}
	}
}

func tlsTestDataSource(port string) *storepb.DataSource {
	return &storepb.DataSource{
		Host:        "localhost",
		Port:        port,
		Username:    "bytebase",
		ServiceName: "ORCLPDB1",
	}
}

// ping opens the driver and returns the message of the ping error that startTCPSListener always
// causes.
func ping(t *testing.T, dataSource *storepb.DataSource) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	driver, err := (&Driver{}).Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{DataSource: dataSource, Password: "secret"})
	require.NoError(t, err)
	defer driver.Close(ctx)
	err = driver.Ping(ctx)
	require.Error(t, err)
	return err.Error()
}
