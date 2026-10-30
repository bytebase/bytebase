package oceanbase

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetDatabaseMetadata pins that OceanBase reaches this package's parser through
// the registry. What the parser extracts is pinned by the goldens in schema/mysql,
// which this package's copy mirrors.
func TestGetDatabaseMetadata(t *testing.T) {
	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_OCEANBASE, "CREATE TABLE `t` (`a` int NOT NULL, PRIMARY KEY (`a`));")
	require.NoError(t, err)

	tables := metadata.GetSchemas()[0].GetTables()
	require.Len(t, tables, 1)
	require.Equal(t, "t", tables[0].GetName())
	require.Len(t, tables[0].GetColumns(), 1)
}

// TestGetDatabaseMetadataSyncedView pins that this package parses the canonical form
// an engine writes back for a view over a derived table joined to a table, which is
// what Bytebase reads when it syncs a view. It needs omni's oceanbase/parser at #570
// or later.
func TestGetDatabaseMetadataSyncedView(t *testing.T) {
	const synced = "CREATE TABLE `j1` (`k1` int);\n" +
		"CREATE TABLE `j2` (`k1` int);\n" +
		"CREATE VIEW `v` AS select `d`.`k1` AS `k1` from ((select `j1`.`k1` AS `k1` from `j1`) `d` join `j2` on((`d`.`k1` = `j2`.`k1`)));\n"

	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_OCEANBASE, synced)
	require.NoError(t, err)

	views := metadata.GetSchemas()[0].GetViews()
	require.Len(t, views, 1)
	require.Equal(t, "v", views[0].GetName())
}
