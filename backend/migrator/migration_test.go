package migrator_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
)

func TestMain(m *testing.M) { testcontainer.Main(m) }

func TestMigration3_24_1_MSSQLVerifyTLSOff(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO instance (resource_id, metadata) VALUES
			('mssql-verify', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true,"sslCa":"ca"},{"id":"ro","type":"READ_ONLY","useSsl":true,"verifyTlsCertificate":true}]}'),
			('mssql-verify-without-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}'),
			('mssql-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}'),
			('mssql-no-data-sources', '{"engine":"MSSQL","dataSources":[]}'),
			('postgres-verify', '{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true}]}');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.24/0001##mssql_verify_tls_off.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for resourceID, want := range map[string]string{
		"mssql-verify":             `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"sslCa":"ca"},{"id":"ro","type":"READ_ONLY","useSsl":true}]}`,
		"mssql-verify-without-tls": `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}`,
		"mssql-tls":                `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}`,
		"mssql-no-data-sources":    `{"engine":"MSSQL","dataSources":[]}`,
		"postgres-verify":          `{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true}]}`,
	} {
		var got string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&got))
		require.JSONEq(t, want, got, resourceID)
	}
}

func TestMigration3_24_2_OracleTLSOff(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO instance (resource_id, metadata) VALUES
			('oracle-tls', '{"engine":"ORACLE","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true,"obfuscatedSslCa":"ca","obfuscatedSslCert":"cert","obfuscatedSslKey":"key"},{"id":"ro","type":"READ_ONLY","useSsl":true,"obfuscatedSslCaPath":"ca-path","obfuscatedSslCertPath":"cert-path","obfuscatedSslKeyPath":"key-path","extraConnectionParameters":{"SSL":"TRUE","WALLET":"/opt/oracle/wallet"}}]}'),
			('oracle-tls-on-one', '{"engine":"ORACLE","dataSources":[{"id":"admin","useSsl":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true,"obfuscatedSslCa":"ca"}]}'),
			('oracle-tls-off-with-ca', '{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true,"obfuscatedSslCa":"ca"}]}'),
			('oracle-no-tls', '{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}'),
			('oracle-no-data-sources', '{"engine":"ORACLE","dataSources":[]}'),
			('oracle-without-data-sources', '{"engine":"ORACLE"}'),
			('oracle-null-data-sources', '{"engine":"ORACLE","dataSources":null}'),
			('postgres-tls', '{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"obfuscatedSslCa":"ca"}]}'),
			('mssql-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.24/0002##oracle_tls_off.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for resourceID, want := range map[string]string{
		"oracle-tls":                  `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true},{"id":"ro","type":"READ_ONLY","extraConnectionParameters":{"SSL":"TRUE","WALLET":"/opt/oracle/wallet"}}]}`,
		"oracle-tls-on-one":           `{"engine":"ORACLE","dataSources":[{"id":"admin"},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}`,
		"oracle-tls-off-with-ca":      `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}`,
		"oracle-no-tls":               `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}`,
		"oracle-no-data-sources":      `{"engine":"ORACLE","dataSources":[]}`,
		"oracle-without-data-sources": `{"engine":"ORACLE"}`,
		"oracle-null-data-sources":    `{"engine":"ORACLE","dataSources":null}`,
		"postgres-tls":                `{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"obfuscatedSslCa":"ca"}]}`,
		"mssql-tls":                   `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}`,
	} {
		var got string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&got))
		require.JSONEq(t, want, got, resourceID)
	}
}
