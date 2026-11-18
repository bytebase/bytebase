package pg

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestPGConfigKeepsEachValueWhole(t *testing.T) {
	open := func(edit func(*storepb.DataSource)) (db.Driver, error) {
		dataSource := &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432"}
		edit(dataSource)
		return (&Driver{}).Open(context.Background(), storepb.Engine_POSTGRES, db.ConnectionConfig{
			DataSource:        dataSource,
			ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
		})
	}

	t.Run("a port cannot name another host", func(t *testing.T) {
		_, err := open(func(ds *storepb.DataSource) { ds.Port = "5432 host=other.example.com" })
		require.ErrorContains(t, err, "invalid port")
	})
	t.Run("a host carrying options stays one host", func(t *testing.T) {
		cfg, err := pgConfig(db.ConnectionConfig{
			DataSource: &storepb.DataSource{Username: "dba", Host: "db.example.com port=6543", Port: "5432"},
		})
		require.NoError(t, err)
		require.Equal(t, "db.example.com port=6543", cfg.Host)
		require.Equal(t, "5432", cfg.Port)
	})
	t.Run("an extra parameter keeps libpq syntax", func(t *testing.T) {
		cfg, err := pgConfig(db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Username:                  "dba",
				Host:                      "db.example.com",
				Port:                      "5432",
				ExtraConnectionParameters: map[string]string{"options": "'-c search_path=hr'"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, "'-c search_path=hr'", cfg.Params["options"])
	})
	t.Run("surrounding whitespace is ignored", func(t *testing.T) {
		cfg, err := pgConfig(db.ConnectionConfig{
			DataSource: &storepb.DataSource{Username: "dba", Host: " db.example.com ", Port: " 5432 "},
		})
		require.NoError(t, err)
		require.Equal(t, "db.example.com", strings.TrimSpace(cfg.Host))
		require.Equal(t, "5432", strings.TrimSpace(cfg.Port))
	})
	t.Run("a blank host is refused", func(t *testing.T) {
		_, err := open(func(ds *storepb.DataSource) { ds.Host = "   " })
		require.ErrorContains(t, err, "host must be set")
	})
	t.Run("a blank port is refused", func(t *testing.T) {
		_, err := open(func(ds *storepb.DataSource) { ds.Port = "   " })
		require.ErrorContains(t, err, "port must be set")
	})
	t.Run("a blank user is refused", func(t *testing.T) {
		_, err := open(func(ds *storepb.DataSource) { ds.Username = "" })
		require.ErrorContains(t, err, "user must be set")
	})
	t.Run("a client certificate needs its key", func(t *testing.T) {
		_, err := open(func(ds *storepb.DataSource) { ds.SslCert = "cert" })
		require.ErrorContains(t, err, "ssl-cert and ssl-key must be both set or unset")
	})
}

func TestPGConfigUsesPgBouncerCompatibleQueryMode(t *testing.T) {
	// The query mode is pgx's, so it is set where pgx is configured: a driver
	// opened with no parameters must still reach a pooler.
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Username: "dba", Host: "pgbouncer.example.com", Port: "6432"},
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
	})
	require.NoError(t, err)
	require.NotContains(t, cfg.Params, "default_query_exec_mode")
	require.NotContains(t, cfg.Params, "statement_cache_capacity")
}

func TestPGConfigReadsTheContextThenTheDataSource(t *testing.T) {
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432", Database: "from-data-source"},
		ConnectionContext: db.ConnectionContext{DatabaseName: "from-context"},
	})
	require.NoError(t, err)
	require.Equal(t, "from-context", cfg.Database)

	cfg, err = pgConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432", Database: "from-data-source"},
	})
	require.NoError(t, err)
	require.Equal(t, "from-data-source", cfg.Database)
}

func TestPGConfigNamesTheSession(t *testing.T) {
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432"},
	})
	require.NoError(t, err)
	require.Equal(t, "bytebase", cfg.ApplicationName)

	taskRunUID := int64(7)
	cfg, err = pgConfig(db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432"},
		ConnectionContext: db.ConnectionContext{TaskRunUID: &taskRunUID, ReadOnly: true, EngineVersion: "16.0"},
	})
	require.NoError(t, err)
	require.Equal(t, "bytebase-taskrun-7", cfg.ApplicationName)
	require.True(t, cfg.ReadOnly)
	require.Equal(t, "16.0", cfg.EngineVersion)
}

func TestRDSIAMBuildsATokenForTheWholeUser(t *testing.T) {
	// Static credentials sign the token locally; no AWS file or endpoint is read.
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/credentials")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// A set AWS_PROFILE makes the SDK load that profile strictly, and the files
	// above have none.
	t.Setenv("AWS_PROFILE", "")
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:               "rds.example.com",
			Port:               "5432",
			Username:           "dba host=other.example.com",
			Region:             "us-east-1",
			AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{AccessKeyId: "AKIDEXAMPLE", SecretAccessKey: "not-a-real-secret"},
			},
		},
	}
	token, err := getRDSConnectionPassword(context.Background(), config)
	require.NoError(t, err)
	require.Contains(t, token, "X-Amz-Signature")

	// The user is a struct field, not a connection-string value, so no part of it
	// can name another keyword.
	cfg, err := pgConfig(config)
	require.NoError(t, err)
	require.Equal(t, "rds.example.com", cfg.Host)
	require.Equal(t, "dba host=other.example.com", cfg.User)
	// The token is the credential; TLS stays pgx's default negotiation, as the
	// connection this replaced left it.
	require.Nil(t, cfg.TLS)
}

func TestPGTLSConfigBindsEveryHost(t *testing.T) {
	hosts := []string{"pg-1.example.com", "pg-2.example.com", "pg-3.example.com"}
	caPEM, serverCertDERByHost := generateCAAndServerCertificates(t, hosts)
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username:             "dba",
			Host:                 strings.Join(hosts, ","),
			Port:                 "5432",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                caPEM,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)

	for _, host := range hosts {
		tlsConfig, err := cfg.TLS(host)
		require.NoError(t, err)
		require.Equal(t, host, tlsConfig.ServerName)
		require.NotNil(t, tlsConfig.RootCAs)
		require.NotNil(t, tlsConfig.VerifyPeerCertificate)
		require.NoError(t, tlsConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[host]}, nil))
		for _, other := range hosts {
			if other == host {
				continue
			}
			require.Error(t, tlsConfig.VerifyPeerCertificate([][]byte{serverCertDERByHost[other]}, nil))
		}
	}
}

func TestPGTLSConfigDisablesVerification(t *testing.T) {
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "primary.example.com,fallback.example.com",
			Port:     "5432",
			UseSsl:   true,
		},
	})
	require.NoError(t, err)

	// SNI still names the host, even though the certificate is not verified.
	for _, host := range []string{"primary.example.com", "fallback.example.com"} {
		tlsConfig, err := cfg.TLS(host)
		require.NoError(t, err)
		require.Equal(t, host, tlsConfig.ServerName)
		require.True(t, tlsConfig.InsecureSkipVerify)
	}
}

func TestPGTLSConfigAddsTheClientCertificate(t *testing.T) {
	certPEM, keyPEM := generateClientCertificatePEM(t)
	cfg, err := pgConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Username: "dba",
			Host:     "db.example.com",
			Port:     "5432",
			UseSsl:   true,
			SslCert:  certPEM,
			SslKey:   keyPEM,
		},
	})
	require.NoError(t, err)
	tlsConfig, err := cfg.TLS("db.example.com")
	require.NoError(t, err)
	require.Len(t, tlsConfig.Certificates, 1)
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

// cloudSQLIAMConnectionConfig returns a Google Cloud SQL IAM data source that
// cloudsqlconn accepts offline. The credential is an authorized_user document so
// that building the connector needs neither a key nor a network round trip; it
// is never good enough to dial with.
const cloudSQLTestInstance = "example-project:us-central1:example-instance"

func cloudSQLIAMConnectionConfig(instance string) db.ConnectionConfig {
	return db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:               storepb.DataSourceType_ADMIN,
			Host:               instance,
			Username:           "bytebase@example-project.iam",
			AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM,
			IamExtension: &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{
					Content: `{"type":"authorized_user","client_id":"not-a-real-client","client_secret":"not-a-real-secret","refresh_token":"not-a-real-token"}`,
				},
			},
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
	}
}

func TestCloudSQLIAMDialerClosesWithDriver(t *testing.T) {
	ctx := context.Background()

	driver := &Driver{}
	opened, err := driver.Open(ctx, storepb.Engine_POSTGRES, cloudSQLIAMConnectionConfig(cloudSQLTestInstance))
	require.NoError(t, err)
	require.Same(t, driver, opened)
	require.NotNil(t, driver.cloudSQLDialer)

	dialer := driver.cloudSQLDialer
	require.NoError(t, driver.Close(ctx))

	// The connector keeps refreshing its ephemeral certificate until it is
	// closed, so a driver that leaves it open leaks it for the life of the
	// process, once per connection opened.
	_, err = dialer.Dial(ctx, cloudSQLTestInstance)
	require.ErrorIs(t, err, cloudsqlconn.ErrDialerClosed)
}

func TestCloudSQLIAMDialerClosesWhenOpenFails(t *testing.T) {
	ctx := context.Background()

	config := cloudSQLIAMConnectionConfig(cloudSQLTestInstance)
	// pgx refuses a port it cannot parse, so this fails Open after the connector
	// has been built.
	config.DataSource.ExtraConnectionParameters = map[string]string{"port": "not-a-port"}

	driver := &Driver{}
	_, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.Error(t, err)

	// A driver whose Open fails is discarded without a Close, so Open has to have
	// closed the connector itself.
	require.NotNil(t, driver.cloudSQLDialer)
	_, err = driver.cloudSQLDialer.Dial(ctx, cloudSQLTestInstance)
	require.ErrorIs(t, err, cloudsqlconn.ErrDialerClosed)
}

// pgTestDriver opens a driver on the shared container's database.
func pgTestDriver(ctx context.Context, t *testing.T, pgContainer *testcontainer.Container, database string, edit func(*db.ConnectionConfig)) *Driver {
	t.Helper()
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
	}
	if edit != nil {
		edit(&config)
	}
	opened, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	driver, ok := opened.(*Driver)
	require.True(t, ok)
	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
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

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	queryContext := db.QueryContext{
		Schema:               "app",
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	}
	results, err := driver.QueryConn(ctx, nil, `SELECT public.polar_alter_relation_to_oss_with_indexes('content_messages_2025q1'::text);`, queryContext)
	require.NoError(t, err)
	require.Equal(t, "content_messages_2025q1", firstStringValue(t, results))

	results, err = driver.QueryConn(ctx, nil, `SELECT marker FROM lookup_precedence;`, queryContext)
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

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	results, err := driver.QueryConn(ctx, nil, `SELECT marker FROM lookup_precedence;`, db.QueryContext{
		Schema:               `app"schema`,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Equal(t, "quoted", firstStringValue(t, results))
}

// TestQueryConnExplainFormat pins both explain outputs at the driver boundary:
// the default stays the human-readable text plan, and JSON returns the plan tree
// as a single row. The JSON assertion is what a plan renderer depends on, and
// running it through QueryConn also proves the editor's read-only validator
// accepts the parenthesized EXPLAIN form.
func TestQueryConnExplainFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `
		CREATE TABLE plan_target (id int PRIMARY KEY, note text);
		INSERT INTO plan_target SELECT g, 'n' || g FROM generate_series(1, 500) g;
		ANALYZE plan_target;
	`)
	require.NoError(t, err)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	const statement = `SELECT note FROM plan_target WHERE id > 100;`

	textResults, err := driver.QueryConn(ctx, nil, statement, db.QueryContext{
		Explain:              true,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.NotEmpty(t, textResults)
	require.Empty(t, textResults[0].GetError())
	require.Nil(t, textResults[0].GetQueryPlan(), "an explain request's format is the request's, not the statement's")
	require.Contains(t, firstStringValue(t, textResults), "(cost=")

	jsonResults, err := driver.QueryConn(ctx, nil, statement, db.QueryContext{
		Explain:              true,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.NoError(t, err)
	require.NotEmpty(t, jsonResults)
	require.Empty(t, jsonResults[0].GetError())
	require.Nil(t, jsonResults[0].GetQueryPlan())
	require.Len(t, jsonResults[0].GetRows(), 1)

	var plan []struct {
		Plan struct {
			NodeType     string  `json:"Node Type"`
			RelationName string  `json:"Relation Name"`
			TotalCost    float64 `json:"Total Cost"`
			PlanRows     float64 `json:"Plan Rows"`
		} `json:"Plan"`
	}
	require.NoError(t, json.Unmarshal([]byte(firstStringValue(t, jsonResults)), &plan))
	require.Len(t, plan, 1)
	require.NotEmpty(t, plan[0].Plan.NodeType)
	require.Equal(t, "plan_target", plan[0].Plan.RelationName)
	require.Positive(t, plan[0].Plan.TotalCost)
	require.Positive(t, plan[0].Plan.PlanRows)

	// A statement that already asks for a plan is planned once: the request's
	// format replaces the caller's EXPLAIN instead of wrapping it, which the
	// server would reject as a syntax error.
	planOfPlan, err := driver.QueryConn(ctx, nil, "EXPLAIN (FORMAT TEXT) "+statement, db.QueryContext{
		Explain:              true,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
	})
	require.NoError(t, err)
	require.NotEmpty(t, planOfPlan)
	require.Empty(t, planOfPlan[0].GetError())
	require.NoError(t, json.Unmarshal([]byte(firstStringValue(t, planOfPlan)), &plan))
	require.Len(t, plan, 1)
	require.Equal(t, "plan_target", plan[0].Plan.RelationName)
}

// TestQueryConnReportsAnAffectedRowForAStatementWithoutACount keeps the editor's
// result shape: a statement that modifies nothing countable still answers one
// "Affected Rows" row, with 0.
func TestQueryConnReportsAnAffectedRowForAStatementWithoutACount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, _ := testcontainer.NewPgDatabase(t)
	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	results, err := driver.QueryConn(ctx, nil, "CREATE TABLE counted (id int);", db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, []string{"Affected Rows"}, results[0].GetColumnNames())
	require.Equal(t, int64(0), results[0].GetRows()[0].GetValues()[0].GetInt64Value())
	require.Equal(t, int64(1), results[0].GetRowsCount())
}

// TestQueryConnHoldsOneSession proves the driver keeps one session across
// QueryConn calls, as the editor's one pooled connection did: a statement's
// session state — here a SET, and a temporary table — holds for the next
// statement of the same stream.
func TestQueryConnHoldsOneSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, _ := testcontainer.NewPgDatabase(t)
	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	queryContext := db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 20}
	_, err := driver.QueryConn(ctx, nil, "SET bytebase.kept = 'yes'; CREATE TEMP TABLE kept (id int);", queryContext)
	require.NoError(t, err)

	results, err := driver.QueryConn(ctx, nil, "SELECT current_setting('bytebase.kept'), (SELECT count(*) FROM kept);", queryContext)
	require.NoError(t, err)
	values := results[0].GetRows()[0].GetValues()
	require.Equal(t, "yes", values[0].GetStringValue())
	require.Equal(t, int64(0), values[1].GetInt64Value())
}

// TestQueryConnReportsAPlanOfItsOwn pins the other side of the split: a caller's
// own EXPLAIN is a plan result, and its typed Plan says which format it is.
func TestQueryConnReportsAPlanOfItsOwn(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `CREATE TABLE plan_target (id int PRIMARY KEY);`)
	require.NoError(t, err)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)
	results, err := driver.QueryConn(ctx, nil, "EXPLAIN (FORMAT JSON) SELECT * FROM plan_target;", db.QueryContext{
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, v1pb.QueryOption_JSON, results[0].GetQueryPlan().GetFormat())
}

func TestQueryConnReportsTheServerError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, _ := testcontainer.NewPgDatabase(t)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)
	results, err := driver.QueryConn(ctx, nil, "SELECT * FROM missing_table;", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Contains(t, results[0].GetError(), `relation "missing_table" does not exist`)
	// The server's own fields ride beside the message, as they did before.
	require.Equal(t, "42P01", results[0].GetPostgresError().GetCode())
	require.Contains(t, results[0].GetPostgresError().GetMessage(), "missing_table")
}

// TestQueryConnWritesATimestampBeyondTheProtoRangeAsText keeps the conversion
// the API needs: PostgreSQL holds dates google.protobuf.Timestamp does not, and
// marshaling such a timestamp in a response fails.
func TestQueryConnWritesATimestampBeyondTheProtoRangeAsText(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, _ := testcontainer.NewPgDatabase(t)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)
	results, err := driver.QueryConn(ctx, nil,
		"SELECT '0001-01-01 BC'::timestamp AS ts, '2024-01-02 03:04:05'::timestamp AS ok;",
		db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].GetError())
	values := results[0].GetRows()[0].GetValues()
	require.Equal(t, "0001-01-01 00:00:00 BC", values[0].GetStringValue())
	require.NotNil(t, values[1].GetTimestampValue(), "a timestamp the proto holds stays a timestamp")
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

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	// Baseline: a plain SELECT returns the secret as a data row. This proves the
	// harness actually surfaces row data, so the negative assertion below is meaningful.
	selectResults, err := driver.QueryConn(ctx, nil, `SELECT secret FROM secret_doc`, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
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
	mergeResults, err := driver.QueryConn(ctx, nil, mergeCTE, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.False(t, resultsContainValue(mergeResults, secret),
		"SECURITY: MERGE-in-CTE must not return the masked column as data rows; got %v", mergeResults)
}

// TestExecuteReportsTheScriptsOrder proves the tenant-mode prelude and the
// script's statements run in the order the script states, which the driver that
// this replaced did not: it moved non-transactional statements ahead.
func TestExecuteReportsTheScriptsOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)
	var mu sync.Mutex
	var executed []string
	_, err := driver.Execute(ctx, "CREATE TABLE first (id int);\nCREATE INDEX CONCURRENTLY second ON first (id);", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			if command := log.GetCommandExecute(); command != nil {
				mu.Lock()
				executed = append(executed, strings.TrimSpace(command.GetStatement()))
				mu.Unlock()
			}
			return nil
		},
		LogCommandStatement: true,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE first (id int);", "CREATE INDEX CONCURRENTLY second ON first (id);"}, executed)

	var count int
	require.NoError(t, rawDB.QueryRow(`SELECT count(*) FROM pg_class WHERE relname = 'second'`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestSyncInstanceAndDump(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE t (id int PRIMARY KEY, note text);
		CREATE SCHEMA %s;
		CREATE TABLE %s.archived (id int);
	`, common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES), common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)))
	require.NoError(t, err)

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, instance.Version)
	require.NotEmpty(t, instance.Metadata.GetRoles())
	var found bool
	for _, database := range instance.Databases {
		if database.Name == dbName {
			found = true
			require.NotEmpty(t, database.Owner)
		}
	}
	require.True(t, found, "the synced database %q", dbName)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, dbName, metadata.GetName())
	require.NotEmpty(t, metadata.GetSearchPath())

	var buf bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &buf, metadata))
	dump := buf.String()
	require.Contains(t, dump, `CREATE TABLE "public"."t"`)
	require.NotContains(t, dump, common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES), "the dump omits the backup schema")
}

func TestPingAMissingDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	driver := pgTestDriver(ctx, t, pgContainer, "no-such-database", nil)

	// The contract states this one: Ping on a database the server does not have
	// fails before any statement reaches it.
	require.ErrorContains(t, driver.Ping(ctx), "3D000")
}

// TestProbeTenantModeRunsAsTheOwner proves the probe plans as the database owner
// in tenant mode: the prelude reads the owner and switches to it on the probe's
// own session, so the statement sees the role the migration will run as. The
// advisor context and the driver's connection context carry the same flag, which
// is what makes this the role a review judges.
func TestProbeTenantModeRunsAsTheOwner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `
		CREATE ROLE app_owner;
		CREATE TABLE owned (id int);
		ALTER TABLE owned OWNER TO app_owner;
	`)
	require.NoError(t, err)
	_, err = rawDB.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s OWNER TO app_owner", dbName))
	require.NoError(t, err)

	tenantDriver := pgTestDriver(ctx, t, pgContainer, dbName, func(config *db.ConnectionConfig) {
		config.ConnectionContext.TenantMode = true
	})
	result, err := db.ProberOf(tenantDriver).Probe(ctx, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, [][]any{{"app_owner"}}, result.Rows)

	// The login is still the superuser: the prelude switched the session role, not
	// the credentials.
	var currentUser string
	require.NoError(t, rawDB.QueryRow("SELECT CURRENT_USER").Scan(&currentUser))
	require.Equal(t, "postgres", currentUser)

	plainDriver := pgTestDriver(ctx, t, pgContainer, dbName, nil)
	result, err = db.ProberOf(plainDriver).Probe(ctx, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, [][]any{{"postgres"}}, result.Rows)
}

func TestTaskRunSessionReadsTheRunSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)

	// The task run's driver names its session after the run; the reader is a
	// second driver, whose own sessions carry the plain name, so it cannot read
	// itself. GetTaskRunSession opens one the same way.
	taskRunUID := int64(42)
	runDriver := pgTestDriver(ctx, t, pgContainer, dbName, func(config *db.ConnectionConfig) {
		config.ConnectionContext.TaskRunUID = &taskRunUID
	})
	reader := pgTestDriver(ctx, t, pgContainer, dbName, nil)

	go func() {
		_, _ = runDriver.QueryConn(ctx, nil, "SELECT pg_sleep(5);", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	}()
	t.Cleanup(func() {
		_, _ = rawDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name = 'bytebase-taskrun-42'`)
	})

	require.Eventually(t, func() bool {
		var count int
		if err := rawDB.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE application_name = 'bytebase-taskrun-42'`).Scan(&count); err != nil {
			return false
		}
		return count == 1
	}, 10*time.Second, 20*time.Millisecond, "the query never started on the server")

	session, err := reader.TaskRunSession(ctx, "bytebase-taskrun-42")
	require.NoError(t, err)
	require.NotNil(t, session.GetPostgres().GetSession())
	require.NotEmpty(t, session.GetPostgres().GetSession().GetPid())
	require.Contains(t, session.GetPostgres().GetSession().GetQuery(), "pg_sleep")
}

func firstStringValue(t *testing.T, results []*v1pb.QueryResult) string {
	t.Helper()
	require.NotEmpty(t, results)
	require.Empty(t, results[0].GetError())
	require.NotEmpty(t, results[0].GetRows())
	require.NotEmpty(t, results[0].GetRows()[0].GetValues())
	return results[0].GetRows()[0].GetValues()[0].GetStringValue()
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
