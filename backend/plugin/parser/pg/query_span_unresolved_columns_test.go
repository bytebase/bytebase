package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func column(name, typ string) *storepb.ColumnMetadata {
	return &storepb.ColumnMetadata{Name: name, Type: typ}
}

// healthySchema mirrors a fully synced database: two tables with columns, a
// view, and a materialized view with the dependency columns real sync records.
func healthySchema() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{
				{Name: "t", Columns: []*storepb.ColumnMetadata{column("id", "int4"), column("email", "text"), column("ssn", "text")}},
				{Name: "o", Columns: []*storepb.ColumnMetadata{column("id", "int4"), column("amt", "numeric")}},
			},
			Views: []*storepb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT id, email FROM public.t",
				Columns:    []*storepb.ColumnMetadata{column("id", "int4"), column("email", "text")},
			}},
			MaterializedViews: []*storepb.MaterializedViewMetadata{{
				Name:       "mv",
				Definition: "SELECT id, ssn FROM public.t",
				DependencyColumns: []*storepb.DependencyColumn{
					{Schema: "public", Table: "t", Column: "id"},
					{Schema: "public", Table: "t", Column: "ssn"},
				},
			}},
		}},
	}
}

// degradedSchema is what a sync leaves behind when the connecting role lost its
// privileges: the table is still listed, with no columns under it.
func degradedSchema() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name:   "public",
			Tables: []*storepb.TableMetadata{{Name: "t"}},
		}},
	}
}

// partialSchema keeps some of the table's columns. Same-arity drift like this is
// out of scope for the unresolved-columns signal; see the masking follow-up.
func partialSchema() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name:   "public",
			Tables: []*storepb.TableMetadata{{Name: "t", Columns: []*storepb.ColumnMetadata{column("email", "text"), column("ssn", "text")}}},
		}},
	}
}

func spanFor(t *testing.T, statement string, metadata *storepb.DatabaseSchemaMetadata) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{
		InstanceID:              "inst",
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: statement}, metadata.Name, "", false)
	require.NoError(t, err)
	return span
}

// TestUnresolvedColumnsSignalFiresOnDegradedSnapshot covers the reported bug: a
// table synced with no columns yields a span whose lineage cannot support
// masking. Every one of these shapes returned unmasked rows before the signal
// existed, and none of them sets NotFoundError, so nothing else marks them.
func TestUnresolvedColumnsSignalFiresOnDegradedSnapshot(t *testing.T) {
	statements := []string{
		"SELECT * FROM public.t",
		"SELECT email FROM public.t",
		"SELECT t.email FROM public.t",
		"SELECT * FROM public.t WHERE email = 'x'",
		"SELECT id, email FROM public.t ORDER BY ssn",
		"WITH c AS (SELECT * FROM public.t) SELECT * FROM c",
	}
	for _, statement := range statements {
		t.Run(statement, func(t *testing.T) {
			span := spanFor(t, statement, degradedSchema())
			require.NotNil(t, span.UnresolvedColumnsError,
				"a table with no synced columns must mark the span as unresolvable")
			require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
			require.Equal(t, []string{"db"}, span.UnresolvedColumnsError.Databases(),
				"the re-sync target must survive even when column resolution produced nothing")
		})
	}
}

// TestUnresolvedColumnsSignalQuietOnHealthySnapshot is the false-positive guard.
// Enforcement refuses the query, so every shape here would become a broken
// query. The join, expression-fallback, and EXPLAIN cases are the ones that
// defeated an earlier arity-based version of this check.
func TestUnresolvedColumnsSignalQuietOnHealthySnapshot(t *testing.T) {
	statements := []string{
		"SELECT * FROM public.t",
		"SELECT email FROM public.t",
		"SELECT * FROM public.t NATURAL JOIN public.o",
		"SELECT * FROM public.t JOIN public.o USING (id)",
		"SELECT * FROM public.t LEFT JOIN public.o ON t.id = o.id",
		"SELECT *, md5(email) FROM public.t",
		"SELECT * FROM public.v",
		"SELECT * FROM public.mv",
		"WITH c AS (SELECT * FROM public.t) SELECT * FROM c",
		"SELECT * FROM public.t UNION ALL SELECT * FROM public.t",
		"SELECT DISTINCT ON (id) * FROM public.t",
		"SELECT count(*) FROM public.t",
		"SELECT 1",
		"SELECT * FROM generate_series(1, 3)",
		"EXPLAIN SELECT * FROM public.t",
		"EXPLAIN ANALYZE SELECT * FROM public.t",
		"-- plan\nEXPLAIN ANALYZE SELECT * FROM public.t",
		"SHOW search_path",
		"SET search_path TO public",
	}
	for _, statement := range statements {
		t.Run(statement, func(t *testing.T) {
			span := spanFor(t, statement, healthySchema())
			require.Nil(t, span.UnresolvedColumnsError,
				"a fully synced snapshot must never mark a span unresolvable")
		})
	}
}

// TestUnresolvedColumnsSignalScope pins what this signal deliberately does not
// cover, so a later change does not silently widen or narrow it.
func TestUnresolvedColumnsSignalScope(t *testing.T) {
	t.Run("partial column loss is out of scope", func(t *testing.T) {
		span := spanFor(t, "SELECT * FROM public.t", partialSchema())
		require.Nil(t, span.UnresolvedColumnsError,
			"a table with some columns resolves; same-arity drift needs the catalog-comparison follow-up")
	})

	t.Run("materialized view without columns is not a degraded table", func(t *testing.T) {
		// MaterializedViewMetadata has no column list by design, so an empty one
		// says nothing about snapshot health.
		span := spanFor(t, "SELECT * FROM public.mv", healthySchema())
		require.Nil(t, span.UnresolvedColumnsError)
	})

	t.Run("error names every unresolved relation", func(t *testing.T) {
		metadata := degradedSchema()
		metadata.Schemas[0].Tables = append(metadata.Schemas[0].Tables, &storepb.TableMetadata{Name: "o"})
		span := spanFor(t, "SELECT * FROM public.t, public.o", metadata)
		require.NotNil(t, span.UnresolvedColumnsError)
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.o")
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
	})
}
