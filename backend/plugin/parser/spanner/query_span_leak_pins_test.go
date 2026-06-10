package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
)

// These pins cover masking-leak shapes that CANNOT live in the legacy-recorded
// differential corpus, because the legacy ANTLR resolver either errors on them
// (fail-closed) or produces lineage that is itself a known leak. The Spanner
// set differs from BigQuery's where the legacy resolvers differed: it adds the
// named-schema and system-schema shapes.

// TestLeakPin_SpannerLowercaseUsingCoalesce: legacy spanner only coalesced a
// fully-upper-case-written USING token (its key map was keyed on the raw
// statement text but probed with the upper-cased field name); the lowercase
// output kept BOTH key columns — a positional masking leak. omni coalesces
// case-insensitively, with the key keeping the field's metadata case.
func TestLeakPin_SpannerLowercaseUsingCoalesce(t *testing.T) {
	span, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT * FROM users JOIN logins USING (id);", "db",
		googlesqltest.DefaultSchemaTables(
			&storepb.TableMetadata{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "email"}}},
			&storepb.TableMetadata{Name: "logins", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret"}}},
		))
	require.NoError(t, err)
	require.Len(t, span.Results, 3, "USING key must be coalesced: real output is [id, email, secret]")
	require.Equal(t, "id", span.Results[0].Name, "the coalesced key keeps the field's metadata case")
	require.Equal(t, []string{"logins.id", "users.id"}, googlesqltest.SourcesOf(span.Results[0]), "coalesced key reads both sides")
	require.Equal(t, []string{"users.email"}, googlesqltest.SourcesOf(span.Results[1]))
	require.Equal(t, []string{"logins.secret"}, googlesqltest.SourcesOf(span.Results[2]))
}

// TestLeakPin_SpannerUnnestLineage: legacy spanner ERRORED on UNNEST in FROM
// ("unsupported table path expression" — fail-closed); omni resolves it and
// must attribute the element to the array column.
func TestLeakPin_SpannerUnnestLineage(t *testing.T) {
	span, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT elem FROM victim, UNNEST(victim.secret_tokens) AS elem", "db",
		googlesqltest.DefaultSchemaTables(
			&storepb.TableMetadata{Name: "victim", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "secret_tokens"}}},
		))
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.Equal(t, []string{"victim.secret_tokens"}, googlesqltest.SourcesOf(span.Results[0]),
		"UNNEST element column must carry the array column's lineage")
}

// TestLeakPin_SpannerByNameMerge: legacy spanner syntax-errors on BY NAME
// (fail-closed); omni parses it and must merge by column NAME, not ordinal.
func TestLeakPin_SpannerByNameMerge(t *testing.T) {
	span, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT id, label FROM lt UNION ALL BY NAME SELECT b_secret AS label, id FROM rt", "db",
		googlesqltest.DefaultSchemaTables(
			&storepb.TableMetadata{Name: "lt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "label"}}},
			&storepb.TableMetadata{Name: "rt", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "b_secret"}}},
		))
	require.NoError(t, err)
	require.Len(t, span.Results, 2)
	require.Equal(t, []string{"lt.id", "rt.id"}, googlesqltest.SourcesOf(span.Results[0]))
	require.Equal(t, []string{"lt.label", "rt.b_secret"}, googlesqltest.SourcesOf(span.Results[1]),
		"label column carries the right arm's b_secret lineage (it is aliased AS label)")
}

// TestLeakPin_SchemaQualifiedStar: `SELECT analytics.events.* FROM
// analytics.events` — the legacy resolver ERRORED on a schema-qualified dot-star
// ("resource not found: column: analytics" — fail-closed); omni resolves it, so
// the star must expand the named schema's table with schema-qualified lineage.
func TestLeakPin_SchemaQualifiedStar(t *testing.T) {
	span, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT analytics.events.* FROM analytics.events", "db",
		[]*storepb.SchemaMetadata{
			{Name: ""},
			{Name: "analytics", Tables: []*storepb.TableMetadata{
				{Name: "events", Columns: []*storepb.ColumnMetadata{{Name: "id"}, {Name: "payload"}}},
			}},
		})
	require.NoError(t, err)
	require.Len(t, span.Results, 2, "the schema-qualified star expands events' two columns")
	require.Equal(t, []string{"analytics.events.id"}, googlesqltest.SourcesOf(span.Results[0]))
	require.Equal(t, []string{"analytics.events.payload"}, googlesqltest.SourcesOf(span.Results[1]))
}

// TestLeakPin_MixedUserSystemRejected: mixing user and system tables must keep
// failing closed (the legacy MixUserSystemTablesError), and a system-only query
// must early-return an EMPTY SelectInfoSchema span (the legacy spanner
// extractor's exact behavior; SPANNER_SYS included).
func TestLeakPin_MixedUserSystemRejected(t *testing.T) {
	userTables := googlesqltest.DefaultSchemaTables(
		&storepb.TableMetadata{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id"}}},
	)
	_, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT * FROM users JOIN INFORMATION_SCHEMA.TABLES ON TRUE", "db", userTables)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	span, err := googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT * FROM INFORMATION_SCHEMA.TABLES", "db", googlesqltest.DefaultSchemaTables())
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type)
	require.Empty(t, span.Results, "system-only query early-returns an empty span (legacy behavior)")
	require.Empty(t, span.SourceColumns)

	span, err = googlesqltest.GetSpan(t, storepb.Engine_SPANNER, GetQuerySpan,
		"SELECT * FROM SPANNER_SYS.QUERY_STATS_TOP_MINUTE", "db", googlesqltest.DefaultSchemaTables())
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type, "SPANNER_SYS is a system schema for the Spanner dialect")
	require.Empty(t, span.Results)
}
