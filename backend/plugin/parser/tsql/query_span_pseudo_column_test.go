package tsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestOmniQuerySpanPseudoColumns covers the downstream mapping for T-SQL
// pseudo-columns the parser accepts since the $-token omni bump:
//   - $IDENTITY / IDENTITYCOL map precisely to the in-scope table's
//     ColumnMetadata.IsIdentity column, so masking rules on the identity
//     column follow the pseudo-column reference.
//   - Graph pseudo-columns get table-level lineage (conservative direction:
//     $from_id/$to_id encode referenced node-row identity).
//   - $ROWGUID stays unresolved (fail-closed) until the catalog carries a
//     rowguid flag.
func TestOmniQuerySpanPseudoColumns(t *testing.T) {
	identSrc := base.ColumnResource{Database: "db", Schema: "dbo", Table: "ident_t", Column: "id"}

	t.Run("identity_bare", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT $IDENTITY FROM ident_t")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		// Engine-verified: the result-set header for SELECT $IDENTITY is the
		// real column name, not "$IDENTITY".
		require.Equal(t, "id", span.Results[0].Name)
		require.Equal(t, base.SourceColumnSet{identSrc: true}, span.Results[0].SourceColumns)
	})

	t.Run("identitycol_keyword", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT IDENTITYCOL FROM ident_t")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{identSrc: true}, span.Results[0].SourceColumns)
	})

	t.Run("identity_qualified_alias", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT x.$IDENTITY FROM ident_t AS x")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{identSrc: true}, span.Results[0].SourceColumns)
	})

	t.Run("identity_qualified_disambiguates", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(),
			"SELECT ident_t.$IDENTITY FROM ident_t, ident_t2")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{identSrc: true}, span.Results[0].SourceColumns)
	})

	t.Run("identity_in_predicate", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT payload FROM ident_t WHERE $IDENTITY = 5")
		require.NoError(t, err)
		require.Equal(t, base.SourceColumnSet{identSrc: true}, span.PredicateColumns)
	})

	t.Run("identity_ambiguous_errors", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT $IDENTITY FROM ident_t, ident_t2")
		require.ErrorContains(t, err, "ambiguous")
	})

	t.Run("identity_no_identity_column_errors", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT $IDENTITY FROM t")
		require.ErrorContains(t, err, "no identity column")
	})

	// Graph result Name stays the written form ("$node_id"): the engine's
	// actual header is the mangled internal column ($node_id_<hex>), which the
	// synced catalog cannot reproduce.
	t.Run("graph_table_level_lineage", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT $node_id FROM t")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{
			{Database: "db", Schema: "dbo", Table: "t", Column: "a"}: true,
			{Database: "db", Schema: "dbo", Table: "t", Column: "b"}: true,
			{Database: "db", Schema: "dbo", Table: "t", Column: "c"}: true,
		}, span.Results[0].SourceColumns)
	})

	t.Run("graph_qualified_two_tables", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT t2.$from_id FROM t, t2")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{
			{Database: "db", Schema: "dbo", Table: "t2", Column: "a"}: true,
			{Database: "db", Schema: "dbo", Table: "t2", Column: "b"}: true,
		}, span.Results[0].SourceColumns)
	})

	t.Run("graph_bare_ambiguous_errors", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT $node_id FROM t, t2")
		require.ErrorContains(t, err, "ambiguous")
	})

	t.Run("graph_empty_table_fails_closed", func(t *testing.T) {
		// Attribute-less edge tables have no catalog columns; empty lineage
		// would mean NoneMasker downstream, so this must error.
		q := newOmniTestExtractor(t, "db")
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT $edge_id FROM bare_edge")
		require.ErrorContains(t, err, "no columns in the catalog")
	})

	// Delimited identifiers are never pseudo-columns: [IDENTITYCOL] and
	// [$node_id] reference real columns with those names.
	t.Run("delimited_identifiers_are_real_columns", func(t *testing.T) {
		for _, sql := range []string{
			"SELECT [$node_id] FROM weird",
			"SELECT weird.[$node_id] FROM weird",
		} {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), sql)
			require.NoError(t, err, sql)
			require.Len(t, span.Results, 1)
			require.Equal(t, base.SourceColumnSet{
				{Database: "db", Schema: "dbo", Table: "weird", Column: "$node_id"}: true,
			}, span.Results[0].SourceColumns, sql)
		}
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT [IDENTITYCOL] FROM weird")
		require.NoError(t, err)
		require.Equal(t, base.SourceColumnSet{
			{Database: "db", Schema: "dbo", Table: "weird", Column: "IDENTITYCOL"}: true,
		}, span.Results[0].SourceColumns)
	})

	// A delimited TABLE with a bare column segment stays a pseudo-column:
	// [weird].$node_id is the graph reference, not the real $node_id column.
	t.Run("delimited_table_bare_column_stays_pseudo", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT [weird].$node_id FROM weird")
		require.NoError(t, err)
		require.Len(t, span.Results, 1)
		require.Equal(t, base.SourceColumnSet{
			{Database: "db", Schema: "dbo", Table: "weird", Column: "$node_id"}:    true,
			{Database: "db", Schema: "dbo", Table: "weird", Column: "IDENTITYCOL"}: true,
		}, span.Results[0].SourceColumns)
	})

	t.Run("rowguid_stays_fail_closed", func(t *testing.T) {
		q := newOmniTestExtractor(t, "db")
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT $ROWGUID FROM t")
		require.Error(t, err)
	})
}
