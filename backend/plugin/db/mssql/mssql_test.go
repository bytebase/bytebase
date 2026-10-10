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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/go-mssqldb/azuread"
	"github.com/microsoft/go-mssqldb/msdsn"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestConnectionDSNAppliesTLSSettings(t *testing.T) {
	dir := t.TempDir()
	writeCA := func(name string) (string, *x509.CertPool) {
		ca := newTestCA(t)
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, ca, 0o600))
		pool := x509.NewCertPool()
		require.True(t, pool.AppendCertsFromPEM(ca))
		return path, pool
	}
	sectionCA, sectionPool := writeCA("section.pem")
	extraCA, extraPool := writeCA("extra.pem")
	systemPool, err := x509.SystemCertPool()
	require.NoError(t, err)

	for _, tc := range []struct {
		name       string
		useSSL     bool
		verify     bool
		caFile     string
		extra      map[string]string
		encryption msdsn.Encryption
		// roots verify the server certificate; nil means any certificate is accepted.
		roots *x509.CertPool
	}{
		{name: "disabled", encryption: msdsn.EncryptionOff},
		{name: "disabled ignores verify", verify: true, encryption: msdsn.EncryptionOff},
		{name: "tls", useSSL: true, encryption: msdsn.EncryptionRequired},
		{name: "tls with a CA", useSSL: true, caFile: sectionCA, encryption: msdsn.EncryptionRequired},
		{name: "tls verified against the CA", useSSL: true, verify: true, caFile: sectionCA, encryption: msdsn.EncryptionRequired, roots: sectionPool},
		{name: "tls verified against system trust", useSSL: true, verify: true, encryption: msdsn.EncryptionRequired, roots: systemPool},

		{name: "extra parameters cannot turn verification off", useSSL: true, verify: true, caFile: sectionCA, extra: map[string]string{"TrustServerCertificate": "true"}, encryption: msdsn.EncryptionRequired, roots: sectionPool},
		{name: "extra parameters cannot make encryption optional", useSSL: true, extra: map[string]string{"encrypt": "false"}, encryption: msdsn.EncryptionRequired},
		{name: "extra parameters cannot disable encryption", useSSL: true, extra: map[string]string{"Encrypt": "DISABLE"}, encryption: msdsn.EncryptionRequired},
		{name: "extra parameters can require strict encryption", useSSL: true, extra: map[string]string{"Encrypt": "Strict"}, encryption: msdsn.EncryptionStrict, roots: systemPool},
		{name: "extra parameters can turn verification on", useSSL: true, caFile: sectionCA, extra: map[string]string{"trustservercertificate": "false"}, encryption: msdsn.EncryptionRequired, roots: sectionPool},
		{name: "the CA replaces a certificate parameter", useSSL: true, verify: true, caFile: sectionCA, extra: map[string]string{"Certificate": extraCA}, encryption: msdsn.EncryptionRequired, roots: sectionPool},
		{name: "a certificate parameter replaces system trust", useSSL: true, verify: true, extra: map[string]string{"certificate": extraCA}, encryption: msdsn.EncryptionRequired, roots: extraPool},

		{name: "disabled applies encrypt", extra: map[string]string{"encrypt": "true"}, encryption: msdsn.EncryptionRequired},
		{name: "disabled applies TrustServerCertificate", extra: map[string]string{"TrustServerCertificate": "false"}, encryption: msdsn.EncryptionOff, roots: systemPool},
		{name: "disabled applies encrypt=disable", extra: map[string]string{"encrypt": "disable"}, encryption: msdsn.EncryptionDisabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := &storepb.DataSource{
				Host:                      "db.example.com",
				Port:                      "1433",
				Username:                  "bytebase",
				UseSsl:                    tc.useSSL,
				VerifyTlsCertificate:      tc.verify,
				ExtraConnectionParameters: tc.extra,
			}
			driverName, dsn := connectionDSN(ds, "password", "", tc.caFile)
			require.Equal(t, "sqlserver", driverName)
			cfg, err := msdsn.Parse(dsn)
			require.NoError(t, err)
			require.Equal(t, tc.encryption, cfg.Encryption)
			if tc.encryption == msdsn.EncryptionDisabled {
				require.Nil(t, cfg.TLSConfig)
				return
			}
			require.Equal(t, tc.roots == nil, cfg.TLSConfig.InsecureSkipVerify)
			if tc.roots != nil {
				require.Equal(t, "db.example.com", cfg.TLSConfig.ServerName)
				require.True(t, tc.roots.Equal(cfg.TLSConfig.RootCAs), "verified against the wrong roots")
			}
		})
	}
}

func TestConnectionDSNMergesExtraParameters(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ds         *storepb.DataSource
		database   string
		driverName string
		// want holds the parameters go-mssqldb reads from the DSN.
		want map[string]string
	}{
		{
			name:       "defaults",
			ds:         &storepb.DataSource{Username: "bytebase"},
			database:   "db1",
			driverName: "sqlserver",
			want:       map[string]string{"app name": "bytebase", "tlsmin": "1.0", "database": "db1", "user id": "bytebase", "password": "password"},
		},
		{
			name:       "extra parameters override the app name and tlsmin",
			ds:         &storepb.DataSource{ExtraConnectionParameters: map[string]string{"App Name": "etl", "TLSMin": "1.2"}},
			driverName: "sqlserver",
			want:       map[string]string{"app name": "etl", "tlsmin": "1.2"},
		},
		{
			name:       "the database comes from Bytebase",
			ds:         &storepb.DataSource{ExtraConnectionParameters: map[string]string{"Database": "other"}},
			database:   "db1",
			driverName: "sqlserver",
			want:       map[string]string{"database": "db1"},
		},
		{
			name:       "extra parameters name the database when Bytebase has none",
			ds:         &storepb.DataSource{ExtraConnectionParameters: map[string]string{"database": "other"}},
			driverName: "sqlserver",
			want:       map[string]string{"database": "other"},
		},
		{
			name: "the Azure credential comes from Bytebase",
			ds: &storepb.DataSource{
				AuthenticationType:        storepb.DataSource_AZURE_IAM,
				IamExtension:              &storepb.DataSource_AzureCredential_{AzureCredential: &storepb.DataSource_AzureCredential{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}},
				ExtraConnectionParameters: map[string]string{"FedAuth": azuread.ActiveDirectoryMSI, "User ID": "other", "Password": "other"},
			},
			driverName: azuread.DriverName,
			want:       map[string]string{"fedauth": azuread.ActiveDirectoryServicePrincipal, "user id": "client@tenant", "password": "secret"},
		},
		{
			name:       "Azure without a credential uses the default chain",
			ds:         &storepb.DataSource{AuthenticationType: storepb.DataSource_AZURE_IAM},
			driverName: azuread.DriverName,
			want:       map[string]string{"fedauth": azuread.ActiveDirectoryDefault},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driverName, dsn := connectionDSN(tc.ds, "password", tc.database, "")
			require.Equal(t, tc.driverName, driverName)
			cfg, err := msdsn.Parse(dsn)
			require.NoError(t, err)
			for key, value := range tc.want {
				require.Equal(t, value, cfg.Parameters[key], key)
			}
		})
	}
}

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

			var encryptOption string
			err = driver.GetDB().QueryRowContext(ctx, "SELECT encrypt_option FROM sys.dm_exec_connections WHERE session_id = @@SPID").Scan(&encryptOption)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.encryptOption, encryptOption)
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

func TestShowplanStatistic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option *v1pb.QueryOption
		want   string
	}{
		{name: "no option", option: nil, want: "SHOWPLAN_ALL"},
		{name: "empty option", option: &v1pb.QueryOption{}, want: "SHOWPLAN_ALL"},
		{
			name:   "text",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT},
			want:   "SHOWPLAN_ALL",
		},
		{
			name:   "xml",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML},
			want:   "SHOWPLAN_XML",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, showplanStatistic(tc.option))
		})
	}
}

func TestQueryConnRecordsLimitedStatement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, "CREATE TABLE dbo.t (id INT PRIMARY KEY);\nINSERT INTO dbo.t VALUES (1), (2), (3);")

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// SET returns nothing, so the driver pads an empty result in for it.
	results, err := driver.QueryConn(ctx, conn, "SET NOCOUNT OFF;\nSELECT TOP 3 id FROM dbo.t ORDER BY id;", db.QueryContext{
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
	driver := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, `CREATE TABLE dbo.t (id INT PRIMARY KEY);
INSERT INTO dbo.t VALUES (1), (2), (3);
GO
CREATE PROCEDURE dbo.two_sets AS
BEGIN
	UPDATE dbo.t SET id = id;
	SELECT id FROM dbo.t ORDER BY id;
	SELECT 'second' AS s;
END`)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
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
	results, err := driver.QueryConn(ctx, conn, `EXEC dbo.two_sets;
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

	results, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;", db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{"EXEC dbo.two_sets; [id] 2", "EXEC dbo.two_sets; [s] 1"}, describe(results))

	// The size limit covers the rows kept, not the ones the row limit cuts.
	results, err = driver.QueryConn(ctx, conn, "EXEC('SELECT TOP 200 name FROM sys.all_objects');", db.QueryContext{Limit: 2, MaximumSQLResultSize: 2000})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].GetError())
	require.Len(t, results[0].GetRows(), 2)

	// Masking cannot trace the rows of a procedure call or an OUTPUT clause.
	results, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;\nGO\nUPDATE dbo.t SET id = id OUTPUT inserted.id;\nSELECT id FROM dbo.t WHERE id = 1;", db.QueryContext{MaskingEnabled: true, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{
		"EXEC dbo.two_sets; [id] 0 error",
		"UPDATE dbo.t SET id = id OUTPUT inserted.id; [id] 0 error",
		"SELECT id FROM dbo.t WHERE id = 1; [id] 1",
		"EXEC dbo.two_sets; [s] 0 error",
	}, describe(results))
	require.Contains(t, results[0].GetError(), "data masking")

	_, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;\nSELECT 1 AS one;", queryContext)
	require.ErrorContains(t, err, "cannot match the batch's result sets to its statements")

	_, err = driver.QueryConn(ctx, conn, "SET NOEXEC ON;\nSELECT 1 AS one;", queryContext)
	require.EqualError(t, err, "SET NOEXEC is not supported")

	_, err = driver.QueryConn(ctx, conn, "SELECT 1 AS one;\nSELECT 1/0 AS x;", queryContext)
	require.ErrorContains(t, err, "Divide by zero error encountered")

	// A result set cut off at the size limit must not stall the rest of the batch.
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	results, err = driver.QueryConn(timeoutCtx, conn, "SELECT TOP 100 name FROM sys.all_objects;\nSELECT 1 AS one;", db.QueryContext{MaximumSQLResultSize: 1})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Contains(t, results[0].GetError(), "exceeds max allowed output size")
	require.Len(t, results[1].GetRows(), 1)
}

func TestMatchResults(t *testing.T) {
	const (
		rows  = stmtTypeResultSetGenerating | stmtTypeRowCountGenerating
		count = stmtTypeRowCountGenerating
		none  = stmtTypeUnknown
		proc  = stmtTypeProcedure
	)
	// describe names a result by its column, or by its affected rows.
	describe := func(results []*v1pb.QueryResult) []string {
		var got []string
		for _, r := range results {
			name := strings.Join(r.GetColumnNames(), ",")
			if name == "Affected Rows" {
				name = fmt.Sprintf("affected %d", r.GetRows()[0].GetValues()[0].GetInt64Value())
			}
			got = append(got, r.GetStatement()+":"+name)
		}
		return got
	}
	for _, tc := range []struct {
		name      string
		types     []stmtType
		sets      []string
		counts    []int64
		want      []string
		wantExtra []string
		wantErr   bool
	}{
		{
			name:   "each statement gets its own result",
			types:  []stmtType{none, rows, count, rows},
			sets:   []string{"a", "b"},
			counts: []int64{3},
			want:   []string{"s0:", "s1:a", "s2:affected 3", "s3:b"},
		},
		{
			name:      "a procedure call takes every result set",
			types:     []stmtType{none, proc, count},
			sets:      []string{"a", "b", "c"},
			counts:    []int64{4},
			want:      []string{"s0:", "s1:a", "s2:affected 4"},
			wantExtra: []string{"s1:b", "s1:c"},
		},
		{
			name:  "a procedure call that returns nothing",
			types: []stmtType{proc, rows},
			sets:  []string{"a"},
			want:  []string{"s0:", "s1:a"},
		},
		{
			name:   "row counts that cannot be told apart are dropped",
			types:  []stmtType{count, proc, count},
			counts: []int64{1, 2, 3},
			want:   []string{"s0:", "s1:", "s2:"},
		},
		{
			name:    "a procedure call beside another statement that returns rows",
			types:   []stmtType{proc, rows},
			sets:    []string{"a", "b"},
			wantErr: true,
		},
		{
			name:    "two procedure calls",
			types:   []stmtType{proc, proc},
			sets:    []string{"a"},
			wantErr: true,
		},
		{
			name:    "a statement without its result set",
			types:   []stmtType{rows, rows},
			sets:    []string{"a"},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statements := make([]string, len(tc.types))
			for i := range statements {
				statements[i] = fmt.Sprintf("s%d", i)
			}
			var sets []*v1pb.QueryResult
			for _, column := range tc.sets {
				sets = append(sets, &v1pb.QueryResult{ColumnNames: []string{column}})
			}
			results, extra, err := matchResults(statements, tc.types, sets, tc.counts)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, describe(results))
			require.Equal(t, tc.wantExtra, describe(extra))
		})
	}
}
