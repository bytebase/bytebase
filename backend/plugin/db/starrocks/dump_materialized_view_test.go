package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// BYT-9689: the dump path re-tags materialized-view rows so it emits
// SHOW CREATE MATERIALIZED VIEW for them. Doris reports MVs as 'BASE TABLE',
// StarRocks as 'VIEW'; both must become 'MATERIALIZED VIEW' when the name is a known
// MV, while regular tables and views are left untouched.
func TestMarkMaterializedViews(t *testing.T) {
	tables := []*TableSchema{
		{Name: "mv_doris", TableType: baseTableType},     // Doris MV reported as BASE TABLE
		{Name: "mv_starrocks", TableType: viewTableType}, // StarRocks MV reported as VIEW
		{Name: "v_regular", TableType: viewTableType},    // regular view (not an MV)
		{Name: "t_plain", TableType: baseTableType},      // plain table
	}
	markMaterializedViews(tables, map[string]bool{"mv_doris": true, "mv_starrocks": true})

	require.Equal(t, materializedViewType, tables[0].TableType)
	require.Equal(t, materializedViewType, tables[1].TableType)
	require.Equal(t, viewTableType, tables[2].TableType, "regular view must stay a view")
	require.Equal(t, baseTableType, tables[3].TableType, "plain table must stay a table")
}

// BYT-9689: the dump emits a temporary regular-view placeholder for every view AND
// materialized view (getTemporaryMaterializedView creates a CREATE VIEW), then drops it
// before emitting the real definition. The drop must therefore be DROP VIEW even for an
// MV — DROP MATERIALIZED VIEW would not match the placeholder, leaving it in the shared
// namespace so the real CREATE MATERIALIZED VIEW collides with it on replay.
func TestMaterializedViewPlaceholderIsDroppedAsView(t *testing.T) {
	placeholder := getTemporaryMaterializedView("mv1", []string{"a"})
	require.Contains(t, placeholder, "CREATE VIEW `mv1`")
	require.NotContains(t, placeholder, "CREATE MATERIALIZED VIEW")

	require.Equal(t, "DROP VIEW IF EXISTS `mv1`;\n", dropPlaceholderStmt("mv1"))
}

// BYT-9689: information_schema.tables is unordered, so an MV can be listed before a view
// it is defined on. A materialized view is materialized from its sources at CREATE time
// (unlike a lazy regular view), so all regular views must be emitted — and on restore,
// created real — before any materialized view; otherwise the MV would materialize from
// the temporary SELECT 1 placeholder. finalEmitOrder partitions views ahead of MVs while
// preserving each group's relative order, and drops base tables (emitted elsewhere).
func TestFinalEmitOrder_ViewsBeforeMaterializedViews(t *testing.T) {
	tables := []*TableSchema{
		{Name: "mv1", TableType: materializedViewType},
		{Name: "v1", TableType: viewTableType},
		{Name: "t1", TableType: baseTableType},
		{Name: "mv2", TableType: materializedViewType},
		{Name: "v2", TableType: viewTableType},
	}
	var names []string
	for _, tbl := range finalEmitOrder(tables) {
		names = append(names, tbl.Name)
	}
	require.Equal(t, []string{"v1", "v2", "mv1", "mv2"}, names)
}
