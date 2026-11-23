package pg

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	omndriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
	token, err := util.RDSAuthToken(context.Background(), config)
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

type stubPGDriver struct {
	sessionOpens int
	session      *stubPGSession
	instance     *omndriver.Instance
	dumpedMeta   *metadatapb.DatabaseSchemaMetadata
}

func (*stubPGDriver) Ping(context.Context) error { return nil }
func (*stubPGDriver) Close() error               { return nil }
func (d *stubPGDriver) Session(context.Context) (omndriver.Session, error) {
	d.sessionOpens++
	return d.session, nil
}
func (d *stubPGDriver) SyncInstance(context.Context) (*omndriver.Instance, error) {
	return d.instance, nil
}
func (*stubPGDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return nil, nil
}
func (d *stubPGDriver) Dump(_ context.Context, w io.Writer, meta *metadatapb.DatabaseSchemaMetadata) error {
	d.dumpedMeta = meta
	_, err := io.WriteString(w, "dumped")
	return err
}

type stubPGSession struct {
	queryFn func(context.Context, string, omndriver.QueryOptions) ([]*omndriver.Result, error)
}

func (s *stubPGSession) Query(ctx context.Context, script string, opts omndriver.QueryOptions) ([]*omndriver.Result, error) {
	return s.queryFn(ctx, script, opts)
}
func (*stubPGSession) Close() error { return nil }

// TestQueryConnAdapterSeam pins PG-specific adapter transformations in
// QueryConn: held session reuse, lowercase ExplainFormat mapping and QueryPlan
// suppression, zero-affected-rows filling, out-of-proto-range timestamp text
// conversion, and *pgconn.PgError detail extraction.
func TestQueryConnAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("reuses held session and forwards options", func(t *testing.T) {
		var seenOpts []omndriver.QueryOptions
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, opts omndriver.QueryOptions) ([]*omndriver.Result, error) {
				seenOpts = append(seenOpts, opts)
				return []*omndriver.Result{{
					Statement: script,
					Plan:      &omndriver.Plan{Format: "JSON"},
				}}, nil
			},
		}
		stub := &stubPGDriver{session: sess}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: stub}}

		explainRes, err := driver.QueryConn(ctx, "SELECT 1", db.QueryContext{
			Schema:               "app",
			Limit:                50,
			Explain:              true,
			MaximumSQLResultSize: 1 << 20,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		})
		require.NoError(t, err)
		require.Nil(t, explainRes[0].GetQueryPlan(), "explain requests omit typed QueryPlan")

		selfExplainRes, err := driver.QueryConn(ctx, "EXPLAIN (FORMAT JSON) SELECT 1", db.QueryContext{
			MaximumSQLResultSize: 1 << 20,
		})
		require.NoError(t, err)
		require.Equal(t, v1pb.QueryOption_JSON, selfExplainRes[0].GetQueryPlan().GetFormat())
		require.Equal(t, 1, stub.sessionOpens, "QueryConn reuses one held session across calls")
		require.Equal(t, omndriver.QueryOptions{
			Schema:         "app",
			Limit:          50,
			Explain:        true,
			ExplainFormat:  "json",
			MaxResultBytes: 1 << 20,
		}, seenOpts[0])
		require.Empty(t, seenOpts[1].ExplainFormat)
	})

	t.Run("fills zero affected rows for statements without a count", func(t *testing.T) {
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, _ omndriver.QueryOptions) ([]*omndriver.Result, error) {
				return []*omndriver.Result{{Statement: script}}, nil
			},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

		results, err := driver.QueryConn(ctx, "CREATE TABLE counted (id int);", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, []string{"Affected Rows"}, results[0].GetColumnNames())
		require.Equal(t, int64(0), results[0].GetRows()[0].GetValues()[0].GetInt64Value())
		require.Equal(t, int64(1), results[0].GetRowsCount())
	})

	t.Run("rewrites out-of-proto-range timestamps as text", func(t *testing.T) {
		bcTime := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
		okTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, _ omndriver.QueryOptions) ([]*omndriver.Result, error) {
				return []*omndriver.Result{{
					Statement: script,
					Columns:   []omndriver.Column{{Name: "ts", TypeName: "TIMESTAMP"}, {Name: "ok", TypeName: "TIMESTAMP"}},
					Rows: [][]omndriver.Value{{
						{Kind: omndriver.Timestamp, Time: bcTime},
						{Kind: omndriver.Timestamp, Time: okTime},
					}},
				}}, nil
			},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

		results, err := driver.QueryConn(ctx, "SELECT ts, ok", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		values := results[0].GetRows()[0].GetValues()
		require.Equal(t, "0001-01-01 00:00:00 BC", values[0].GetStringValue())
		require.NotNil(t, values[1].GetTimestampValue())
	})

	t.Run("attaches PostgresError details from *pgconn.PgError", func(t *testing.T) {
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, _ omndriver.QueryOptions) ([]*omndriver.Result, error) {
				return []*omndriver.Result{{
					Statement: script,
					Err: &pgconn.PgError{
						Severity: "ERROR",
						Code:     "42P01",
						Message:  `relation "missing_table" does not exist`,
					},
				}}, nil
			},
		}
		driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

		results, err := driver.QueryConn(ctx, "SELECT * FROM missing_table;", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "42P01", results[0].GetPostgresError().GetCode())
		require.Contains(t, results[0].GetPostgresError().GetMessage(), "missing_table")
	})
}

func TestSyncInstanceSkipsNeonPostgresDatabase(t *testing.T) {
	ctx := context.Background()
	makeDriver := func(vendor storepb.Vendor) *Driver {
		return &Driver{
			OmniAdapter: util.OmniAdapter{
				Driver: &stubPGDriver{
					instance: &omndriver.Instance{
						Version: "16.0",
						Databases: []*metadatapb.DatabaseSchemaMetadata{
							{Name: "postgres"},
							{Name: "appdb"},
						},
					},
				},
				WithInstanceMetadata: true,
			},
			connectionCtx: db.ConnectionContext{Vendor: vendor},
		}
	}

	genericMeta, err := makeDriver(storepb.Vendor_VENDOR_UNSPECIFIED).SyncInstance(ctx)
	require.NoError(t, err)
	require.Len(t, genericMeta.Databases, 2)

	neonMeta, err := makeDriver(storepb.Vendor_NEON).SyncInstance(ctx)
	require.NoError(t, err)
	require.Len(t, neonMeta.Databases, 1)
	require.Equal(t, "appdb", neonMeta.Databases[0].Name)
}

func TestDumpOmitsBackupSchema(t *testing.T) {
	backupSchema := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	stub := &stubPGDriver{}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: stub}}

	input := &metadatapb.DatabaseSchemaMetadata{
		Name: "appdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "public"},
			{Name: backupSchema},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, driver.Dump(context.Background(), &buf, input))
	require.Len(t, stub.dumpedMeta.Schemas, 1)
	require.Equal(t, "public", stub.dumpedMeta.Schemas[0].Name)
	require.Len(t, input.Schemas, 2, "Dump does not mutate the caller's metadata")
}

func TestTaskRunSessionConversion(t *testing.T) {
	start := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	queryStart := start.Add(2 * time.Second)
	state := "active"
	got := taskRunSession(&pgdriver.BackendSession{
		PID:             "100",
		BlockedByPIDs:   []string{"50"},
		Query:           "SELECT pg_sleep(5);",
		State:           &state,
		ApplicationName: "bytebase-taskrun-42",
		BackendStart:    start,
		QueryStart:      &queryStart,
	})
	require.Equal(t, "100", got.GetPid())
	require.Equal(t, []string{"50"}, got.GetBlockedByPids())
	require.Equal(t, "SELECT pg_sleep(5);", got.GetQuery())
	require.Equal(t, "active", got.GetState())
	require.Equal(t, "bytebase-taskrun-42", got.GetApplicationName())
	require.Nil(t, got.GetXactStart())
	require.Equal(t, queryStart, got.GetQueryStart().AsTime())
}
