package drivers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPGConfigKeepsEachValueWhole(t *testing.T) {
	open := func(edit func(*storepb.DataSource)) (db.Driver, error) {
		dataSource := &storepb.DataSource{Username: "dba", Host: "db.example.com", Port: "5432"}
		edit(dataSource)
		return newDriver(pgSpec).Open(context.Background(), storepb.Engine_POSTGRES, db.ConnectionConfig{
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

func TestPGRDSIAMBuildsATokenForTheWholeUser(t *testing.T) {
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
	// The token is the credential; TLS stays pgx's default negotiation.
	require.Nil(t, cfg.TLS)
}

func TestPGTLSConfigBindsEveryHost(t *testing.T) {
	hosts := []string{"pg-1.example.com", "pg-2.example.com", "pg-3.example.com"}
	caPEM, serverCertDERByHost := generatePGCAAndServerCertificates(t, hosts)
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
	certPEM, keyPEM := generatePGClientCertificatePEM(t)
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

func generatePGClientCertificatePEM(t *testing.T) (string, string) {
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

func generatePGCAAndServerCertificates(t *testing.T, hosts []string) (string, map[string][]byte) {
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

const pgCloudSQLTestInstance = "example-project:us-central1:example-instance"

// pgCloudSQLIAMConnectionConfig returns a Google Cloud SQL IAM data source that
// cloudsqlconn accepts offline. The credential is an authorized_user document so
// that building the connector needs neither a key nor a network round trip; it
// is never good enough to dial with.
func pgCloudSQLIAMConnectionConfig(instance string) db.ConnectionConfig {
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

// The dialer is held only by the adapter's closers and by the pg driver's
// Config.Dial, so a closed dialer shows as the pg driver's dial failing with
// cloudsqlconn.ErrDialerClosed while its own pool is still open.
func TestPGCloudSQLIAMDialerClosesWithDriver(t *testing.T) {
	ctx := context.Background()
	config := pgCloudSQLIAMConnectionConfig(pgCloudSQLTestInstance)

	opened, err := newDriver(pgSpec).Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	require.NoError(t, opened.Close(ctx))

	d := newDriver(pgSpec)
	impl, err := pgSpec.open(ctx, d, config)
	require.NoError(t, err)
	t.Cleanup(func() { _ = impl.Close() })
	require.NoError(t, d.Close(ctx))

	// The connector keeps refreshing its ephemeral certificate until it is
	// closed, so a driver that leaves it open leaks it for the life of the
	// process, once per connection opened.
	require.ErrorIs(t, impl.Ping(ctx), cloudsqlconn.ErrDialerClosed)
}

func TestPGCloudSQLIAMDialerClosesWhenOpenFails(t *testing.T) {
	ctx := context.Background()

	config := pgCloudSQLIAMConnectionConfig(pgCloudSQLTestInstance)
	// pgx refuses a port it cannot parse, so this fails Open after the connector
	// has been built.
	config.DataSource.ExtraConnectionParameters = map[string]string{"port": "not-a-port"}
	_, err := newDriver(pgSpec).Open(ctx, storepb.Engine_POSTGRES, config)
	require.Error(t, err)

	// A driver whose Open fails is discarded without a Close, so Open has to
	// close the connector itself. A failure after the spec built the connector
	// is forced here, keeping the pg driver to observe the dialer through.
	var impl odriver.Driver
	d := newDriver(pgSpec)
	err = d.OpenWith(ctx, pgSpec.adapter, func(*util.OmniAdapter) (odriver.Driver, error) {
		opened, err := pgSpec.open(ctx, d, pgCloudSQLIAMConnectionConfig(pgCloudSQLTestInstance))
		require.NoError(t, err)
		impl = opened
		return nil, errors.New("open failed after the connector was built")
	})
	require.Error(t, err)
	t.Cleanup(func() { _ = impl.Close() })
	require.ErrorIs(t, impl.Ping(ctx), cloudsqlconn.ErrDialerClosed)
}

type stubPGDriver struct {
	sessionOpens int
	session      odriver.Session
	instance     *odriver.Instance
	dumpedMeta   *metadatapb.DatabaseSchemaMetadata
}

func (*stubPGDriver) Ping(context.Context) error { return nil }
func (*stubPGDriver) Close() error               { return nil }
func (d *stubPGDriver) Session(context.Context) (odriver.Session, error) {
	d.sessionOpens++
	return d.session, nil
}
func (d *stubPGDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
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
	queryFn func(context.Context, string, odriver.QueryOptions) ([]*odriver.Result, error)
}

func (s *stubPGSession) Query(ctx context.Context, script string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	return s.queryFn(ctx, script, opts)
}
func (*stubPGSession) Close() error { return nil }

// TestPGQueryConnAdapterSeam pins PG-specific adapter transformations in
// QueryConn: held session reuse, lowercase ExplainFormat mapping and QueryPlan
// suppression, zero-affected-rows filling, out-of-proto-range timestamp text
// conversion, and *pgconn.PgError detail extraction.
func TestPGQueryConnAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("reuses held session and forwards options", func(t *testing.T) {
		var seenOpts []odriver.QueryOptions
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
				seenOpts = append(seenOpts, opts)
				return []*odriver.Result{{
					Statement: script,
					Plan:      &odriver.Plan{Format: "JSON"},
				}}, nil
			},
		}
		stub := &stubPGDriver{session: sess}
		driver := &Driver{spec: pgSpec, OmniAdapter: util.OmniAdapter{Driver: stub}}

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
		require.Equal(t, odriver.QueryOptions{
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
			queryFn: func(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				return []*odriver.Result{{Statement: script}}, nil
			},
		}
		driver := &Driver{spec: pgSpec, OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

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
			queryFn: func(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				return []*odriver.Result{{
					Statement: script,
					Columns:   []odriver.Column{{Name: "ts", TypeName: "TIMESTAMP"}, {Name: "ok", TypeName: "TIMESTAMP"}},
					Rows: [][]odriver.Value{{
						{Kind: odriver.Timestamp, Time: bcTime},
						{Kind: odriver.Timestamp, Time: okTime},
					}},
				}}, nil
			},
		}
		driver := &Driver{spec: pgSpec, OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

		results, err := driver.QueryConn(ctx, "SELECT ts, ok", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		values := results[0].GetRows()[0].GetValues()
		require.Equal(t, "0001-01-01 00:00:00 BC", values[0].GetStringValue())
		require.NotNil(t, values[1].GetTimestampValue())
	})

	t.Run("attaches PostgresError details from *pgconn.PgError", func(t *testing.T) {
		sess := &stubPGSession{
			queryFn: func(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				return []*odriver.Result{{
					Statement: script,
					Err: &pgconn.PgError{
						Severity: "ERROR",
						Code:     "42P01",
						Message:  `relation "missing_table" does not exist`,
					},
				}}, nil
			},
		}
		driver := &Driver{spec: pgSpec, OmniAdapter: util.OmniAdapter{Driver: &stubPGDriver{session: sess}}}

		results, err := driver.QueryConn(ctx, "SELECT * FROM missing_table;", db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "42P01", results[0].GetPostgresError().GetCode())
		require.Contains(t, results[0].GetPostgresError().GetMessage(), "missing_table")
	})
}

func TestPGSyncInstanceSkipsNeonPostgresDatabase(t *testing.T) {
	ctx := context.Background()
	makeDriver := func(vendor storepb.Vendor) *Driver {
		return &Driver{
			spec: pgSpec,
			OmniAdapter: util.OmniAdapter{
				Driver: &stubPGDriver{
					instance: &odriver.Instance{
						Version: "16.0",
						Databases: []*metadatapb.DatabaseSchemaMetadata{
							{Name: "postgres"},
							{Name: "appdb"},
						},
					},
				},
				WithInstanceMetadata: true,
			},
			config: db.ConnectionConfig{ConnectionContext: db.ConnectionContext{Vendor: vendor}},
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

func TestPGDumpOmitsBackupSchema(t *testing.T) {
	backupSchema := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	stub := &stubPGDriver{}
	driver := &Driver{spec: pgSpec, OmniAdapter: util.OmniAdapter{Driver: stub}}

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

func TestPGTaskRunSessionConversion(t *testing.T) {
	start := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	queryStart := start.Add(2 * time.Second)
	state := "active"
	got := pgSession(&pgdriver.BackendSession{
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

func TestPGQueryConnStopsServerQueryWhenContextEnds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sshTunnel bool
		tls       bool
		// timeout is the context deadline; zero cancels the context instead.
		timeout time.Duration
		wantErr string
	}{
		{name: "direct, canceled", wantErr: "context canceled"},
		{name: "direct, deadline exceeded", timeout: 2 * time.Second, wantErr: "context deadline exceeded"},
		{name: "ssh tunnel, canceled", sshTunnel: true, wantErr: "context canceled"},
		{name: "ssh tunnel, deadline exceeded", sshTunnel: true, timeout: 2 * time.Second, wantErr: "context deadline exceeded"},
		{name: "ssh tunnel with TLS, canceled", sshTunnel: true, tls: true, wantErr: "context canceled"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var pgContainer *testcontainer.Container
			var dbName string
			var rawDB *sql.DB
			if tc.tls {
				pgContainer = testcontainer.SharedTLSPgContainer(t)
				dbName, rawDB = testcontainer.NewTLSPgDatabase(t)
			} else {
				pgContainer = testcontainer.SharedPgContainer(t)
				dbName, rawDB = testcontainer.NewPgDatabase(t)
			}
			dataSource := &storepb.DataSource{
				Host:     pgContainer.GetHost(),
				Port:     pgContainer.GetPort(),
				Username: "postgres",
				UseSsl:   tc.tls,
			}
			if tc.sshTunnel {
				sshServer := testcontainer.NewSSHServer(t)
				dataSource.SshHost = sshServer.GetHost()
				dataSource.SshPort = sshServer.GetPort()
				dataSource.SshUser = sshServer.GetUser()
				dataSource.SshPrivateKey = sshServer.GetPrivateKey()
			}

			background := context.Background()
			driver, err := newDriver(pgSpec).Open(background, storepb.Engine_POSTGRES, db.ConnectionConfig{
				DataSource:        dataSource,
				Password:          "root-password",
				ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = driver.Close(background) })

			// Runs first, so a query that outlived its context cannot hold up the
			// cleanup below.
			t.Cleanup(func() {
				_, err := rawDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
				require.NoError(t, err)
			})

			ctx, cancel := context.WithCancel(background)
			if tc.timeout > 0 {
				ctx, cancel = context.WithTimeout(background, tc.timeout)
			}
			defer cancel()

			done := make(chan []*v1pb.QueryResult, 1)
			go func() {
				results, err := driver.QueryConn(ctx, "SELECT pg_sleep(30);", db.QueryContext{MaximumSQLResultSize: 1 << 30})
				if err != nil {
					results = []*v1pb.QueryResult{{Error: err.Error()}}
				}
				done <- results
			}()

			const runningQueries = `state = 'active' AND query LIKE '%pg_sleep%'`
			require.Eventually(t, func() bool { return countPGSessions(t, rawDB, dbName, runningQueries) == 1 }, 10*time.Second, 20*time.Millisecond,
				"the query never started on the server")
			if tc.timeout == 0 {
				cancel()
			}

			select {
			case results := <-done:
				require.Len(t, results, 1)
				require.Contains(t, results[0].GetError(), tc.wantErr)
			case <-time.After(10 * time.Second):
				require.FailNow(t, "QueryConn was still blocked 10s after the query started")
			}
			require.Eventually(t, func() bool { return countPGSessions(t, rawDB, dbName, runningQueries) == 0 }, 10*time.Second, 20*time.Millisecond,
				"the query kept running on the server after its context ended")
			require.Eventually(t, func() bool { return countPGSessions(t, rawDB, dbName, `application_name = 'bytebase'`) == 0 }, 10*time.Second, 20*time.Millisecond,
				"the canceled connection stayed open on the server")

			require.NoError(t, driver.Ping(background), "the driver must reconnect once the canceled connection is discarded")
		})
	}
}

// countPGSessions counts the sessions in the database, other than the one asking, that match the pg_stat_activity condition.
func countPGSessions(t *testing.T, rawDB *sql.DB, dbName, condition string) int {
	t.Helper()
	var count int
	err := rawDB.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid() AND `+condition, dbName).Scan(&count)
	require.NoError(t, err)
	return count
}

var errPGProbeStatement = errors.New("the statement was refused")

// recordingPGSession stands in for the omni session a probe runs on: it records
// each script the probe sends, in order, and answers the owner lookup and the
// probed statement.
type recordingPGSession struct {
	statements []string
	// statementErr is what the probed statement answered, when set.
	statementErr error
}

func (s *recordingPGSession) Query(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
	s.statements = append(s.statements, strings.Join(strings.Fields(script), " "))
	switch {
	case strings.Contains(script, "pg_roles"):
		return []*odriver.Result{{
			Columns: []odriver.Column{{Name: "rolname", TypeName: "NAME"}},
			Rows:    [][]odriver.Value{{{Kind: odriver.String, String: "app_owner"}}},
		}}, nil
	case strings.HasPrefix(strings.TrimSpace(script), "BEGIN"), strings.HasPrefix(strings.TrimSpace(script), "ROLLBACK"), strings.HasPrefix(strings.TrimSpace(script), "SET"):
		return []*odriver.Result{{}}, nil
	default:
		if s.statementErr != nil {
			//nolint:nilerr // a refused statement's error rides on its result, as the contract says.
			return []*odriver.Result{{Err: s.statementErr}}, nil
		}
		return []*odriver.Result{{
			Columns: []odriver.Column{{Name: "QUERY PLAN", TypeName: "TEXT"}},
			Rows:    [][]odriver.Value{{{Kind: odriver.String, String: "plan"}}},
		}}, nil
	}
}

func (*recordingPGSession) Close() error { return nil }

// probePGSession runs the driver's probe, which opens its one session on the
// stub driver.
func probePGSession(t *testing.T, session odriver.Session, connectionCtx db.ConnectionContext, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	t.Helper()
	stub := &stubPGDriver{session: session}
	driver := &Driver{
		spec:        pgSpec,
		OmniAdapter: util.OmniAdapter{Driver: stub},
		config:      db.ConnectionConfig{ConnectionContext: connectionCtx},
	}
	prober := db.ProberOf(driver)
	require.NotNil(t, prober)
	return prober.Probe(context.Background(), pCtx, statement)
}

// TestPGProbeTenantMode proves the probe reads the database owner and sets the
// session role before the statement, in the order the tenant-mode prelude has
// always used, and rolls the transaction back — all on the one session it opened.
func TestPGProbeTenantMode(t *testing.T) {
	session := &recordingPGSession{}

	_, err := probePGSession(t, session, db.ConnectionContext{TenantMode: true},
		db.ProbeContext{PreExecutions: []string{"SET search_path = app"}},
		"EXPLAIN (FORMAT JSON) UPDATE t SET c = 1")
	require.NoError(t, err)

	require.Equal(t, []string{
		"SELECT u.rolname FROM pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid) WHERE d.datname = current_database();",
		"BEGIN;",
		"SET SESSION ROLE 'app_owner';",
		"SET search_path = app",
		"EXPLAIN (FORMAT JSON) UPDATE t SET c = 1",
		"ROLLBACK;",
	}, session.statements)
}

// TestPGProbeWithoutTenantMode proves a non-tenant probe runs the statement alone.
func TestPGProbeWithoutTenantMode(t *testing.T) {
	session := &recordingPGSession{}

	_, err := probePGSession(t, session, db.ConnectionContext{}, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, []string{"BEGIN;", "SELECT CURRENT_USER", "ROLLBACK;"}, session.statements)
}

// TestPGProbeRollsBackAfterAFailedStatement proves a refused statement still ends
// its transaction, so the session does not return to the pool inside one.
func TestPGProbeRollsBackAfterAFailedStatement(t *testing.T) {
	session := &recordingPGSession{statementErr: errPGProbeStatement}

	_, err := probePGSession(t, session, db.ConnectionContext{}, db.ProbeContext{}, "EXPLAIN INSERT INTO t VALUES (1)")
	require.ErrorIs(t, err, errPGProbeStatement)
	require.Equal(t, []string{"BEGIN;", "EXPLAIN INSERT INTO t VALUES (1)", "ROLLBACK;"}, session.statements)
}

// TestPGProbeAutocommitRefusesPreExecutions proves the contract's one refusal.
func TestPGProbeAutocommitRefusesPreExecutions(t *testing.T) {
	session := &recordingPGSession{}

	_, err := probePGSession(t, session, db.ConnectionContext{},
		db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET search_path = app"}}, "SELECT 1")
	require.ErrorContains(t, err, "cannot run outside a transaction")
	require.Empty(t, session.statements)
}

// TestPGProbeResultTypes proves a rule reads a text cell as a string, the shape the
// advisor rules type-assert.
func TestPGProbeResultTypes(t *testing.T) {
	session := &recordingPGSession{}

	result, err := probePGSession(t, session, db.ConnectionContext{}, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, []string{"QUERY PLAN"}, result.Columns)
	require.Equal(t, []string{"TEXT"}, result.ColumnTypes)
	require.Equal(t, [][]any{{"plan"}}, result.Rows)
}

type stubPGExplainReply struct {
	planJSON string
	errMsg   string
}

func startStubPGExplainServer(t *testing.T, handle func(sql string) stubPGExplainReply) (string, string, *[]string, *sync.Mutex) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	var queries []string
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		_ = ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveStubPGExplainConn(conn, done, &mu, &queries, handle)
		}
	}()

	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	return host, port, &queries, &mu
}

func serveStubPGExplainConn(c net.Conn, done <-chan struct{}, mu *sync.Mutex, queries *[]string, handle func(sql string) stubPGExplainReply) {
	defer func() { _ = c.Close() }()
	go func() {
		<-done
		_ = c.Close()
	}()

	backend := pgproto3.NewBackend(c, c)
	for {
		msg, err := backend.ReceiveStartupMessage()
		if err != nil {
			return
		}
		switch msg.(type) {
		case *pgproto3.SSLRequest:
			if _, err := c.Write([]byte{'N'}); err != nil {
				return
			}
		case *pgproto3.StartupMessage:
			backend.Send(&pgproto3.AuthenticationOk{})
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
			if err := backend.Flush(); err != nil {
				return
			}
			serveStubPGExplainLoop(backend, mu, queries, handle)
			return
		default:
			return
		}
	}
}

func serveStubPGExplainLoop(backend *pgproto3.Backend, mu *sync.Mutex, queries *[]string, handle func(sql string) stubPGExplainReply) {
	var curSQL string
	txStatus := byte('I')
	run := func(sql string) {
		trimmed := strings.TrimSpace(sql)
		mu.Lock()
		*queries = append(*queries, trimmed)
		mu.Unlock()
		switch upper := strings.ToUpper(trimmed); {
		case strings.HasPrefix(upper, "BEGIN"):
			txStatus = 'T'
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("BEGIN")})
		case strings.HasPrefix(upper, "ROLLBACK"):
			txStatus = 'I'
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("ROLLBACK")})
		case strings.HasPrefix(upper, "SET "):
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SET")})
		default:
			rep := handle(trimmed)
			if rep.errMsg != "" {
				if txStatus == 'T' {
					txStatus = 'E'
				}
				backend.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "42P01", Message: rep.errMsg})
				return
			}
			backend.Send(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{{Name: []byte("QUERY PLAN"), DataTypeOID: pgtype.TextOID, TypeModifier: -1}}})
			backend.Send(&pgproto3.DataRow{Values: [][]byte{[]byte(rep.planJSON)}})
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("EXPLAIN")})
		}
	}
	for {
		msg, err := backend.Receive()
		if err != nil {
			return
		}
		switch m := msg.(type) {
		case *pgproto3.Terminate:
			return
		case *pgproto3.Query:
			run(m.String)
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: txStatus})
			if err := backend.Flush(); err != nil {
				return
			}
		case *pgproto3.Parse:
			curSQL = m.Query
			backend.Send(&pgproto3.ParseComplete{})
		case *pgproto3.Describe:
			backend.Send(&pgproto3.ParameterDescription{})
			backend.Send(&pgproto3.NoData{})
		case *pgproto3.Bind:
			backend.Send(&pgproto3.BindComplete{})
		case *pgproto3.Execute:
			run(curSQL)
		case *pgproto3.Sync:
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: txStatus})
			if err := backend.Flush(); err != nil {
				return
			}
		default:
		}
	}
}

// TestPGCountAffectedRows pins the driver's row estimate, found through
// db.RowEstimator, including Bytebase's base.WithSearchPath compatibility with
// omni/pg/driver.
func TestPGCountAffectedRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	host, port, queries, mu := startStubPGExplainServer(t, func(sql string) stubPGExplainReply {
		switch {
		case strings.Contains(sql, "SELECT * FROM big"):
			return stubPGExplainReply{planJSON: `[{"Plan":{"Node Type":"Seq Scan","Plan Rows":10000}}]`}
		case strings.Contains(sql, "missing_table"):
			return stubPGExplainReply{errMsg: `relation "missing_table" does not exist`}
		case strings.Contains(sql, "only_in_app"):
			return stubPGExplainReply{planJSON: `[{"Plan":{"Node Type":"ModifyTable","Operation":"Delete","Plans":[{"Node Type":"Seq Scan","Parent Relationship":"Outer","Plan Rows":100}]}}]`}
		default:
			return stubPGExplainReply{planJSON: `[{"Plan":{"Node Type":"ModifyTable","Operation":"Delete","Plans":[{"Node Type":"Hash Join","Parent Relationship":"Outer","Plan Rows":1000}]}}]`}
		}
	})

	driver, err := newDriver(pgSpec).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                      host,
			Port:                      port,
			Username:                  "dba",
			ExtraConnectionParameters: map[string]string{"sslmode": "disable"},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, driver.Close(context.Background())) })

	estimator, ok := db.As[db.RowEstimator](driver)
	require.True(t, ok)

	rows, err := estimator.CountAffectedRows(ctx, "DELETE FROM big USING s WHERE big.s_id = s.id AND s.flag;")
	require.NoError(t, err)
	require.Equal(t, int64(1000), rows)

	_, err = estimator.CountAffectedRows(ctx, "SELECT * FROM big;")
	require.Error(t, err)

	_, err = estimator.CountAffectedRows(ctx, "DELETE FROM missing_table;")
	require.ErrorContains(t, err, "does not exist")

	mu.Lock()
	*queries = nil
	mu.Unlock()

	rows, err = estimator.CountAffectedRows(ctx, base.WithSearchPath("DELETE FROM only_in_app;", []string{"app"}))
	require.NoError(t, err)
	require.Equal(t, int64(100), rows)

	mu.Lock()
	recorded := append([]string(nil), (*queries)...)
	mu.Unlock()
	require.Equal(t, []string{
		"begin",
		`SET LOCAL search_path TO "app"`,
		"EXPLAIN (FORMAT JSON) DELETE FROM only_in_app;",
		"rollback",
	}, recorded)
}
