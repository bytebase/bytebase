package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/ast"

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

// degradedViewSchema is the same degradation reaching a view. The syncer fills
// a view's column list from the same map as a table's, so a snapshot written
// before #20581 — when the column query still ran through information_schema
// and privileges could hide rows — empties both.
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

// analyzerResolves reports whether omni's analyzer resolved the statement.
//
// It is what puts a span on the path that consults the analyzer's scope. A
// statement the analyzer rejects takes the fallback path, which checks every
// access and would make a coverage assertion pass without the walk over the
// analyzed query being involved at all.
func analyzerResolves(t *testing.T, statement string, metadata *storepb.DatabaseSchemaMetadata) bool {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	extractor := newOmniQuerySpanExtractor(metadata.Name, []string{"public"}, base.GetQuerySpanContext{
		InstanceID:              "inst",
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	})
	extractor.ctx = context.Background()
	statements, err := ParsePg(statement)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	selStmt, ok := statements[0].AST.(*ast.SelectStmt)
	require.True(t, ok)
	require.NoError(t, extractor.initCatalog())
	_, err = extractor.cat.AnalyzeSelectStmt(selStmt)
	return err == nil
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

	t.Run("a view stripped of its columns is unresolved", func(t *testing.T) {
		span := spanFor(t, "SELECT * FROM public.v", degradedViewSchema())
		require.NotNil(t, span.UnresolvedColumnsError,
			"a view with no synced columns hides its base table's masking just as a table does")
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.v")
	})

	t.Run("a foreign table stripped of its columns is unresolved", func(t *testing.T) {
		metadata := &storepb.DatabaseSchemaMetadata{
			Name: "db",
			Schemas: []*storepb.SchemaMetadata{{
				Name:           "public",
				ExternalTables: []*storepb.ExternalTableMetadata{{Name: "ft"}},
			}},
		}
		span := spanFor(t, "SELECT * FROM public.ft", metadata)
		require.NotNil(t, span.UnresolvedColumnsError,
			"foreign tables take their columns from the same sync map as tables and views")
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.ft")
	})

	t.Run("a CTE shadowing a degraded table is not a read of it", func(t *testing.T) {
		// ExtractAccessTables resolves the unqualified `t` to public.t without
		// modelling CTE scope, so the access set names a table this query never
		// reads. The query resolves entirely from the CTE, and the analyzer's
		// range table says so: it holds a CTE reference, not a relation.
		span := spanFor(t, "WITH t AS (SELECT 1 AS n) SELECT * FROM t", degradedSchema())
		require.Nil(t, span.UnresolvedColumnsError,
			"a query that reads only a CTE must not be refused for a same-named table")
		require.Len(t, span.Results, 1, "the CTE resolved, so the span has real lineage")
	})

	t.Run("a CTE nested inside another CTE also shadows", func(t *testing.T) {
		span := spanFor(t, "WITH outer_q AS (WITH t AS (SELECT 1 AS n) SELECT * FROM t) SELECT * FROM outer_q", degradedSchema())
		require.Nil(t, span.UnresolvedColumnsError)
	})

	t.Run("a CTE inside a derived table also shadows", func(t *testing.T) {
		// A WITH clause can sit anywhere a SELECT can, so the walk covers every
		// nested query rather than the top-level one alone.
		span := spanFor(t, "SELECT * FROM (WITH t AS (SELECT 1 AS n) SELECT * FROM t) q", degradedSchema())
		require.Nil(t, span.UnresolvedColumnsError,
			"a CTE bound inside a subquery shadows the physical table just as a top-level one does")
	})

	t.Run("a non-recursive CTE reading its own name reads the table", func(t *testing.T) {
		// This is the case name-based shadowing cannot express, and the reason
		// the check reads the analyzer's range table instead. A non-recursive
		// CTE's own name is not visible inside its own body, so the inner `t` is
		// the physical public.t — verified on PostgreSQL 17, where this statement
		// returns the table's row. The outer `t` is the CTE.
		require.True(t, analyzerResolves(t, "WITH t AS (SELECT * FROM t) SELECT * FROM t", degradedSchema()),
			"the analyzer resolves this statement, so the scope walk decides it")
		span := spanFor(t, "WITH t AS (SELECT * FROM t) SELECT * FROM t", degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError,
			"the CTE body reads the degraded table, so masking cannot be evaluated")
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
	})

	t.Run("a recursive CTE reading its own name reads the CTE", func(t *testing.T) {
		// WITH RECURSIVE is the opposite binding: the name is visible inside the
		// body, so the self-reference is the CTE and no table is read.
		span := spanFor(t,
			"WITH RECURSIVE t AS (SELECT 1 AS n UNION ALL SELECT n + 1 FROM t WHERE n < 3) SELECT * FROM t",
			degradedSchema())
		require.Nil(t, span.UnresolvedColumnsError,
			"a recursive self-reference is the CTE, not the same-named table")
	})

	t.Run("a qualified read is still checked when a CTE shares the name", func(t *testing.T) {
		// PostgreSQL does not let a CTE shadow a schema-qualified name, so this
		// statement really does read the degraded public.t. Excluding by name
		// alone would have dropped it; the analyzer binds it as a relation.
		require.True(t, analyzerResolves(t, "WITH t AS (SELECT 1 AS n) SELECT * FROM public.t", degradedSchema()),
			"the analyzer resolves this statement, so the scope walk decides it")
		span := spanFor(t, "WITH t AS (SELECT 1 AS n) SELECT * FROM public.t", degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError,
			"a schema-qualified read is never shadowed by a CTE of the same name")
		require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
	})

	t.Run("a statement reading both the CTE and the qualified table is checked", func(t *testing.T) {
		// The arms project different column counts, so the analyzer rejects the
		// statement. This is the fallback path, which has no scope to consult and
		// checks every access — including the ones a CTE shares a name with.
		const statement = "WITH t AS (SELECT 1 AS n) SELECT * FROM t UNION ALL SELECT * FROM public.t"
		require.False(t, analyzerResolves(t, statement, degradedSchema()),
			"the analyzer rejects the mismatched set operation, which is what puts this on the fallback path")
		span := spanFor(t, statement, degradedSchema())
		require.NotNil(t, span.UnresolvedColumnsError,
			"the qualified arm is a genuine read regardless of the CTE arm")
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

// TestUnresolvedColumnsSignalReachesEveryReadPosition is the completeness guard
// for the walk over the analyzed query.
//
// Every statement here binds a CTE named t and reads the degraded public.t in
// one position. The CTE name alone would drop the access, so the signal fires
// only if the walk reaches that position. A position the walk misses stops
// checking the reads inside it, which lets a query the snapshot cannot mask run
// unmasked — so this list is the one that must grow when omni grows a new node.
func TestUnresolvedColumnsSignalReachesEveryReadPosition(t *testing.T) {
	const cte = "WITH t AS (SELECT 1 AS n) "
	statements := []string{
		// Range table.
		"SELECT * FROM public.t",
		"SELECT * FROM (SELECT count(*) FROM public.t) s",
		"SELECT * FROM t, LATERAL (SELECT count(*) FROM public.t) s",
		"SELECT * FROM (WITH u AS (SELECT count(*) AS n FROM public.t) SELECT * FROM u) s",
		// Target list, through each expression node that can carry a subquery.
		"SELECT (SELECT count(*) FROM public.t) FROM t",
		"SELECT COALESCE((SELECT count(*) FROM public.t), 0) FROM t",
		"SELECT CASE WHEN true THEN (SELECT count(*) FROM public.t) ELSE 0 END FROM t",
		"SELECT CASE (SELECT count(*) FROM public.t) WHEN 1 THEN 1 ELSE 0 END FROM t",
		"SELECT ARRAY[(SELECT count(*) FROM public.t)] FROM t",
		"SELECT ROW((SELECT count(*) FROM public.t)) FROM t",
		"SELECT NULLIF((SELECT count(*) FROM public.t), 0) FROM t",
		"SELECT GREATEST((SELECT count(*) FROM public.t), 0) FROM t",
		"SELECT ((SELECT count(*) FROM public.t))::text FROM t",
		"SELECT (SELECT count(*) FROM public.t) IS NULL FROM t",
		"SELECT (SELECT count(*) FROM public.t) IS NOT TRUE FROM t",
		"SELECT (SELECT count(*) FROM public.t) IS DISTINCT FROM 1 FROM t",
		"SELECT n = ANY (SELECT count(*) FROM public.t) FROM t",
		"SELECT n IN (SELECT count(*) FROM public.t) FROM t",
		"SELECT abs((SELECT count(*) FROM public.t)) FROM t",
		"SELECT sum((SELECT count(*) FROM public.t)) FROM t",
		"SELECT n + (SELECT count(*) FROM public.t) FROM t",
		"SELECT EXISTS (SELECT 1 FROM public.t) FROM t",
		// Clauses that hold expressions of their own.
		"SELECT n FROM t WHERE n > (SELECT count(*) FROM public.t)",
		"SELECT t.n FROM t JOIN t t2 ON t.n > (SELECT count(*) FROM public.t)",
		"SELECT n FROM t GROUP BY n HAVING count(*) > (SELECT count(*) FROM public.t)",
		"SELECT n FROM t LIMIT (SELECT count(*) FROM public.t)",
		"SELECT n FROM t OFFSET (SELECT count(*) FROM public.t)",
		"SELECT sum(n) OVER (ORDER BY n ROWS BETWEEN (SELECT count(*) FROM public.t) PRECEDING AND CURRENT ROW) FROM t",
		"SELECT count(*) FILTER (WHERE n > (SELECT count(*) FROM public.t)) OVER () FROM t",
		// GROUP BY, ORDER BY and DISTINCT ON carry no expression of their own:
		// omni moves them into the target list as hidden entries.
		"SELECT n FROM t GROUP BY (SELECT count(*) FROM public.t), n",
		"SELECT n FROM t ORDER BY (SELECT count(*) FROM public.t)",
		"SELECT DISTINCT ON ((SELECT count(*) FROM public.t)) n FROM t",
		// Set operation branches.
		"SELECT n FROM t UNION ALL SELECT count(*) FROM public.t",
		"SELECT n FROM t EXCEPT SELECT count(*) FROM public.t",
	}
	for _, statement := range statements {
		t.Run(statement, func(t *testing.T) {
			require.True(t, analyzerResolves(t, cte+statement, degradedSchema()),
				"the analyzer must resolve this statement, or the fallback path would carry the test")
			span := spanFor(t, cte+statement, degradedSchema())
			require.NotNil(t, span.UnresolvedColumnsError,
				"a read of the degraded table in this position must still be checked")
			require.Contains(t, span.UnresolvedColumnsError.Error(), "public.t")
		})
	}
}

// TestUnresolvedColumnsSignalSurvivesUnwalkedExpressions covers the shapes that
// hide a read inside an expression the analyzed-query walk does not reach.
//
// Each statement wraps a subquery over the degraded relation in an expression
// node, and names a CTE after that relation so the walk's CTE evidence would
// otherwise drop the access. An earlier revision returned unmasked rows for
// every one of these: it enumerated catalog.AnalyzedExpr against a stale module
// directory, so twenty-two live types fell through to a default arm that only
// logged. Two defenses close them — the schema-qualified names the parse tree
// carries, which no CTE can shadow, and a default arm that marks the walk's
// evidence incomplete instead of trusting it.
//
// Every case asserts the analyzer resolved the statement, so none of them can
// pass by falling back to the unfiltered path.
func TestUnresolvedColumnsSignalSurvivesUnwalkedExpressions(t *testing.T) {
	const cte = "WITH d AS (SELECT 1 AS id) "
	statements := map[string]string{
		"array subscript":    cte + "SELECT (ARRAY[1,2,3])[(SELECT count(*)::int FROM public.d)]",
		"row comparison":     cte + "SELECT 1 WHERE (1, 1) < ((SELECT count(*)::int FROM public.d), 5)",
		"xml constructor":    cte + "SELECT xmlelement(name e, (SELECT count(*) FROM public.d))",
		"json constructor":   cte + "SELECT json_array((SELECT count(*) FROM public.d))",
		"json is predicate":  cte + "SELECT 1 WHERE (SELECT count(*)::text FROM public.d) IS JSON",
		"aggregate filter":   cte + "SELECT count(*) FILTER (WHERE EXISTS (SELECT 1 FROM public.d))",
		"aggregate order by": cte + "SELECT string_agg('x', ',' ORDER BY (SELECT count(*) FROM public.d))",
		"tablesample arg":    cte + "SELECT 1 FROM public.d TABLESAMPLE BERNOULLI ((SELECT 1.0::float8))",
	}
	for name, statement := range statements {
		t.Run(name, func(t *testing.T) {
			span := spanFor(t, statement, degradedSchemaNamed("d"))
			require.NotNil(t, span.UnresolvedColumnsError,
				"a read of the degraded relation inside this expression must still be seen")
		})
	}
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
