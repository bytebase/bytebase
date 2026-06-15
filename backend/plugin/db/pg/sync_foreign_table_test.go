package pg

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// legacyListForeignTableQuery is the previous information_schema.foreign_tables-based
// query, kept as the oracle: for a privileged user the pg_catalog-based query must
// produce identical rows.
const legacyListForeignTableQuery = `SELECT
	foreign_table.foreign_table_schema,
	foreign_table.foreign_table_name,
	foreign_table.foreign_server_catalog,
	foreign_table.foreign_server_name
FROM information_schema.foreign_tables AS foreign_table
ORDER BY foreign_table.foreign_table_schema, foreign_table.foreign_table_name;`

const foreignTableSetupSQL = `
CREATE EXTENSION postgres_fdw;
CREATE SERVER loopback_server FOREIGN DATA WRAPPER postgres_fdw OPTIONS (host 'localhost', dbname 'postgres');
CREATE SCHEMA fdw_schema;
CREATE FOREIGN TABLE fdw_schema.remote_orders (
    id integer NOT NULL,
    note varchar(30)
) SERVER loopback_server OPTIONS (schema_name 'public', table_name 'orders');
CREATE FOREIGN TABLE public.remote_items (
    item_id bigint
) SERVER loopback_server OPTIONS (schema_name 'public', table_name 'items');
`

func queryForeignTableRows(t *testing.T, conn *sql.DB, query string) [][4]string {
	t.Helper()
	rows, err := conn.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	var result [][4]string
	for rows.Next() {
		var r [4]string
		require.NoError(t, rows.Scan(&r[0], &r[1], &r[2], &r[3]))
		result = append(result, r)
	}
	require.NoError(t, rows.Err())
	return result
}

// TestListForeignTableQueryMatchesInformationSchema verifies that the pg_catalog-based
// foreign table query produces exactly the same rows as the previous
// information_schema.foreign_tables-based query for a privileged user.
func TestListForeignTableQueryMatchesInformationSchema(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	_, err := pgDB.Exec(foreignTableSetupSQL)
	require.NoError(t, err)

	legacyRows := queryForeignTableRows(t, pgDB, legacyListForeignTableQuery)
	newRows := queryForeignTableRows(t, pgDB, getListForeignTableQuery())

	require.Len(t, legacyRows, 2)
	require.Equal(t, legacyRows, newRows)
}

// TestSyncForeignTablesWithoutTablePrivilege verifies that foreign table sync no longer
// depends on the sync user's privileges. information_schema.foreign_tables
// privilege-filters its rows, so a sync user without privileges on a foreign table used
// to silently lose the whole table from the synced metadata.
func TestSyncForeignTablesWithoutTablePrivilege(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	_, err := pgDB.Exec(foreignTableSetupSQL)
	require.NoError(t, err)

	// pg_read_all_stats lets the sync read pg_table_size/pg_indexes_size; it grants no
	// table or column privileges, so information_schema.foreign_tables still filters.
	_, err = pgDB.Exec(`
CREATE USER limited_user WITH LOGIN PASSWORD 'limited-password';
GRANT pg_read_all_stats TO limited_user;
`)
	require.NoError(t, err)

	limitedDB, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=limited_user password=limited-password database=postgres", pgContainer.GetHost(), pgContainer.GetPort()))
	require.NoError(t, err)
	defer limitedDB.Close()
	require.NoError(t, limitedDB.Ping())

	// Confirm the privilege filtering this fix works around: the unprivileged user sees
	// no foreign tables through information_schema.
	var count int
	require.NoError(t, limitedDB.QueryRow(`SELECT COUNT(*) FROM information_schema.foreign_tables`).Scan(&count))
	require.Zero(t, count)

	driver := &Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "limited_user",
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Database: "postgres",
		},
		Password: "limited-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  "postgres",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	pgDriver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	metadata, err := pgDriver.SyncDBSchema(ctx)
	require.NoError(t, err)

	var fdwSchema *storepb.SchemaMetadata
	for _, schema := range metadata.Schemas {
		if schema.Name == "fdw_schema" {
			fdwSchema = schema
			break
		}
	}
	require.NotNil(t, fdwSchema, "fdw_schema not found in synced metadata")
	require.Len(t, fdwSchema.ExternalTables, 1)

	remoteOrders := fdwSchema.ExternalTables[0]
	require.Equal(t, "remote_orders", remoteOrders.Name)
	require.Equal(t, "loopback_server", remoteOrders.ExternalServerName)
	require.Equal(t, "postgres", remoteOrders.ExternalDatabaseName)

	// Columns of foreign tables come from the privilege-independent column sync.
	require.Len(t, remoteOrders.Columns, 2)
	require.Equal(t, "id", remoteOrders.Columns[0].Name)
	require.Equal(t, "integer", remoteOrders.Columns[0].Type)
	require.False(t, remoteOrders.Columns[0].Nullable)
	require.Equal(t, "note", remoteOrders.Columns[1].Name)
	require.Equal(t, "character varying(30)", remoteOrders.Columns[1].Type)
}
