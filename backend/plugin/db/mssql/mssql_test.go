package mssql

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	odriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
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

// TestOpenAppliesTLSSettings runs the settings through the client against the
// pinned server, which presents a certificate that names localhost and no IP
// address.
func TestOpenAppliesTLSSettings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)
	signer, err := os.ReadFile(container.GetTLSCAPath())
	require.NoError(t, err)
	ca, otherCA := string(signer), string(newTestCA(t))

	for _, tc := range []struct {
		name string
		// host is localhost, the name in the server certificate, or 127.0.0.1, which it lacks.
		host string
		ds   *storepb.DataSource
		// encryptOption is the session's sys.dm_exec_connections.encrypt_option.
		encryptOption string
		wantErr       string
	}{
		{name: "disabled ignores verify", host: "127.0.0.1", ds: &storepb.DataSource{VerifyTlsCertificate: true, SslCa: otherCA}, encryptOption: "FALSE"},
		{name: "tls accepts any certificate", host: "127.0.0.1", ds: &storepb.DataSource{UseSsl: true, SslCa: otherCA}, encryptOption: "TRUE"},
		{name: "tls verified against the CA", host: "localhost", ds: &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: ca}, encryptOption: "TRUE"},
		{name: "a host missing from the certificate", host: "127.0.0.1", ds: &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: ca}, wantErr: "doesn't contain any IP SANs"},
		{name: "a CA that did not sign the certificate", host: "localhost", ds: &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: otherCA}, wantErr: "certificate signed by unknown authority"},
		{name: "system trust rejects a private CA", host: "localhost", ds: &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true}, wantErr: "tls: failed to verify certificate"},
		{
			name:    "extra parameters cannot turn verification off",
			host:    "127.0.0.1",
			ds:      &storepb.DataSource{UseSsl: true, VerifyTlsCertificate: true, SslCa: ca, ExtraConnectionParameters: map[string]string{"TrustServerCertificate": "true"}},
			wantErr: "doesn't contain any IP SANs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.ds.Username, tc.ds.Host, tc.ds.Port = container.GetUsername(), tc.host, container.GetPort()
			driver, err := (&Driver{}).Open(ctx, storepb.Engine_MSSQL, db.ConnectionConfig{DataSource: tc.ds, Password: container.GetPassword()})
			require.NoError(t, err)
			defer driver.Close(ctx)

			// The adapter holds its session, so the probe runs through QueryConn.
			results, err := driver.QueryConn(ctx, "SELECT encrypt_option FROM sys.dm_exec_connections WHERE session_id = @@SPID", db.QueryContext{MaximumSQLResultSize: 1 << 30})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Len(t, results[0].GetRows(), 1)
			require.Equal(t, tc.encryptOption, results[0].GetRows()[0].GetValues()[0].GetStringValue())
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

func TestQueryConnRecordsLimitedStatement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, "CREATE TABLE dbo.t (id INT PRIMARY KEY);\nINSERT INTO dbo.t VALUES (1), (2), (3);")

	// SET returns nothing, so the driver pads an empty result in for it.
	results, err := driver.QueryConn(ctx, "SET NOCOUNT OFF;\nSELECT TOP 3 id FROM dbo.t ORDER BY id;", db.QueryContext{
		Limit:                2,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "SET NOCOUNT OFF;", results[0].GetStatement())
	require.Equal(t, "\nSELECT TOP 2 id FROM dbo.t ORDER BY id;", results[1].GetStatement())
	require.Len(t, results[1].GetRows(), 2)
}

func TestQueryConnMatchesResultsToStatements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, `CREATE TABLE dbo.t (id INT PRIMARY KEY);
INSERT INTO dbo.t VALUES (1), (2), (3);
GO
CREATE PROCEDURE dbo.two_sets AS
BEGIN
	UPDATE dbo.t SET id = id;
	SELECT id FROM dbo.t ORDER BY id;
	SELECT 'second' AS s;
END`)

	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 30}
	describe := func(results []*v1pb.QueryResult) []string {
		var got []string
		for _, r := range results {
			line := fmt.Sprintf("%s %v %d", strings.TrimSpace(r.GetStatement()), r.GetColumnNames(), len(r.GetRows()))
			if r.GetError() != "" {
				line += " error"
			}
			got = append(got, line)
		}
		return got
	}

	// Statements that return nothing keep their places across GO, and the
	// procedure's second result set comes after every statement's result.
	results, err := driver.QueryConn(ctx, `EXEC dbo.two_sets;
DECLARE @x INT;
SELECT @x = id FROM dbo.t;
GO
DECLARE @y INT;
SELECT @y = id FROM dbo.t;
SELECT id FROM dbo.t WHERE id = 3;`, queryContext)
	require.NoError(t, err)
	require.Equal(t, []string{
		"EXEC dbo.two_sets; [id] 3",
		"DECLARE @x INT; [] 0",
		"SELECT @x = id FROM dbo.t; [] 0",
		"DECLARE @y INT; [] 0",
		"SELECT @y = id FROM dbo.t; [] 0",
		"SELECT id FROM dbo.t WHERE id = 3; [id] 1",
		"EXEC dbo.two_sets; [s] 1",
	}, describe(results))

	results, err = driver.QueryConn(ctx, "EXEC dbo.two_sets;", db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{"EXEC dbo.two_sets; [id] 2", "EXEC dbo.two_sets; [s] 1"}, describe(results))

	// The size limit covers the rows kept, not the ones the row limit cuts.
	results, err = driver.QueryConn(ctx, "EXEC('SELECT TOP 200 name FROM sys.all_objects');", db.QueryContext{Limit: 2, MaximumSQLResultSize: 2000})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].GetError())
	require.Len(t, results[0].GetRows(), 2)

	// Masking cannot trace the rows of a procedure call or an OUTPUT clause.
	results, err = driver.QueryConn(ctx, "EXEC dbo.two_sets;\nGO\nUPDATE dbo.t SET id = id OUTPUT inserted.id;\nSELECT id FROM dbo.t WHERE id = 1;", db.QueryContext{MaskingEnabled: true, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{
		"EXEC dbo.two_sets; [id] 0 error",
		"UPDATE dbo.t SET id = id OUTPUT inserted.id; [id] 0 error",
		"SELECT id FROM dbo.t WHERE id = 1; [id] 1",
		"EXEC dbo.two_sets; [s] 0 error",
	}, describe(results))
	require.Contains(t, results[0].GetError(), "data masking")

	_, err = driver.QueryConn(ctx, "EXEC dbo.two_sets;\nSELECT 1 AS one;", queryContext)
	require.ErrorContains(t, err, "cannot match the batch's result sets to its statements")

	_, err = driver.QueryConn(ctx, "SET NOEXEC ON;\nSELECT 1 AS one;", queryContext)
	require.EqualError(t, err, "SET NOEXEC is not supported")
	_, err = driver.QueryConn(ctx, "SET ANSI_NULLS, NOEXEC ON;\nDELETE FROM dbo.t;", queryContext)
	require.EqualError(t, err, "SET NOEXEC is not supported")

	_, err = driver.QueryConn(ctx, "SELECT 1 AS one;\nSELECT 1/0 AS x;", queryContext)
	require.ErrorContains(t, err, "Divide by zero error encountered")

	// A result set cut off at the size limit must not stall the rest of the batch.
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	results, err = driver.QueryConn(timeoutCtx, "SELECT TOP 100 name FROM sys.all_objects;\nSELECT 1 AS one;", db.QueryContext{MaximumSQLResultSize: 1})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Contains(t, results[0].GetError(), "exceeds max allowed output size")
	require.Len(t, results[1].GetRows(), 1)
}

// TestQueryConnValueShapes pins how a column's type reaches the API, which is
// what the grid renders. The empty binary cell is the one the driver this
// replaced reported as NULL.
func TestQueryConnValueShapes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))

	results, err := driver.QueryConn(ctx, `SELECT
	CAST(0x AS VARBINARY(10)) AS empty_bin,
	CAST(0x01 AS VARBINARY(10)) AS bin,
	CAST(1.25 AS DECIMAL(18,2)) AS dec,
	CAST(1 AS BIT) AS flag,
	CAST('03:04:05' AS TIME) AS t,
	CAST('2024-01-02' AS DATE) AS d,
	CAST('2024-01-02T03:04:05.123+02:00' AS DATETIMEOFFSET) AS dto,
	CAST('2024-01-02T03:04:05.12' AS DATETIME2(2)) AS dt2`, db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 1)
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
}

// TestQueryConnReportsAffectedRows pins the shape of a statement that returns
// only its count: one "Affected Rows" row, which the API shows, and RowsCount 1.
// The old driver left RowsCount at 0 for the same result.
func TestQueryConnReportsAffectedRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, "CREATE TABLE dbo.counted (id INT PRIMARY KEY);")

	results, err := driver.QueryConn(ctx, "INSERT INTO dbo.counted (id) VALUES (1);", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, []string{"Affected Rows"}, results[0].GetColumnNames())
	require.Len(t, results[0].GetRows(), 1)
	require.Equal(t, int64(1), results[0].GetRows()[0].GetValues()[0].GetInt64Value())
	require.Equal(t, int64(1), results[0].GetRowsCount())
}

// TestQueryConnKeepsOneSession pins the session the editor relies on: it keeps
// one driver for a stream and calls QueryConn once per request, so a session
// setting and a temporary table one request creates are still there for the
// next. The driver this replaced ran both on the editor's pooled connection.
func TestQueryConnKeepsOneSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)
	driver, database := newSyncTestDatabase(ctx, t, container)

	// The first request leaves session state behind.
	_, err := driver.QueryConn(ctx, "CREATE TABLE #session (id INT);\nINSERT INTO #session VALUES (7);\nSET CONTEXT_INFO 0x0102;", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)

	// The next request sees the temporary table and the context info.
	results, err := driver.QueryConn(ctx, "SELECT id FROM #session;\nSELECT CONTEXT_INFO() AS context;", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, int64(7), results[0].GetRows()[0].GetValues()[0].GetInt64Value())
	context := results[1].GetRows()[0].GetValues()[0].GetBytesValue()
	require.Len(t, context, 128)
	require.Equal(t, []byte{0x01, 0x02}, context[:2])

	// Another driver is another session, even on the same database.
	otherOpened := openMSSQL(ctx, t, container.GetHost(), container.GetPort(), database)
	results, err = otherOpened.QueryConn(ctx, "SELECT COUNT(*) AS n FROM #session;", db.QueryContext{MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Contains(t, results[0].GetError(), "#session")
}

type fakeMSSQLDriver struct {
	execScript string
	execOpts   odriver.ExecuteOptions
}

func (*fakeMSSQLDriver) Ping(context.Context) error { return nil }
func (*fakeMSSQLDriver) Close() error               { return nil }

func (f *fakeMSSQLDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	return 1, nil
}

func TestExecuteParsesTxnModeAndIgnoresIsolation(t *testing.T) {
	fake := &fakeMSSQLDriver{}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	rows, err := d.Execute(context.Background(), "-- txn-mode = off\n-- txn-isolation = READ COMMITTED\nINSERT INTO dbo.t VALUES (1);", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	require.Equal(t, "INSERT INTO dbo.t VALUES (1);", fake.execScript)
	require.Equal(t, odriver.AutoCommit, fake.execOpts.Transaction)
	require.Empty(t, fake.execOpts.Isolation)
	require.True(t, fake.execOpts.CreateDatabase)
	require.Equal(t, 2, fake.execOpts.MaxRetries)
}
