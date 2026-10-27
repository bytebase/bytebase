package oracle

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"maps"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	goora "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/configurations"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
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

func TestBuildExecuteCommandsSplitsLargeOracleScript(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`CREATE TABLE APPS.M_SKU_STOCK_KEY_TMP (
  SKU_CODE VARCHAR2(100) NOT NULL,
  STOCK_KEY VARCHAR2(100)
);
`)
	builder.WriteString("CREATE INDEX IDX_TMP_SKU_CODE ON APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE);\n")
	for i := 0; builder.Len() <= common.MaxSheetCheckSize+1024; i++ {
		_, err := fmt.Fprintf(&builder, "INSERT INTO APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE, STOCK_KEY) VALUES('SKU_%06d', '%s');\n", i, strings.Repeat("x", 64))
		require.NoError(t, err)
	}
	statement := builder.String()
	require.Greater(t, len(statement), common.MaxSheetCheckSize)

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)

	require.Greater(t, len(commands), 2)
	require.NotEqual(t, statement, commands[0].Text)
	require.Equal(t, "CREATE TABLE APPS.M_SKU_STOCK_KEY_TMP (\n  SKU_CODE VARCHAR2(100) NOT NULL,\n  STOCK_KEY VARCHAR2(100)\n)", strings.TrimSpace(commands[0].Text))
	require.Equal(t, "CREATE INDEX IDX_TMP_SKU_CODE ON APPS.M_SKU_STOCK_KEY_TMP(SKU_CODE)", strings.TrimSpace(commands[1].Text))
}

func TestConnectionOptions(t *testing.T) {
	tests := []struct {
		name            string
		sid             string
		extraParameters map[string]string
		useTLS          bool
		want            map[string]string
	}{
		{
			name: "no extra parameters",
			want: map[string]string{"CONNECTION TIMEOUT": "10"},
		},
		{
			name: "SID field",
			sid:  "ORCL",
			want: map[string]string{"CONNECTION TIMEOUT": "10", "SID": "ORCL"},
		},
		{
			name:            "dial timeout in the exact spelling",
			extraParameters: map[string]string{"CONNECTION TIMEOUT": "5"},
			want:            map[string]string{"CONNECTION TIMEOUT": "5"},
		},
		{
			name:            "dial timeout in lowercase",
			extraParameters: map[string]string{"connection timeout": "5"},
			want:            map[string]string{"connection timeout": "5"},
		},
		{
			name:            "dial timeout alias",
			extraParameters: map[string]string{"CONNECT TIMEOUT": "5"},
			want:            map[string]string{"CONNECT TIMEOUT": "5"},
		},
		{
			name:            "dial timeout alias in mixed case",
			extraParameters: map[string]string{"Connect Timeout": "5"},
			want:            map[string]string{"Connect Timeout": "5"},
		},
		{
			name:            "dial timeout turned off",
			extraParameters: map[string]string{"CONNECTION TIMEOUT": "0"},
			want:            map[string]string{"CONNECTION TIMEOUT": "0"},
		},
		{
			name:            "read timeout is a different option",
			extraParameters: map[string]string{"TIMEOUT": "30"},
			want:            map[string]string{"TIMEOUT": "30", "CONNECTION TIMEOUT": "10"},
		},
		{
			name:            "SID key in the exact spelling",
			sid:             "ORCL",
			extraParameters: map[string]string{"SID": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "SID": "STANDBY"},
		},
		{
			name:            "SID key in lowercase",
			sid:             "ORCL",
			extraParameters: map[string]string{"sid": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "sid": "STANDBY"},
		},
		{
			name:            "SID key without the SID field",
			extraParameters: map[string]string{"Sid": "STANDBY"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "Sid": "STANDBY"},
		},
		{
			name: "other keys pass through in their own spelling",
			sid:  "ORCL",
			extraParameters: map[string]string{
				"SERVER":          "standby-1.example.com:1521, standby-2.example.com:1521",
				"SSL":             "true",
				"ssl verify":      "false",
				"WALLET":          "/opt/oracle/wallet",
				"WALLET PASSWORD": "wallet-secret",
				"AUTH TYPE":       "TCPS",
				"prefetch_rows":   "500",
			},
			want: map[string]string{
				"CONNECTION TIMEOUT": "10",
				"SID":                "ORCL",
				"SERVER":             "standby-1.example.com:1521, standby-2.example.com:1521",
				"SSL":                "true",
				"ssl verify":         "false",
				"WALLET":             "/opt/oracle/wallet",
				"WALLET PASSWORD":    "wallet-secret",
				"AUTH TYPE":          "TCPS",
				"prefetch_rows":      "500",
			},
		},
		{
			name:   "TLS mode TLS sets SSL",
			useTLS: true,
			want:   map[string]string{"CONNECTION TIMEOUT": "10", "SSL": "TRUE"},
		},
		{
			name:            "TLS mode TLS replaces an SSL key in any spelling",
			extraParameters: map[string]string{"ssl": "FALSE"},
			useTLS:          true,
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "SSL": "TRUE"},
		},
		{
			name:            "TLS mode TLS passes the other TLS keys through",
			extraParameters: map[string]string{"SSL VERIFY": "FALSE", "WALLET": "/opt/oracle/wallet"},
			useTLS:          true,
			want: map[string]string{
				"CONNECTION TIMEOUT": "10",
				"SSL":                "TRUE",
				"SSL VERIFY":         "FALSE",
				"WALLET":             "/opt/oracle/wallet",
			},
		},
		{
			name:            "TLS mode Disabled passes an SSL key through",
			extraParameters: map[string]string{"ssl": "TRUE"},
			want:            map[string]string{"CONNECTION TIMEOUT": "10", "ssl": "TRUE"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extraParameters := maps.Clone(test.extraParameters)
			got := connectionOptions(test.sid, extraParameters, test.useTLS)
			require.Equal(t, test.want, got)
			require.Equal(t, test.extraParameters, extraParameters)
		})
	}
}

// go-ora reads the options back from the URL, so these cases check what the driver ends up
// with rather than what Bytebase sent.
func TestConnectionOptionsAsReadByDriver(t *testing.T) {
	parse := func(t *testing.T, sid string, extraParameters map[string]string, useTLS bool) *configurations.ConnectionConfig {
		t.Helper()
		dsn := goora.BuildUrl("primary.example.com", 1521, "ORCLPDB1", "bytebase", "secret", connectionOptions(sid, extraParameters, useTLS))
		config, err := configurations.ParseConfig(dsn)
		require.NoError(t, err)
		return config
	}

	t.Run("default", func(t *testing.T) {
		config := parse(t, "", nil, false)
		require.Equal(t, 10*time.Second, config.ConnectTimeout)
		// A read timeout would cut off long statements.
		require.Zero(t, config.Timeout)
		require.Empty(t, config.SID)
	})

	for _, key := range []string{"CONNECTION TIMEOUT", "connection timeout", "CONNECT TIMEOUT", "connect timeout", "Connect Timeout"} {
		t.Run("dial timeout as "+key, func(t *testing.T) {
			config := parse(t, "", map[string]string{key: "5"}, false)
			require.Equal(t, 5*time.Second, config.ConnectTimeout)
			require.Zero(t, config.Timeout)
		})
	}

	t.Run("dial timeout turned off", func(t *testing.T) {
		config := parse(t, "", map[string]string{"connect timeout": "0"}, false)
		require.Zero(t, config.ConnectTimeout)
	})

	t.Run("SID field", func(t *testing.T) {
		config := parse(t, "ORCL", nil, false)
		require.Equal(t, "ORCL", config.SID)
	})

	t.Run("SID key replaces the SID field", func(t *testing.T) {
		config := parse(t, "ORCL", map[string]string{"sid": "STANDBY"}, false)
		require.Equal(t, "STANDBY", config.SID)
	})

	t.Run("second address over TCPS", func(t *testing.T) {
		config := parse(t, "", map[string]string{
			"SERVER":     "standby-1.example.com:2484, standby-2.example.com:2484",
			"SSL":        "true",
			"SSL VERIFY": "false",
			"AUTH TYPE":  "TCPS",
		}, false)
		require.Equal(t, []configurations.ServerAddr{
			{Addr: "primary.example.com", Port: 1521},
			{Addr: "standby-1.example.com", Port: 2484},
			{Addr: "standby-2.example.com", Port: 2484},
		}, config.Servers)
		require.True(t, config.SSL)
		require.False(t, config.SSLVerify)
		require.Equal(t, configurations.TCPS, config.AuthType)
		require.Equal(t, 10*time.Second, config.ConnectTimeout)
	})

	for _, key := range []string{"SSL", "ssl", "Ssl"} {
		t.Run("TLS mode TLS overrides "+key+"=FALSE", func(t *testing.T) {
			config := parse(t, "", map[string]string{key: "FALSE"}, true)
			require.True(t, config.SSL)
		})
	}

	t.Run("TLS mode Disabled reads SSL from extra parameters", func(t *testing.T) {
		config := parse(t, "", map[string]string{"ssl": "true"}, false)
		require.True(t, config.SSL)
	})
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

	t.Run("each connection gets its own copy of the TLS configuration", func(t *testing.T) {
		port, handshakes := startTCPSListener(t, serverConfig)
		portNumber, err := strconv.Atoi(port)
		require.NoError(t, err)
		config, err := getTLSConfig(verified(port))
		require.NoError(t, err)
		c := &connector{
			dsn:       goora.BuildUrl("localhost", portNumber, "ORCLPDB1", "bytebase", "secret", connectionOptions("", nil, true)),
			tlsConfig: config,
		}
		_, err = c.Connect(context.Background())
		require.Error(t, err)
		require.NoError(t, nextHandshake(t, handshakes).err)
		// go-ora sets ServerName on the configuration it is handed, just before the handshake.
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
	driver, err := open(schema)
	require.NoError(t, err)
	defer driver.Close(ctx)
	// With no idle connection kept, the pool opens a new session for every connection it hands out.
	driver.GetDB().SetMaxIdleConns(0)

	t.Run("QueryConn", func(t *testing.T) {
		conn, err := driver.GetDB().Conn(ctx)
		require.NoError(t, err)
		defer conn.Close()
		results, err := driver.QueryConn(ctx, conn, "SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') FROM DUAL", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		require.Equal(t, schema, results[0].Rows[0].Values[0].GetStringValue())
	})

	t.Run("Execute", func(t *testing.T) {
		_, err := driver.Execute(ctx, "CREATE TABLE SESSION_SCHEMA AS SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') AS NAME FROM DUAL", db.ExecuteOptions{})
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

// TestQueryPlanSteps covers the one form of plan the query-plan viewer draws:
// Oracle states a plan as rows of PLAN_TABLE, and the driver is where those
// become the JSON document the viewer parses.
func TestQueryPlanSteps(t *testing.T) {
	t.Parallel()
	connector := &fakePlanConnector{
		columns: []string{"ID", "PARENT_ID", "OPERATION", "OPTIONS", "OBJECT_NAME", "COST", "ACCESS_PREDICATES"},
		plan: [][]driver.Value{
			{"0", nil, "SELECT STATEMENT", nil, nil, "48", nil},
			// The driver hands text back as bytes, which JSON would otherwise
			// encode as base64.
			{"1", "0", "TABLE ACCESS", []byte("FULL"), []byte("ORDERS"), "48", []byte(`"TOTAL">880`)},
		},
	}
	database := sql.OpenDB(connector)
	defer database.Close()
	conn, err := database.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	result, err := queryPlanSteps(context.Background(), conn, "SELECT ...")
	require.NoError(t, err)
	require.Equal(t, []string{"PLAN"}, result.GetColumnNames())
	require.Len(t, result.GetRows(), 1)

	var steps []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.GetRows()[0].GetValues()[0].GetStringValue()), &steps))
	require.Equal(t, []map[string]any{
		// A column the optimizer said nothing about is left out, not carried
		// as a null the viewer would have to skip.
		{"id": "0", "operation": "SELECT STATEMENT", "cost": "48"},
		{
			"id": "1", "parent_id": "0", "operation": "TABLE ACCESS",
			"options": "FULL", "object_name": "ORDERS", "cost": "48",
			"access_predicates": `"TOTAL">880`,
		},
	}, steps)
}

// TestRenderPlanStepsHandlesARaggedRow keeps the rendering off the end of a row
// shorter than the column list, which no Oracle returns but a fake might.
func TestRenderPlanStepsHandlesARaggedRow(t *testing.T) {
	t.Parallel()
	payload, err := renderPlanSteps([]string{"ID", "COST"}, [][]any{{"0"}})
	require.NoError(t, err)
	require.JSONEq(t, `[{"id":"0"}]`, payload)

	// A plan with no steps is an empty list rather than a JSON null.
	payload, err = renderPlanSteps([]string{"ID"}, nil)
	require.NoError(t, err)
	require.JSONEq(t, `[]`, payload)
}
