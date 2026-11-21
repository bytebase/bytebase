package oracle

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql/driver"
	"encoding/pem"
	"fmt"
	"maps"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	goora "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/converters"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
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

// A character set latched on go-ora's process-wide driver must not reach Bytebase's
// connections. WE8ISO8859P1 stands in for another Oracle data source, as a ZHS16GBK one did in
// #21902: its decoder mangles this container's AL32UTF8 bytes, so text that survives the round
// trip proves the connection used its own server's character set.
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

func TestQueryIgnoresTheSharedDriverCharset(t *testing.T) {
	ctx := context.Background()
	container, schema := testcontainer.NewOracleSchema(t)
	// A VARCHAR2 column, which the server sends in the database character set. UNISTR spells
	// the text out in code points, so no client encoding decides what gets stored.
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.CHARSET_PROBE (C VARCHAR2(100))", schema))
	require.NoError(t, err)
	_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s.CHARSET_PROBE VALUES (UNISTR('\6570\636E\5E93'))`, schema))
	require.NoError(t, err)

	const we8iso8859p1, al16utf16 = 31, 2000
	goora.SetStringConverter(sharedGoOraDriver{}, converters.NewStringConverter(we8iso8859p1), converters.NewStringConverter(al16utf16))
	t.Cleanup(func() { goora.SetStringConverter(sharedGoOraDriver{}, nil, nil) })

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schema},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	results, err := driver.QueryConn(ctx, "SELECT C FROM CHARSET_PROBE", db.QueryContext{MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Equal(t, "数据库", results[0].Rows[0].Values[0].GetStringValue())
}

// sharedGoOraDriver hands go-ora's process-wide driver to SetStringConverter, which takes
// whatever exposes a Driver method.
type sharedGoOraDriver struct{}

func (sharedGoOraDriver) Driver() driver.Driver { return goora.GetDefaultDriver() }

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

func TestCurrentSchemaOnNewSessions(t *testing.T) {
	ctx := context.Background()
	container, schema := testcontainer.NewOracleSchema(t)
	open := func(schema string) (db.Driver, error) {
		return (&Driver{}).Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:        container.GetHost(),
				Port:        container.GetPort(),
				Username:    container.GetUsername(),
				ServiceName: testcontainer.OracleServiceName,
			},
			Password:          container.GetPassword(),
			ConnectionContext: db.ConnectionContext{DatabaseName: schema},
		})
	}
	t.Run("QueryConn", func(t *testing.T) {
		// A driver per subtest is a fresh pool, so this reads the schema the driver
		// set on the session it opened rather than one an earlier statement did.
		driver, err := open(schema)
		require.NoError(t, err)
		defer driver.Close(ctx)
		results, err := driver.QueryConn(ctx, "SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		require.Equal(t, schema, results[0].Rows[0].Values[0].GetStringValue())
	})

	t.Run("Execute", func(t *testing.T) {
		driver, err := open(schema)
		require.NoError(t, err)
		defer driver.Close(ctx)
		_, err = driver.Execute(ctx, "CREATE TABLE SESSION_SCHEMA AS SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') AS NAME FROM DUAL", db.ExecuteOptions{})
		require.NoError(t, err)
		var name string
		require.NoError(t, container.GetDB().QueryRowContext(ctx, fmt.Sprintf("SELECT NAME FROM %s.SESSION_SCHEMA", schema)).Scan(&name))
		require.Equal(t, schema, name)
	})

	t.Run("Open fails on a schema that doesn't exist", func(t *testing.T) {
		_, err := open("NO_SUCH_SCHEMA")
		require.ErrorContains(t, err, `failed to set current schema to "NO_SUCH_SCHEMA"`)
	})
}

// TestQueryConnKeepsTheSessionForTheDriver pins that an editor stream's
// statements share one session, which a session opened per QueryConn call would
// lose: the API keeps the driver for the whole stream and calls QueryConn once
// per statement. ALTER SESSION in the first call must therefore reach the SELECT
// in the second, whose text would otherwise render in the server's default
// format.
func TestQueryConnKeepsTheSessionForTheDriver(t *testing.T) {
	ctx := context.Background()
	container, schema := testcontainer.NewOracleSchema(t)
	driver, err := newDriver().Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schema},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)
	queryContext := db.QueryContext{MaximumSQLResultSize: 1024}

	results, err := driver.QueryConn(ctx, "ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY/MM/DD'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	results, err = driver.QueryConn(ctx, "SELECT TO_CHAR(DATE '2026-01-02') AS D FROM DUAL", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, "2026/01/02", results[0].Rows[0].Values[0].GetStringValue())
}

// TestProbeRunsTheDryRunAutocommit pins the advisor seam: EXPLAIN PLAN FOR is a
// PL/SQL block opener, so the rule passes Autocommit and the adapter must run the
// statement alone rather than refuse it.
func TestProbeRunsTheDryRunAutocommit(t *testing.T) {
	ctx := context.Background()
	container, schema := testcontainer.NewOracleSchema(t)
	driver, err := newDriver().Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schema},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)
	prober, ok := driver.(db.Prober)
	require.True(t, ok)

	_, err = prober.Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT 1 FROM DUAL")
	require.NoError(t, err)

	_, err = prober.Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN PLAN FOR SELECT * FROM NO_SUCH_TABLE")
	require.ErrorContains(t, err, "ORA-00942")

	// The contract's session has no transaction to roll a non-autocommit probe
	// back with, so it is refused rather than run and left behind.
	_, err = prober.Probe(ctx, db.ProbeContext{PreExecutions: []string{"ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY'"}}, "SELECT 1 FROM DUAL")
	require.ErrorContains(t, err, "runs alone")
}

// TestOwnsPrivateDatabaseLinkAsksTheDriver pins the guard's capability: with no
// *sql.DB handle, the driver asks with a session of its own.
func TestOwnsPrivateDatabaseLinkAsksTheDriver(t *testing.T) {
	ctx := context.Background()
	container, schema := testcontainer.NewOracleSchema(t)
	driver, err := newDriver().Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schema},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	owner, ok := driver.(interface {
		OwnsPrivateDatabaseLink(context.Context) (bool, error)
	})
	require.True(t, ok)
	owns, err := owner.OwnsPrivateDatabaseLink(ctx)
	require.NoError(t, err)
	require.False(t, owns)
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
	execScript string
	execOpts   omnidriver.ExecuteOptions
}

func (*fakeOracleDriver) Ping(context.Context) error { return nil }
func (*fakeOracleDriver) Close() error               { return nil }

func (f *fakeOracleDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	return 1, nil
}

type captureOracleSession struct {
	gotStatement string
	gotOpts      omnidriver.QueryOptions
}

func (s *captureOracleSession) Query(_ context.Context, stmt string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotStatement = stmt
	s.gotOpts = opts
	return []*omnidriver.Result{{Statement: stmt}}, nil
}

func (*captureOracleSession) Close() error { return nil }

func TestExecuteAndQueryConnSQLPlusSeam(t *testing.T) {
	ctx := context.Background()
	fake := &fakeOracleDriver{}
	session := &captureOracleSession{}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
	d.SetSession(session)

	_, err := d.Execute(ctx, "CREATE TABLE T (ID NUMBER)", db.ExecuteOptions{CreateDatabase: true})
	require.ErrorContains(t, err, "create database is not supported for Oracle")

	rows, err := d.Execute(ctx, "-- txn-mode = off\nSET DEFINE OFF\nINSERT INTO T VALUES (1)", db.ExecuteOptions{MaximumRetries: 2})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, omnidriver.AutoCommit, fake.execOpts.Transaction)
	require.Empty(t, fake.execOpts.Isolation)
	require.Equal(t, 2, fake.execOpts.MaxRetries)
	require.NotContains(t, fake.execScript, "SET DEFINE OFF")
	require.Contains(t, fake.execScript, "INSERT INTO T VALUES (1)")

	_, err = d.QueryConn(ctx, "SET DEFINE OFF\nSELECT 1 FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.NotContains(t, session.gotStatement, "SET DEFINE OFF")
	require.Contains(t, session.gotStatement, "SELECT 1 FROM DUAL")
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
