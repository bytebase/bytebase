package spanner

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// These pins cover masking-leak shapes that CANNOT live in the legacy-recorded
// differential corpus (test-data/query-span/standard.yaml), because the legacy
// ANTLR resolver either errors on them (fail-closed) or produces lineage that is
// itself a known leak. For each, omni must produce CORRECT lineage — the
// structural rule for shapes legacy cannot resolve: correct lineage or a closed
// failure, never silently empty/misaligned attribution (bytebase's masker is
// positional and fail-open).

func leakPinSpan(t *testing.T, statement string, schemas []*storepb.SchemaMetadata) (*base.QuerySpan, error) {
	t.Helper()
	meta := &storepb.DatabaseSchemaMetadata{Name: "db", Schemas: schemas}
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{meta})
	return GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: statement}, "db", "", false)
}

func defaultSchemaTables(tables ...*storepb.TableMetadata) []*storepb.SchemaMetadata {
	return []*storepb.SchemaMetadata{{Name: "", Tables: tables}}
}

func sourcesOf(r base.QuerySpanResult) []string {
	out := make([]string, 0, len(r.SourceColumns))
	for c := range r.SourceColumns {
		name := c.Table + "." + c.Column
		if c.Schema != "" {
			name = c.Schema + "." + name
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// TestLeakPin_LowercaseUsingCoalesce: `SELECT * FROM a JOIN b USING (id)` with a
// lowercase key. Real Spanner coalesces USING keys case-insensitively (one key
// column), but the LEGACY resolver only coalesced a fully-upper-case-written
// USING token — its lowercase output kept BOTH key columns, shifting every later
// position against the executed result (a masking leak legacy shipped with).
// omni coalesces case-insensitively: one key column reading BOTH sides, then the
// non-key columns, positionally aligned with the real output.
func TestLeakPin_LowercaseUsingCoalesce(t *testing.T) {
	span, err := leakPinSpan(t, "SELECT * FROM users JOIN logins USING (id);", defaultSchemaTables(
		&storepb.TableMetadata{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "email"}}},
		&storepb.TableMetadata{Name: "logins", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret"}}},
	))
	require.NoError(t, err)
	require.Len(t, span.Results, 3, "USING key must be coalesced: real output is [id, email, secret]")
	require.Equal(t, "id", span.Results[0].Name, "the coalesced key keeps the field's metadata case")
	require.Equal(t, []string{"logins.id", "users.id"}, sourcesOf(span.Results[0]), "coalesced key reads both sides")
	require.Equal(t, []string{"users.email"}, sourcesOf(span.Results[1]))
	require.Equal(t, []string{"logins.secret"}, sourcesOf(span.Results[2]))
}

// TestLeakPin_UnnestLineage: the UNNEST output column's lineage is the unnested
// array column. The legacy resolver ERRORED on UNNEST in FROM ("unsupported
// table path expression" — fail-closed); omni resolves it, so it must attribute
// correctly — silently-empty lineage would return the sensitive array elements
// unmasked.
func TestLeakPin_UnnestLineage(t *testing.T) {
	span, err := leakPinSpan(t, "SELECT elem FROM victim, UNNEST(victim.secret_tokens) AS elem", defaultSchemaTables(
		&storepb.TableMetadata{Name: "victim", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret_tokens"}}},
	))
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.Equal(t, []string{"victim.secret_tokens"}, sourcesOf(span.Results[0]),
		"UNNEST element column must carry the array column's lineage")
}

// TestLeakPin_ByNameMerge: `UNION ALL BY NAME` aligns set-op arms by column NAME,
// not ordinal. The legacy parser syntax-errors on BY NAME (fail-closed); omni
// parses it, so the analysis must name-merge — an ordinal merge would attribute
// the right arm's sensitive column to the wrong output position.
func TestLeakPin_ByNameMerge(t *testing.T) {
	span, err := leakPinSpan(t, "SELECT id, label FROM lt UNION ALL BY NAME SELECT b_secret AS label, id FROM rt", defaultSchemaTables(
		&storepb.TableMetadata{Name: "lt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "label"}}},
		&storepb.TableMetadata{Name: "rt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "b_secret"}}},
	))
	require.NoError(t, err)
	require.Len(t, span.Results, 2)
	require.Equal(t, []string{"lt.id", "rt.id"}, sourcesOf(span.Results[0]),
		"id column merges both arms' id lineage by NAME")
	require.Equal(t, []string{"lt.label", "rt.b_secret"}, sourcesOf(span.Results[1]),
		"label column carries the right arm's b_secret lineage (it is aliased AS label)")
}

// TestLeakPin_SchemaQualifiedStar: `SELECT analytics.events.* FROM
// analytics.events` — the legacy resolver ERRORED on a schema-qualified dot-star
// ("resource not found: column: analytics" — fail-closed); omni resolves it, so
// the star must expand the named schema's table with schema-qualified lineage.
func TestLeakPin_SchemaQualifiedStar(t *testing.T) {
	span, err := leakPinSpan(t, "SELECT analytics.events.* FROM analytics.events", []*storepb.SchemaMetadata{
		{Name: ""},
		{Name: "analytics", Tables: []*storepb.TableMetadata{
			{Name: "events", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "payload"}}},
		}},
	})
	require.NoError(t, err)
	require.Len(t, span.Results, 2, "the schema-qualified star expands events' two columns")
	require.Equal(t, []string{"analytics.events.id"}, sourcesOf(span.Results[0]))
	require.Equal(t, []string{"analytics.events.payload"}, sourcesOf(span.Results[1]))
}

// TestLeakPin_MixedUserSystemRejected: mixing user and system tables must keep
// failing closed (the legacy MixUserSystemTablesError), and a system-only query
// must early-return an EMPTY SelectInfoSchema span.
func TestLeakPin_MixedUserSystemRejected(t *testing.T) {
	_, err := leakPinSpan(t, "SELECT * FROM users JOIN INFORMATION_SCHEMA.TABLES ON TRUE", defaultSchemaTables(
		&storepb.TableMetadata{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id"}}},
	))
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	span, err := leakPinSpan(t, "SELECT * FROM INFORMATION_SCHEMA.TABLES", defaultSchemaTables())
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type)
	require.Empty(t, span.Results, "system-only query early-returns an empty span (legacy behavior)")
	require.Empty(t, span.SourceColumns)

	span, err = leakPinSpan(t, "SELECT * FROM SPANNER_SYS.QUERY_STATS_TOP_MINUTE", defaultSchemaTables())
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type, "SPANNER_SYS is a system schema for the Spanner dialect")
	require.Empty(t, span.Results)
}
