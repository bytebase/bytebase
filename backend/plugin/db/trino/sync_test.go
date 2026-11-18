package trino

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestSyncInstanceAndDatabaseAgainstALiveServer pins what the omni syncer reads:
// the catalogs as databases, and one catalog's schemas, tables, columns and
// views. Two shapes differ from the old driver: information_schema is no longer
// listed as a schema, and views are now read and dumped.
func TestSyncInstanceAndDatabaseAgainstALiveServer(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedTrinoContainer(t)
	driver := openTestDriver(t, &storepb.DataSource{
		Host:     c.GetHost(),
		Port:     c.GetPort(),
		Username: c.GetUsername(),
		Database: "memory",
	})

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "482", instance.Version)
	var catalogs []string
	for _, database := range instance.Databases {
		catalogs = append(catalogs, database.Name)
	}
	require.Contains(t, catalogs, "memory")

	_, err = driver.Execute(ctx, "CREATE TABLE memory.default.cutover_sync_t (id BIGINT, name VARCHAR(10));\n"+
		"CREATE VIEW memory.default.cutover_sync_v AS SELECT id FROM memory.default.cutover_sync_t", db.ExecuteOptions{})
	require.NoError(t, err)

	schema, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "memory", schema.Name)

	var schemas []string
	var table *struct {
		columns []string
	}
	var view string
	for _, s := range schema.Schemas {
		schemas = append(schemas, s.Name)
		for _, t := range s.Tables {
			if t.Name == "cutover_sync_t" {
				var columns []string
				for _, column := range t.Columns {
					columns = append(columns, column.Name)
				}
				table = &struct{ columns []string }{columns: columns}
			}
		}
		for _, v := range s.Views {
			if v.Name == "cutover_sync_v" {
				view = v.Definition
			}
		}
	}
	// Known difference: the old driver listed information_schema and read no views.
	require.Equal(t, []string{"default"}, schemas)
	require.Equal(t, []string{"id", "name"}, table.columns)
	require.NotEmpty(t, view)

	var dump strings.Builder
	require.NoError(t, driver.Dump(ctx, &dump, schema))
	// Known difference: the omni dump writes CREATE SCHEMA and views; the old
	// driver wrote tables only.
	require.Contains(t, dump.String(), `CREATE SCHEMA IF NOT EXISTS "default";`)
	require.Contains(t, dump.String(), `CREATE TABLE IF NOT EXISTS "default"."cutover_sync_t"`)
	require.Contains(t, dump.String(), `CREATE OR REPLACE VIEW "default"."cutover_sync_v"`)
}
