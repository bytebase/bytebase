package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
)

// These pins cover masking-leak shapes that CANNOT live in the legacy-recorded
// differential corpus (test-data/query-span/standard.yaml), because the legacy
// ANTLR resolver either errors on them (fail-closed) or produces lineage that is
// itself a known leak. For each, omni must produce CORRECT lineage — the
// structural rule for shapes legacy cannot resolve: correct lineage or a closed
// failure, never silently empty/misaligned attribution (bytebase's masker is
// positional and fail-open).

func leakPinSpan(t *testing.T, statement string, tables []*storepb.TableMetadata) []base.QuerySpanResult {
	t.Helper()
	span, err := googlesqltest.GetSpan(t, storepb.Engine_BIGQUERY, GetQuerySpan, statement, "ds1",
		googlesqltest.DefaultSchemaTables(tables...))
	require.NoError(t, err, "statement: %s", statement)
	return span.Results
}

// TestLeakPin_LowercaseUsingCoalesce: `SELECT * FROM a JOIN b USING (k)` with a
// lowercase key. Real BigQuery coalesces USING keys case-insensitively (3 output
// columns), but the LEGACY resolver only coalesced upper-case-written keys — its
// lowercase output was 4 uncoalesced columns, shifting every later position
// against the executed result (a masking leak legacy shipped with). omni
// coalesces case-insensitively: one key column whose lineage reads BOTH sides,
// then the non-key columns, positionally aligned with the real output.
func TestLeakPin_LowercaseUsingCoalesce(t *testing.T) {
	results := leakPinSpan(t, "SELECT * FROM a JOIN b USING (k);", []*storepb.TableMetadata{
		{Name: "a", Columns: []*storepb.ColumnMetadata{{Name: "k"}, {Name: "x"}}},
		{Name: "b", Columns: []*storepb.ColumnMetadata{{Name: "k"}, {Name: "y"}}},
	})
	require.Len(t, results, 3, "USING key must be coalesced: real output is [k, x, y]")
	require.Equal(t, []string{"a.k", "b.k"}, googlesqltest.SourcesOf(results[0]), "coalesced key reads both sides")
	require.Equal(t, []string{"a.x"}, googlesqltest.SourcesOf(results[1]))
	require.Equal(t, []string{"b.y"}, googlesqltest.SourcesOf(results[2]))
}

// TestLeakPin_UnnestLineage: `SELECT elem FROM victim, UNNEST(victim.secret_tokens)
// AS elem` — the UNNEST output column's lineage is the unnested array column. The
// legacy resolver ERRORED on UNNEST in FROM (fail-closed); omni resolves it, so it
// must attribute correctly — silently-empty lineage would return the sensitive
// array elements unmasked.
func TestLeakPin_UnnestLineage(t *testing.T) {
	results := leakPinSpan(t, "SELECT elem FROM victim, UNNEST(victim.secret_tokens) AS elem", []*storepb.TableMetadata{
		{Name: "victim", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret_tokens"}}},
	})
	require.Len(t, results, 1)
	require.Equal(t, []string{"victim.secret_tokens"}, googlesqltest.SourcesOf(results[0]),
		"UNNEST element column must carry the array column's lineage")
}

// TestLeakPin_UnnestWithOffsetNoLineage: the WITH OFFSET companion column is
// positional metadata, not data — empty lineage is correct for it, while the
// element column keeps the array lineage.
func TestLeakPin_UnnestWithOffsetNoLineage(t *testing.T) {
	results := leakPinSpan(t, "SELECT elem, pos FROM victim, UNNEST(victim.secret_tokens) AS elem WITH OFFSET AS pos", []*storepb.TableMetadata{
		{Name: "victim", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret_tokens"}}},
	})
	require.Len(t, results, 2)
	require.Equal(t, []string{"victim.secret_tokens"}, googlesqltest.SourcesOf(results[0]))
	require.Empty(t, googlesqltest.SourcesOf(results[1]), "OFFSET is positional, carries no data lineage")
}

// TestLeakPin_ByNameMerge: `UNION ALL BY NAME` aligns set-op arms by column NAME,
// not ordinal. The legacy parser syntax-errors on BY NAME (fail-closed); omni
// parses it, so the analysis must name-merge — an ordinal merge would attribute
// the right arm's sensitive column to the wrong output position (verified leak
// pre-fix: ID <- rt.b_secret).
func TestLeakPin_ByNameMerge(t *testing.T) {
	results := leakPinSpan(t, "SELECT id, label FROM lt UNION ALL BY NAME SELECT b_secret AS label, id FROM rt", []*storepb.TableMetadata{
		{Name: "lt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "label"}}},
		{Name: "rt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "b_secret"}}},
	})
	require.Len(t, results, 2)
	require.Equal(t, []string{"lt.id", "rt.id"}, googlesqltest.SourcesOf(results[0]),
		"id column merges both arms' id lineage by NAME")
	require.Equal(t, []string{"lt.label", "rt.b_secret"}, googlesqltest.SourcesOf(results[1]),
		"label column carries the right arm's b_secret lineage (it is aliased AS label)")
}

// TestLeakPin_ByNameStarArmReversedOrder: a BY NAME set-op whose left arm is a
// STAR takes the deferred SetOpMerge consumer path (metadata expansion first).
// The consumer must name-merge there too — pre-fix it ordinal-merged, which
// mis-attributes whenever the arms' column orders differ (round-3 gate finding).
func TestLeakPin_ByNameStarArmReversedOrder(t *testing.T) {
	results := leakPinSpan(t, "SELECT * FROM pub UNION ALL BY NAME SELECT secret AS label, id FROM priv", []*storepb.TableMetadata{
		{Name: "pub", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "label"}}},
		{Name: "priv", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret"}}},
	})
	require.Len(t, results, 2)
	require.Equal(t, []string{"priv.id", "pub.id"}, googlesqltest.SourcesOf(results[0]),
		"id merges by NAME despite the right arm listing it second")
	require.Equal(t, []string{"priv.secret", "pub.label"}, googlesqltest.SourcesOf(results[1]),
		"label carries priv.secret (aliased AS label) despite it being first in the right arm")
}

// TestLeakPin_ByNameOnMatchColumns: `BY NAME ON (cols)` outputs ONLY the listed
// columns, in list order. Pre-fix the consumer emitted the full ordinal merge
// (wrong arity), shifting the positional masker off every real output column —
// the real single `label` column would have received `id`'s masker.
func TestLeakPin_ByNameOnMatchColumns(t *testing.T) {
	results := leakPinSpan(t, "SELECT * FROM pub UNION ALL BY NAME ON (label) SELECT id, secret AS label FROM priv", []*storepb.TableMetadata{
		{Name: "pub", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "label"}}},
		{Name: "priv", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret"}}},
	})
	require.Len(t, results, 1, "ON (label) restricts the output to exactly the listed column")
	require.Equal(t, "label", results[0].Name)
	require.Equal(t, []string{"priv.secret", "pub.label"}, googlesqltest.SourcesOf(results[0]),
		"the single output column carries both arms' label lineage")
}
