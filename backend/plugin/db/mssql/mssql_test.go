package mssql

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestMSSQLConfigDefaults maps a data source with nothing set onto the contract
// config: the client's defaults, with the CA file absent.
func TestMSSQLConfigDefaults(t *testing.T) {
	cfg, caFile, err := mssqlConfig(db.ConnectionConfig{
		DataSource:        &storepb.DataSource{Host: "db.example.com", Port: "1433", Username: "bytebase"},
		Password:          "password",
		ConnectionContext: db.ConnectionContext{DatabaseName: "db1"},
	})
	require.NoError(t, err)
	require.Empty(t, caFile)
	require.Equal(t, "db.example.com", cfg.Host)
	require.Equal(t, "1433", cfg.Port)
	require.Equal(t, "db1", cfg.Database)
	require.Equal(t, "bytebase", cfg.User)
	require.Equal(t, "password", cfg.Password)
	require.Equal(t, "bytebase", cfg.ApplicationName)
	// The client accepts the server's certificate unless the data source asks
	// it to verify, as the driver this replaced did.
	require.Equal(t, map[string]string{"trustservercertificate": "true"}, cfg.Params)
	require.Nil(t, cfg.TokenProvider)
	require.Empty(t, cfg.EngineVersion)
}

// TestMSSQLConfigExtraParameters keeps the precedence the old driver gave the
// data source's own parameters: they override the app name, the login, the
// server and the port, but never the database.
func TestMSSQLConfigExtraParameters(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ds       *storepb.DataSource
		database string
		wantUser string
		wantPass string
		wantApp  string
		wantHost string
		wantPort string
		wantDB   string
		wantRest map[string]string
	}{
		{
			name: "extra parameters override the credentials",
			ds: &storepb.DataSource{Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"User ID": "other", "Password": "other",
			}},
			wantUser: "other", wantPass: "other", wantApp: "bytebase", wantRest: map[string]string{},
		},
		{
			name: "extra parameters override the app name",
			ds: &storepb.DataSource{Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"App Name": "etl",
			}},
			wantUser: "bytebase", wantPass: "password", wantApp: "etl", wantRest: map[string]string{},
		},
		{
			name: "the database comes from Bytebase",
			ds: &storepb.DataSource{Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"Database": "other",
			}},
			database: "db1", wantUser: "bytebase", wantPass: "password", wantApp: "bytebase", wantDB: "db1",
			wantRest: map[string]string{},
		},
		{
			name: "extra parameters name the database when Bytebase has none",
			ds: &storepb.DataSource{Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"database": "other",
			}},
			wantUser: "bytebase", wantPass: "password", wantApp: "bytebase", wantDB: "other",
			wantRest: map[string]string{},
		},
		{
			name: "extra parameters override the server and port",
			ds: &storepb.DataSource{Host: "db.example.com", Port: "1433", Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"Server": "other.example.com", "Port": "1444",
			}},
			wantUser: "bytebase", wantPass: "password", wantApp: "bytebase", wantHost: "other.example.com", wantPort: "1444",
			wantRest: map[string]string{},
		},
		{
			name: "extra parameters reach a named instance",
			ds: &storepb.DataSource{Host: "db.example.com", Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"Server": `other.example.com\SQLEXPRESS`,
			}},
			wantUser: "bytebase", wantPass: "password", wantApp: "bytebase", wantHost: `other.example.com\SQLEXPRESS`,
			wantRest: map[string]string{},
		},
		{
			name: "every other parameter reaches the client",
			ds: &storepb.DataSource{Username: "bytebase", ExtraConnectionParameters: map[string]string{
				"packet size": "8192", "TLSMin": "1.2",
			}},
			wantUser: "bytebase", wantPass: "password", wantApp: "bytebase",
			wantRest: map[string]string{"packet size": "8192", "tlsmin": "1.2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, caFile, err := mssqlConfig(db.ConnectionConfig{
				DataSource:        tc.ds,
				Password:          "password",
				ConnectionContext: db.ConnectionContext{DatabaseName: tc.database},
			})
			require.NoError(t, err)
			require.Empty(t, caFile)
			require.Equal(t, tc.wantUser, cfg.User)
			require.Equal(t, tc.wantPass, cfg.Password)
			require.Equal(t, tc.wantApp, cfg.ApplicationName)
			require.Equal(t, tc.wantHost, cfg.Host)
			require.Equal(t, tc.wantPort, cfg.Port)
			require.Equal(t, tc.wantDB, cfg.Database)
			// The client's TLS default is the only parameter left when the data
			// source names none.
			want := map[string]string{"trustservercertificate": "true"}
			for key, value := range tc.wantRest {
				want[key] = value
			}
			require.Equal(t, want, cfg.Params)
		})
	}
}

// TestMSSQLConfigTLS pins the parameters go-mssqldb reads the TLS settings
// from. TLS mode TLS sets a floor that the data source's parameters may raise
// but not lower, and the section's CA replaces a certificate parameter.
func TestMSSQLConfigTLS(t *testing.T) {
	caPEM := string(newTestCA(t))
	for _, tc := range []struct {
		name   string
		ds     *storepb.DataSource
		want   map[string]string
		wantCA bool
	}{
		{
			name: "verification off",
			ds:   &storepb.DataSource{UseSsl: true},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "true"},
		},
		{
			name: "verification off ignores verify",
			ds:   &storepb.DataSource{VerifyTlsCertificate: true},
			want: map[string]string{"trustservercertificate": "true"},
		},
		{
			name:   "the CA travels as a path",
			ds:     &storepb.DataSource{UseSsl: true, SslCa: caPEM},
			want:   map[string]string{"encrypt": "true", "trustservercertificate": "true"},
			wantCA: true,
		},
		{
			name:   "verified against the CA",
			ds:     &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: caPEM},
			want:   map[string]string{"encrypt": "true", "trustservercertificate": "false"},
			wantCA: true,
		},
		{
			name: "verified against system trust",
			ds:   &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "false"},
		},
		{
			name: "parameters cannot make encryption optional",
			ds:   &storepb.DataSource{UseSsl: true, ExtraConnectionParameters: map[string]string{"encrypt": "false"}},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "true"},
		},
		{
			name: "parameters can require strict encryption",
			ds:   &storepb.DataSource{UseSsl: true, ExtraConnectionParameters: map[string]string{"Encrypt": "Strict"}},
			want: map[string]string{"encrypt": "Strict", "trustservercertificate": "true"},
		},
		{
			name: "parameters cannot turn verification off",
			ds:   &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, ExtraConnectionParameters: map[string]string{"TrustServerCertificate": "true"}},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "false"},
		},
		{
			name: "parameters can turn verification on",
			ds:   &storepb.DataSource{UseSsl: true, ExtraConnectionParameters: map[string]string{"trustservercertificate": "false"}},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "false"},
		},
		{
			name: "the section CA replaces a certificate parameter",
			ds: &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: caPEM, ExtraConnectionParameters: map[string]string{
				"Certificate": "/tmp/other.pem",
			}},
			want:   map[string]string{"encrypt": "true", "trustservercertificate": "false"},
			wantCA: true,
		},
		{
			name: "TLS disabled applies an encrypt parameter",
			ds:   &storepb.DataSource{ExtraConnectionParameters: map[string]string{"encrypt": "true"}},
			want: map[string]string{"encrypt": "true", "trustservercertificate": "true"},
		},
		{
			name: "TLS disabled applies a trust parameter",
			ds:   &storepb.DataSource{ExtraConnectionParameters: map[string]string{"TrustServerCertificate": "false"}},
			want: map[string]string{"trustservercertificate": "false"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, caFile, err := mssqlConfig(db.ConnectionConfig{DataSource: tc.ds})
			require.NoError(t, err)
			if !tc.wantCA {
				require.Empty(t, caFile)
				require.Equal(t, tc.want, cfg.Params)
				return
			}
			t.Cleanup(func() { removeCAFile(caFile) })
			require.NotEmpty(t, caFile)
			written, err := os.ReadFile(caFile)
			require.NoError(t, err)
			require.Equal(t, tc.ds.GetSslCa(), string(written))
			want := tc.want
			want["certificate"] = caFile
			require.Equal(t, want, cfg.Params)
		})
	}
}

// TestMSSQLConfigAzureCredential: an Azure login is a token provider and no
// user, and the fedauth parameter the old driver wrote has no reader.
func TestMSSQLConfigAzureCredential(t *testing.T) {
	for _, tc := range []struct {
		name string
		ds   *storepb.DataSource
	}{
		{
			name: "a client secret",
			ds: &storepb.DataSource{
				Username:           "ignored",
				AuthenticationType: storepb.DataSource_AZURE_IAM,
				IamExtension: &storepb.DataSource_AzureCredential_{AzureCredential: &storepb.DataSource_AzureCredential{
					TenantId: "tenant", ClientId: "client", ClientSecret: "secret",
				}},
				ExtraConnectionParameters: map[string]string{"FedAuth": "ActiveDirectoryMSI", "User ID": "other", "Password": "other"},
			},
		},
		{
			name: "the default chain",
			ds:   &storepb.DataSource{AuthenticationType: storepb.DataSource_AZURE_IAM},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _, err := mssqlConfig(db.ConnectionConfig{DataSource: tc.ds, Password: "ignored"})
			require.NoError(t, err)
			require.Empty(t, cfg.User)
			require.Empty(t, cfg.Password)
			require.NotNil(t, cfg.TokenProvider)
			require.NotContains(t, cfg.Params, "fedauth")
			require.NotContains(t, cfg.Params, "user id")
			require.NotContains(t, cfg.Params, "password")
		})
	}

	// The non-Azure path keeps the login and carries no provider.
	cfg, _, err := mssqlConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{Username: "bytebase"}, Password: "password"})
	require.NoError(t, err)
	require.Equal(t, "bytebase", cfg.User)
	require.Equal(t, "password", cfg.Password)
	require.Nil(t, cfg.TokenProvider)
}

// recordingCredential records what the token provider asks it for.
type recordingCredential struct {
	scopes []string
	token  string
}

func (c *recordingCredential) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.scopes = opts.Scopes
	return azcore.AccessToken{Token: c.token}, nil
}

// TestAzureTokenProviderAsksForAzureSQL pins the audience: Azure SQL accepts a
// token only for its own scope.
func TestAzureTokenProviderAsksForAzureSQL(t *testing.T) {
	credential := &recordingCredential{token: "token"}
	provider := azureTokenProvider(credential)
	token, err := provider(context.Background())
	require.NoError(t, err)
	require.Equal(t, "token", token)
	require.Equal(t, []string{"https://database.windows.net/.default"}, credential.scopes)
}

func TestExplainFormat(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option *v1pb.QueryOption
		want   string
	}{
		{name: "no option", option: nil, want: "TEXT"},
		{name: "empty option", option: &v1pb.QueryOption{}, want: "TEXT"},
		{name: "text", option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT}, want: "TEXT"},
		{name: "xml", option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML}, want: "XML"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, util.OmniExplainFormat(tc.option.GetExplainFormat(), "TEXT", v1pb.QueryOption_XML))
		})
	}
}

// newTestCA returns a self-signed CA certificate in PEM.
func newTestCA(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Unrelated Test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func requireTable(t *testing.T, metadata *metadatapb.DatabaseSchemaMetadata, schemaName, tableName string) *metadatapb.TableMetadata {
	t.Helper()
	for _, schema := range metadata.Schemas {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			if table.Name == tableName {
				return table
			}
		}
	}
	require.FailNowf(t, "table not synced", "%s.%s", schemaName, tableName)
	return nil
}

type fakeMSSQLDriver struct {
	sessionOpens int
	sessions     []*fakeMSSQLSession

	execScript string
	execOpts   odriver.ExecuteOptions
	execFn     func(context.Context, string, odriver.ExecuteOptions) (int64, error)

	instance *odriver.Instance
	dumpDDL  string
}

func (*fakeMSSQLDriver) Ping(context.Context) error { return nil }
func (*fakeMSSQLDriver) Close() error               { return nil }

func (f *fakeMSSQLDriver) Session(context.Context) (odriver.Session, error) {
	f.sessionOpens++
	if len(f.sessions) > 0 {
		s := f.sessions[0]
		f.sessions = f.sessions[1:]
		return s, nil
	}
	return &fakeMSSQLSession{}, nil
}

func (f *fakeMSSQLDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	return f.instance, nil
}

func (*fakeMSSQLDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

func (f *fakeMSSQLDriver) Dump(_ context.Context, out io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(out, f.dumpDDL)
	return err
}

func (f *fakeMSSQLDriver) Execute(ctx context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if f.execFn != nil {
		return f.execFn(ctx, script, opts)
	}
	return 1, nil
}

type fakeMSSQLSession struct {
	gotOpts odriver.QueryOptions
	queryFn func(context.Context, string, odriver.QueryOptions) ([]*odriver.Result, error)
	closed  bool
}

func (s *fakeMSSQLSession) Query(ctx context.Context, statement string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotOpts = opts
	if s.queryFn != nil {
		return s.queryFn(ctx, statement, opts)
	}
	return []*odriver.Result{{Statement: statement}}, nil
}

func (s *fakeMSSQLSession) Close() error {
	s.closed = true
	return nil
}

// TestQueryConnAdapterSeam pins QueryConn's held session reuse, option
// forwarding, value shape conversion (including empty binary, timestamps,
// affected rows, truncation, and partial results on error), and session drop
// on cancellation.
func TestQueryConnAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("reuses one session per driver and converts value shapes", func(t *testing.T) {
		tz := time.FixedZone("UTC+2", 7200)
		dtoTime := time.Date(2024, 1, 2, 3, 4, 5, 123000000, tz)
		dt2Time := time.Date(2024, 1, 2, 3, 4, 5, 120000000, time.UTC)
		affected := int64(1)

		sess := &fakeMSSQLSession{
			queryFn: func(_ context.Context, statement string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
				if statement == "CREATE TABLE #session (id INT);" {
					return []*odriver.Result{{Statement: statement}}, nil
				}
				return []*odriver.Result{
					{
						Statement: statement,
						Columns: []odriver.Column{
							{Name: "empty_bin", TypeName: "VARBINARY"},
							{Name: "bin", TypeName: "VARBINARY"},
							{Name: "dec", TypeName: "DECIMAL"},
							{Name: "flag", TypeName: "BIT"},
							{Name: "t", TypeName: "TIME"},
							{Name: "d", TypeName: "DATE"},
							{Name: "dto", TypeName: "DATETIMEOFFSET"},
							{Name: "dt2", TypeName: "DATETIME2"},
						},
						Rows: [][]odriver.Value{{
							{Kind: odriver.Bytes, Bytes: []byte{}},
							{Kind: odriver.Bytes, Bytes: []byte{0x01}},
							{Kind: odriver.String, String: "1.25"},
							{Kind: odriver.Int64, Int64: 1},
							{Kind: odriver.String, String: "03:04:05"},
							{Kind: odriver.String, String: "2024-01-02"},
							{Kind: odriver.TimestampTZ, Time: dtoTime, Precision: 7},
							{Kind: odriver.Timestamp, Time: dt2Time, Precision: 2},
						}},
					},
					{
						Statement:    "INSERT INTO dbo.counted (id) VALUES (1);",
						AffectedRows: &affected,
					},
				}, nil
			},
		}
		fake := &fakeMSSQLDriver{sessions: []*fakeMSSQLSession{sess}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
		qctx := db.QueryContext{
			Limit:                2,
			Explain:              true,
			MaximumSQLResultSize: 1 << 20,
			MaskingEnabled:       true,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML},
		}

		_, err := d.QueryConn(ctx, "CREATE TABLE #session (id INT);", qctx)
		require.NoError(t, err)
		results, err := d.QueryConn(ctx, "SELECT ...", qctx)
		require.NoError(t, err)

		require.Equal(t, 1, fake.sessionOpens)
		require.Same(t, sess, d.Session())
		require.Equal(t, 2, sess.gotOpts.Limit)
		require.True(t, sess.gotOpts.Explain)
		require.Equal(t, "XML", sess.gotOpts.ExplainFormat)
		require.Equal(t, int64(1<<20), sess.gotOpts.MaxResultBytes)
		require.True(t, sess.gotOpts.MaskingEnabled)

		require.Len(t, results, 2)
		values := results[0].GetRows()[0].GetValues()
		require.Len(t, values, 8)
		require.Equal(t, []byte{}, values[0].GetBytesValue())
		require.Equal(t, []byte{0x01}, values[1].GetBytesValue())
		require.Equal(t, "1.25", values[2].GetStringValue())
		require.Equal(t, int64(1), values[3].GetInt64Value())
		require.Equal(t, "03:04:05", values[4].GetStringValue())
		require.Equal(t, "2024-01-02", values[5].GetStringValue())
		require.Equal(t, int32(7), values[6].GetTimestampTzValue().GetAccuracy())
		require.Equal(t, int32(7200), values[6].GetTimestampTzValue().GetOffset())
		require.Equal(t, int32(2), values[7].GetTimestampValue().GetAccuracy())

		require.Equal(t, []string{"Affected Rows"}, results[1].GetColumnNames())
		require.Len(t, results[1].GetRows(), 1)
		require.Equal(t, int64(1), results[1].GetRows()[0].GetValues()[0].GetInt64Value())
		require.Equal(t, int64(1), results[1].GetRowsCount())
	})

	t.Run("returns partial converted results alongside query error and drops session", func(t *testing.T) {
		batchErr := errors.New("cannot match the batch's result sets to its statements")
		sess := &fakeMSSQLSession{
			queryFn: func(context.Context, string, odriver.QueryOptions) ([]*odriver.Result, error) {
				return []*odriver.Result{
					{
						Statement: "SELECT 1;",
						Columns:   []odriver.Column{{Name: "one", TypeName: "INT"}},
						Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
						Truncated: true,
					},
				}, batchErr
			},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeMSSQLDriver{sessions: []*fakeMSSQLSession{sess}}}}

		results, err := d.QueryConn(ctx, "SELECT 1;\nGO\nEXEC dbo.two_sets;\nSELECT 1;", db.QueryContext{MaximumSQLResultSize: 1})
		require.ErrorIs(t, err, batchErr)
		require.True(t, sess.closed)
		require.Nil(t, d.Session())
		require.Len(t, results, 1)
		require.Contains(t, results[0].GetError(), "exceeds max allowed output size")
	})
}

func TestExecuteParsesTxnModeAndIgnoresIsolation(t *testing.T) {
	fake := &fakeMSSQLDriver{
		execFn: func(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(1, []int64{1}, nil)
			return 1, nil
		},
		instance: &odriver.Instance{Version: "16.0.4150.1 (2022)", Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "appdb"}}},
		dumpDDL:  "CREATE TABLE [dbo].[t] ([id] INT);",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	var logTypes []storepb.TaskRunLog_Type
	rows, err := d.Execute(context.Background(), "-- txn-mode = off\n-- txn-isolation = READ COMMITTED\nINSERT INTO dbo.t VALUES (1);", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logTypes = append(logTypes, entry.Type)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, "INSERT INTO dbo.t VALUES (1);", fake.execScript)
	require.Equal(t, odriver.AutoCommit, fake.execOpts.Transaction)
	require.Empty(t, fake.execOpts.Isolation)
	require.True(t, fake.execOpts.CreateDatabase)
	require.Equal(t, 2, fake.execOpts.MaxRetries)
	require.Equal(t, []storepb.TaskRunLog_Type{
		storepb.TaskRunLog_COMMAND_EXECUTE,
		storepb.TaskRunLog_COMMAND_RESPONSE,
	}, logTypes)

	inst, err := d.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Equal(t, "16.0.4150.1 (2022)", inst.Version)
	var dump bytes.Buffer
	require.NoError(t, d.Dump(context.Background(), &dump, &metadatapb.DatabaseSchemaMetadata{Name: "appdb"}))
	require.Contains(t, dump.String(), "CREATE TABLE [dbo].[t]")
}

func TestOpenDoesNotDialAndCleansUpCAFileOnClose(t *testing.T) {
	ctx := context.Background()
	d, err := (&Driver{}).Open(ctx, storepb.Engine_MSSQL, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                 "127.0.0.1",
			Port:                 "1433",
			Username:             "sa",
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                string(newTestCA(t)),
		},
		Password: "secret",
	})
	require.NoError(t, err)
	require.NoError(t, d.Close(ctx))
}
