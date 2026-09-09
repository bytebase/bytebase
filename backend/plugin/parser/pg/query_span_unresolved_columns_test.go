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

// degradedSchema is what a pre-#20581 sync left behind when the connecting role
// lost its privileges: the table is still listed, with no columns under it. A
// current PostgreSQL sync reads pg_catalog and cannot produce this, but a
// snapshot written by an older version and never re-synced still carries it.
func degradedSchema() *storepb.DatabaseSchemaMetadata {
	return degradedSchemaNamed("t")
}

// degradedSchemaNamed is degradedSchema with the column-less table under a
// chosen name, for statements that also bind a CTE of that name.
func degradedSchemaNamed(table string) *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name:   "public",
			Tables: []*storepb.TableMetadata{{Name: table}},
		}},
	}
}

// degradedViewSchema is the same degradation reaching a view. The syncer fills a
// view's column list from the same map as a table's, so a pre-#20581 snapshot
// empties both. Masking cannot read a view's columns either way, so this fixture
// pins that a view is not refused.
func degradedViewSchema() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{{
			Name:   "public",
			Tables: []*storepb.TableMetadata{{Name: "t"}},
			Views:  []*storepb.ViewMetadata{{Name: "v", Definition: "SELECT id, email FROM public.t"}},
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
		// The analyzer rejects the four shapes below — the degraded table has no
		// column to bind email, id or ssn to — so they exercise the fallback path,
		// which has no scope resolution to consult and checks every access.
		"SELECT email FROM public.t",
		"SELECT t.email FROM public.t",
		"SELECT * FROM public.t WHERE email = 'x'",
		"SELECT id, email FROM public.t ORDER BY ssn",
		"WITH c AS (SELECT * FROM public.t) SELECT * FROM c",
		// The unqualified t inside a non-recursive CTE body is the physical
		// table: a CTE's own name is not visible in its body. These three sat
		// in positions an analyzed-query walk did not reach (an aggregate's
		// FILTER and ORDER BY subqueries) or folded case on a quoted CTE name,
		// and an earlier revision let them run unmasked.
		"WITH t AS (SELECT count(*) FILTER (WHERE EXISTS (SELECT 1 FROM t)) AS c FROM (SELECT 1) x) SELECT * FROM t",
		"WITH t AS (SELECT string_agg('x', ',' ORDER BY (SELECT count(*) FROM t)) AS c FROM (SELECT 1) x) SELECT * FROM t",
		`WITH "T" AS (SELECT 1 AS n) SELECT count(*) FILTER (WHERE EXISTS (SELECT 1 FROM t)) FROM "T"`,
	}
	for _, statement := range statements {
		t.Run(statement, func(t *testing.T) {
			span := spanFor(t, statement, degradedSchema())
			require.NotNil(t, span.UnresolvedColumnsError,
				"a table with no synced columns must mark the span as unresolvable")
			require.Nil(t, span.NotFoundError,
				"this signal exists because NotFoundError does not fire here; if it starts to, one rule is enforced at two points")
			require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
			require.Equal(t, []string{"db"}, span.UnresolvedColumnsError.Databases(),
				"the re-sync must target the database holding the unresolved relation")
		})
	}
}

// TestUnresolvedColumnsSignalQuietOnHealthySnapshot is the false-positive guard.
// Enforcement refuses the query, so every shape here would become a broken query.
//
// The joins and the expression-fallback case are the load-bearing ones: they
// report relations, so a wrong predicate would refuse them. The EXPLAIN, SHOW,
// SET and constant-select cases report no relation at all and cannot fire
// whatever the predicate does; they are here as regression guards on that, not
// as evidence the predicate is right. They defeated an earlier arity-based
// version of this check, which is a different mechanism.
//
// Quiet here means "the snapshot describes every relation", not "masking is
// correct". NATURAL JOIN in particular produces one span result per input
// column (5 for t(id,email,ssn) NATURAL JOIN o(id,amt)) while the driver
// returns the 4 merged ones, and doMaskResult is positional, so its maskers
// land one column off. That is a separate defect on the snapshot's healthy path.
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

	t.Run("a view with no synced columns is not refused", func(t *testing.T) {
		// Why only tables are judged is at relationHasNoSyncedColumns.
		span := spanFor(t, "SELECT * FROM public.v", degradedViewSchema())
		require.Nil(t, span.UnresolvedColumnsError)
	})

	t.Run("a foreign table with no synced columns is not refused", func(t *testing.T) {
		metadata := &storepb.DatabaseSchemaMetadata{
			Name: "db",
			Schemas: []*storepb.SchemaMetadata{{
				Name:           "public",
				ExternalTables: []*storepb.ExternalTableMetadata{{Name: "ft"}},
			}},
		}
		span := spanFor(t, "SELECT * FROM public.ft", metadata)
		require.Nil(t, span.UnresolvedColumnsError)
	})

	t.Run("a CTE named like a degraded relation is refused too", func(t *testing.T) {
		// ExtractAccessTables resolves the unqualified t to public.t without
		// modeling CTE scope, so the access set names a table this query never
		// reads. That is accepted: resolving CTE scope needs a second walk over
		// the analyzed query, and an earlier revision's walk let real reads
		// through (see the FILTER and ORDER BY cases in the fires test). On a
		// snapshot a re-sync repairs, the refusal lasts one query.
		for _, statement := range []string{
			"WITH t AS (SELECT 1 AS n) SELECT * FROM t",
			"WITH outer_q AS (WITH t AS (SELECT 1 AS n) SELECT * FROM t) SELECT * FROM outer_q",
			"SELECT * FROM (WITH t AS (SELECT 1 AS n) SELECT * FROM t) q",
			"WITH RECURSIVE t AS (SELECT 1 AS n UNION ALL SELECT n + 1 FROM t WHERE n < 3) SELECT * FROM t",
		} {
			span := spanFor(t, statement, degradedSchema())
			require.NotNil(t, span.UnresolvedColumnsError, statement)
		}
	})

	t.Run("a non-recursive CTE reading its own name reads the table", func(t *testing.T) {
		// A non-recursive CTE's own name is not visible inside its own body, so
		// the inner t is the physical public.t (verified on PostgreSQL 17).
		span := spanFor(t, "WITH t AS (SELECT * FROM t) SELECT * FROM t", degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError)
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
	})

	t.Run("a qualified read is still checked when a CTE shares the name", func(t *testing.T) {
		// PostgreSQL does not let a CTE shadow a schema-qualified name.
		span := spanFor(t, "WITH t AS (SELECT 1 AS n) SELECT * FROM public.t", degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError)
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
	})

	t.Run("a statement reading both the CTE and the qualified table is checked", func(t *testing.T) {
		// The arms project different column counts, so the analyzer rejects the
		// statement and the fallback path carries it.
		span := spanFor(t, "WITH t AS (SELECT 1 AS n) SELECT * FROM t UNION ALL SELECT * FROM public.t", degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError)
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

// TestUnresolvedColumnsSignalNotCoveredShapes pins the reads this signal cannot
// see, so the boundary is a recorded decision rather than something a reviewer
// rediscovers.
//
// These are not walk gaps. ExtractAccessTables never reports the relation at
// all, so it is absent from span.SourceColumns too — the access-level fix
// belongs upstream and is tracked in BYT-10076. Until then a query shaped like
// this returns unmasked rows against a degraded snapshot.
func TestUnresolvedColumnsSignalNotCoveredShapes(t *testing.T) {
	notCovered := map[string]string{
		"subquery in a FROM-clause function argument": "SELECT * FROM generate_series(1, (SELECT count(*)::int FROM public.d))",
		"subquery inside a VALUES list":               "SELECT * FROM (VALUES ((SELECT count(*) FROM public.d))) v(x)",
	}
	for name, statement := range notCovered {
		t.Run(name, func(t *testing.T) {
			span := spanFor(t, statement, degradedSchemaNamed("d"))
			require.Empty(t, span.SourceColumns,
				"the premise of this gap is that the access set is empty; if this fails the gap may have been closed upstream")
			require.Nil(t, span.UnresolvedColumnsError,
				"documented gap: no access reported, so nothing to check (BYT-10076)")
		})
	}
}
