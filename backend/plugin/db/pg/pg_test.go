package pg

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetDatabaseInCreateDatabaseStatement(t *testing.T) {
	tests := []struct {
		createDatabaseStatement string
		want                    string
		wantErr                 bool
	}{
		{
			`CREATE DATABASE "hello" ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE "hello";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello;`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE hello ENCODING "UTF8";`,
			"hello",
			false,
		},
		{
			`CREATE DATABASE;`,
			"",
			true,
		},
	}

	for _, test := range tests {
		got, err := getDatabaseInCreateDatabaseStatement(test.createDatabaseStatement)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, test.want, got)
	}
}

func TestGetPGConnectionConfigUsesPgBouncerCompatibleQueryMode(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "pgbouncer.example.com",
			Port:     "6432",
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "postgres",
		},
	})
	require.NoError(t, err)
	require.Equal(t, pgx.QueryExecModeExec, connConfig.DefaultQueryExecMode)
	require.Zero(t, connConfig.StatementCacheCapacity)
}

func TestGetPGConnectionConfigPreservesExplicitQueryExecMode(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "proxy.example.com",
			Port:     "6432",
			ExtraConnectionParameters: map[string]string{
				"default_query_exec_mode":  "simple_protocol",
				"statement_cache_capacity": "128",
			},
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "postgres",
		},
	})
	require.NoError(t, err)
	require.Equal(t, pgx.QueryExecModeSimpleProtocol, connConfig.DefaultQueryExecMode)
	require.Equal(t, 128, connConfig.StatementCacheCapacity)
}

func TestGetPGConnectionConfigDisablesTLSVerificationForAllHosts(t *testing.T) {
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "172.18.22.61,172.18.22.62,172.18.22.63",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: false,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.True(t, connConfig.TLSConfig.InsecureSkipVerify)
	require.Len(t, connConfig.Fallbacks, 2)
	for _, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.True(t, fallback.TLSConfig.InsecureSkipVerify)
	}
}

func TestGetPGConnectionConfigAddsClientCertificateForAllHosts(t *testing.T) {
	certPEM, keyPEM := generateClientCertificatePEM(t)
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "172.18.22.61,172.18.22.62,172.18.22.63",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: false,
			SslCert:              certPEM,
			SslKey:               keyPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.Len(t, connConfig.TLSConfig.Certificates, 1)
	require.Len(t, connConfig.Fallbacks, 2)
	for _, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.Len(t, fallback.TLSConfig.Certificates, 1)
	}
}

func TestGetPGConnectionConfigVerifiesCustomCAForAllHosts(t *testing.T) {
	hosts := []string{"pg-1.example.com", "pg-2.example.com", "pg-3.example.com"}
	caPEM, serverCertDERByHost := generateCAAndServerCertificates(t, hosts)
	connConfig, err := getPGConnectionConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 "pg-1.example.com,pg-2.example.com,pg-3.example.com",
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                caPEM,
		},
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "lapidlive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, connConfig.TLSConfig)
	require.NotNil(t, connConfig.TLSConfig.RootCAs)
	require.NotNil(t, connConfig.TLSConfig.VerifyPeerCertificate)
	require.NoError(t, connConfig.TLSConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[hosts[0]]}, nil))
	require.Len(t, connConfig.Fallbacks, 2)
	for i, fallback := range connConfig.Fallbacks {
		require.NotNil(t, fallback.TLSConfig)
		require.NotNil(t, fallback.TLSConfig.RootCAs)
		require.NotNil(t, fallback.TLSConfig.VerifyPeerCertificate)
		require.NoError(t, fallback.TLSConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[hosts[i+1]]}, nil))
	}
}

func generateClientCertificatePEM(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "bytebase-test-client",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return string(certPEM), string(keyPEM)
}

func generateCAAndServerCertificates(t *testing.T, hosts []string) (string, map[string][]byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bytebase-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	serverCertDERByHost := make(map[string][]byte, len(hosts))
	for i, host := range hosts {
		serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		serverTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 2)),
			Subject:      pkix.Name{CommonName: host},
			DNSNames:     []string{host},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
		require.NoError(t, err)
		serverCertDERByHost[host] = serverDER
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return string(caPEM), serverCertDERByHost
}

func TestQueryConnSearchPathIncludesPublicAfterSelectedSchema(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `
		CREATE SCHEMA app;
		CREATE TABLE public.lookup_precedence (marker text);
		CREATE TABLE app.lookup_precedence (marker text);
		INSERT INTO public.lookup_precedence VALUES ('public');
		INSERT INTO app.lookup_precedence VALUES ('app');
		CREATE FUNCTION public.polar_osfs_extract_table_name(rel text) RETURNS text
			LANGUAGE sql
			AS $$ SELECT rel $$;
		CREATE FUNCTION public.polar_alter_relation_to_oss_with_indexes(rel text) RETURNS text
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN polar_osfs_extract_table_name(rel);
			END;
			$$;
	`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	queryContext := db.QueryContext{
		Schema:               "app",
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	}
	results, err := driver.QueryConn(ctx, conn, `SELECT public.polar_alter_relation_to_oss_with_indexes('content_messages_2025q1'::text);`, queryContext)
	require.NoError(t, err)
	require.Equal(t, "content_messages_2025q1", firstStringValue(t, results))

	results, err = driver.QueryConn(ctx, conn, `SELECT marker FROM lookup_precedence;`, queryContext)
	require.NoError(t, err)
	require.Equal(t, "app", firstStringValue(t, results))
}

func TestQueryConnSearchPathEscapesSelectedSchemaName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `
		CREATE SCHEMA "app""schema";
		CREATE TABLE "app""schema".lookup_precedence (marker text);
		INSERT INTO "app""schema".lookup_precedence VALUES ('quoted');
	`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	results, err := driver.QueryConn(ctx, conn, `SELECT marker FROM lookup_precedence;`, db.QueryContext{
		Schema:               `app"schema`,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Equal(t, "quoted", firstStringValue(t, results))
}

func firstStringValue(t *testing.T, results []*v1pb.QueryResult) string {
	t.Helper()
	require.NotEmpty(t, results)
	require.Empty(t, results[0].GetError())
	require.NotEmpty(t, results[0].GetRows())
	require.NotEmpty(t, results[0].GetRows()[0].GetValues())
	return results[0].GetRows()[0].GetValues()[0].GetStringValue()
}

// TestQueryConnMergeCTEDoesNotLeakRows is a security regression test for the
// PostgreSQL data-masking bypass where a MERGE smuggled into a data-modifying CTE
//
//	WITH x AS (MERGE ... RETURNING masked_col) SELECT masked_col FROM x
//
// was misclassified as a read-only query (hasDMLInTree did not recognise MergeStmt
// as a write). That routed it through QueryConn's row-returning QueryContext path,
// so the RETURNING columns came back to the client and — because the query-span
// extractor cannot trace lineage through a MERGE — were returned UNMASKED.
//
// The fix classifies MERGE as a write, so the editor routes the statement to
// ExecContext, which discards the RETURNING rows. This test asserts the
// security-relevant outcome at the driver boundary: the MERGE-CTE must NOT return
// the secret value as data rows. MERGE ... RETURNING requires PostgreSQL 17.
func TestQueryConnMergeCTEDoesNotLeakRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPg17Container(t)
	dbName, rawDB := testcontainer.NewPg17Database(t)

	const secret = "TOP-SECRET-SSN-123-45-6789"
	require.NoError(t, rawDB.Ping())
	_, err := rawDB.ExecContext(ctx, `CREATE TABLE secret_doc (id int PRIMARY KEY, secret text);`)
	require.NoError(t, err)
	_, err = rawDB.ExecContext(ctx, `INSERT INTO secret_doc VALUES (1, '`+secret+`');`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Baseline: a plain SELECT returns the secret as a data row. This proves the
	// harness actually surfaces row data, so the negative assertion below is meaningful.
	selectResults, err := driver.QueryConn(ctx, conn, `SELECT secret FROM secret_doc`, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.True(t, resultsContainValue(selectResults, secret),
		"baseline plain SELECT should return the secret as a data row")

	// Exploit attempt: a MERGE smuggled into a data-modifying CTE. After the fix it
	// is classified as a write and routed to Exec, so the RETURNING rows are dropped
	// and the secret is never returned as data.
	const mergeCTE = `WITH x AS (
		MERGE INTO secret_doc t USING secret_doc s ON t.id = s.id
		WHEN MATCHED THEN UPDATE SET id = t.id
		RETURNING t.secret AS secret
	) SELECT secret FROM x`
	mergeResults, err := driver.QueryConn(ctx, conn, mergeCTE, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.False(t, resultsContainValue(mergeResults, secret),
		"SECURITY: MERGE-in-CTE must not return the masked column as data rows; got %v", mergeResults)
}

// resultsContainValue reports whether any string cell in the results contains want.
func resultsContainValue(results []*v1pb.QueryResult, want string) bool {
	for _, r := range results {
		for _, row := range r.GetRows() {
			for _, v := range row.GetValues() {
				if sv, ok := v.GetKind().(*v1pb.RowValue_StringValue); ok && strings.Contains(sv.StringValue, want) {
					return true
				}
			}
		}
	}
	return false
}
