package cockroachdb

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"

	// Registers the statement splitter that CockroachDB parsing uses.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

func TestExtractChangedResources(t *testing.T) {
	const statement = `CREATE TABLE t1 (c1 INT);
DROP TABLE t1;
ALTER TABLE t1 ADD COLUMN c2 INT;
ALTER TABLE t1 RENAME TO t2;
INSERT INTO t1 (c1) VALUES (1), (5);
UPDATE t1
SET c1 = 5
WHERE c1 = 1;
DELETE FROM t1; -- trailing comment
`
	want := model.NewChangedResources(nil /* dbMetadata */)
	want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t1"}, true)
	want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t2"}, false)

	got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, statement), statement)
	require.NoError(t, err)
	require.Equal(t, &base.ChangeSummary{
		ChangedResources: want,
		DMLCount:         2,
		DMLStatements:    []string{"UPDATE t1\nSET c1 = 5\nWHERE c1 = 1", "DELETE FROM t1"},
		InsertCount:      2,
	}, got)
}

func TestExtractChangedResourcesDMLCounts(t *testing.T) {
	for _, tc := range []struct {
		name            string
		statement       string
		wantDMLCount    int
		wantInsertCount int
		wantTables      []string
	}{
		{
			name:            "insert values",
			statement:       `INSERT INTO t VALUES (1), (2), (3)`,
			wantInsertCount: 3,
			wantTables:      []string{"t"},
		},
		{
			name:            "parenthesized insert values",
			statement:       `INSERT INTO t (VALUES (1), (2))`,
			wantInsertCount: 2,
			wantTables:      []string{"t"},
		},
		{
			name:            "upsert values",
			statement:       `UPSERT INTO t VALUES (1), (2)`,
			wantInsertCount: 2,
			wantTables:      []string{"t"},
		},
		{
			name:            "insert values on conflict",
			statement:       `INSERT INTO t VALUES (1), (2) ON CONFLICT (id) DO UPDATE SET c = excluded.c`,
			wantInsertCount: 2,
			wantTables:      []string{"t"},
		},
		{
			name:            "insert default values",
			statement:       `INSERT INTO t DEFAULT VALUES`,
			wantInsertCount: 1,
			wantTables:      []string{"t"},
		},
		{
			name:         "insert select",
			statement:    `INSERT INTO t SELECT * FROM t2 ON CONFLICT DO NOTHING`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "upsert select",
			statement:    `UPSERT INTO t SELECT * FROM t2`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "insert select with read-only CTE",
			statement:    `WITH x AS (SELECT id FROM t2) INSERT INTO t SELECT id FROM x`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "update",
			statement:    `UPDATE t SET c = 1 FROM t2 WHERE t.id = t2.id`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "delete",
			statement:    `DELETE FROM t USING t2 WHERE t.id = t2.id ORDER BY t.id LIMIT 10`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "data-modifying CTE feeding insert",
			statement:    `WITH d AS (DELETE FROM t WHERE id > 0 RETURNING *) INSERT INTO t2 SELECT * FROM d`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "data-modifying CTE feeding insert values",
			statement:    `WITH i AS (INSERT INTO t VALUES (1) RETURNING id) INSERT INTO t2 VALUES (2), (3)`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "data-modifying CTE in select",
			statement:    `WITH d AS (DELETE FROM t RETURNING 1) SELECT count(*) FROM d`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "data-modifying CTEs feeding delete",
			statement:    `WITH u AS (UPDATE t SET c = 1 RETURNING id), i AS (UPSERT INTO t3 SELECT * FROM t2 RETURNING id) DELETE FROM t2 WHERE id IN (SELECT id FROM u)`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2", "t3"},
		},
		{
			name:         "statement source in select",
			statement:    `SELECT * FROM [DELETE FROM t WHERE c < 10 RETURNING id]`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "statement source in scalar subquery of insert values",
			statement:    `INSERT INTO t2 VALUES ((SELECT max(id) FROM [DELETE FROM t RETURNING id]))`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "statement source in update subquery",
			statement:    `UPDATE t SET c = 1 WHERE id IN (SELECT id FROM [DELETE FROM t2 WHERE c < 3 RETURNING id])`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "statement source joined in delete using",
			statement:    `DELETE FROM t USING t3 JOIN [UPDATE t2 SET c = 1 RETURNING id] AS u ON t3.id = u.id WHERE t.id = t3.id`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "statement source in create table as",
			statement:    `CREATE TABLE t_moved AS SELECT * FROM [DELETE FROM t RETURNING *]`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t_moved"},
		},
		{
			name:       "create table as",
			statement:  `CREATE TABLE t_copy AS SELECT * FROM t`,
			wantTables: []string{"t_copy"},
		},
		{
			name:      "read-only CTE in select",
			statement: `WITH x AS (SELECT id FROM t) SELECT * FROM x WHERE id IN (SELECT id FROM [SHOW TABLES])`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := model.NewChangedResources(nil /* dbMetadata */)
			for _, table := range tc.wantTables {
				want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: table}, false)
			}
			var wantDMLStatements []string
			if tc.wantDMLCount > 0 {
				wantDMLStatements = []string{tc.statement}
			}

			got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, tc.statement), tc.statement)
			require.NoError(t, err)
			require.Equal(t, &base.ChangeSummary{
				ChangedResources: want,
				DMLCount:         tc.wantDMLCount,
				DMLStatements:    wantDMLStatements,
				InsertCount:      tc.wantInsertCount,
			}, got)
		})
	}
}

func TestExtractChangedResourcesFollowsSearchPath(t *testing.T) {
	const statement = `DELETE FROM t1;
SET search_path = app, public;
DELETE FROM t2;
SET search_path TO '$user', "Reports";
DELETE FROM t3;
RESET search_path;
DELETE FROM t4;`
	got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, statement), statement)
	require.NoError(t, err)
	require.Equal(t, []string{"db.Reports.t3", "db.app.t2", "db.public.t1", "db.public.t4"}, getTableNames(got.ChangedResources))
	// A statement after the search path changes carries it for its EXPLAIN.
	require.Equal(t, []string{
		"DELETE FROM t1",
		"SET LOCAL search_path TO \"app\", \"public\";\nDELETE FROM t2",
		"SET LOCAL search_path TO \"Reports\";\nDELETE FROM t3",
		"DELETE FROM t4",
	}, got.DMLStatements)

	t.Run("SET LOCAL lasts until the transaction ends, SET outlasts ROLLBACK, and DISCARD ALL resets", func(t *testing.T) {
		const statement = `BEGIN;
SET LOCAL search_path = app;
DELETE FROM t1;
COMMIT;
DELETE FROM t2;
BEGIN;
SET search_path = app;
SET LOCAL search_path = other;
ROLLBACK;
DELETE FROM t3;
DISCARD ALL;
DELETE FROM t4;`
		got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, statement), statement)
		require.NoError(t, err)
		require.Equal(t, []string{"db.app.t1", "db.app.t3", "db.public.t2", "db.public.t4"}, getTableNames(got.ChangedResources))
	})

	t.Run("the synced search path applies without a current schema", func(t *testing.T) {
		dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:       "db",
			SearchPath: `"$user", app, public`,
			Schemas: []*metadatapb.SchemaMetadata{
				{Name: "app", Tables: []*metadatapb.TableMetadata{{Name: "t"}}},
				{Name: "public", Tables: []*metadatapb.TableMetadata{{Name: "t2"}}},
			},
		}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_COCKROACHDB, true /* caseSensitive */)
		const statement = `DELETE FROM t; DELETE FROM t2; DROP MATERIALIZED VIEW mv; DROP VIEW v;`
		got, err := extractChangedResources("db", "", dbMetadata, parseASTs(t, statement), statement)
		require.NoError(t, err)
		require.Equal(t, []string{"db.app.mv", "db.app.t", "db.public.t2"}, getTableNames(got.ChangedResources))
	})

	t.Run("existing tables resolve to the first schema that has them", func(t *testing.T) {
		dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "db",
			Schemas: []*metadatapb.SchemaMetadata{
				{Name: "public", Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 1000}}},
				{Name: "empty"},
			},
		}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_COCKROACHDB, true /* caseSensitive */)
		const statement = `SET search_path = empty, public;
DELETE FROM t;
CREATE TABLE t2 (id INT PRIMARY KEY);`
		got, err := extractChangedResources("db", "public", dbMetadata, parseASTs(t, statement), statement)
		require.NoError(t, err)
		require.Equal(t, []string{"db.empty.t2", "db.public.t"}, getTableNames(got.ChangedResources))
	})
}

func TestExtractChangedResourcesNames(t *testing.T) {
	for _, tc := range []struct {
		name          string
		statement     string
		currentSchema string
		want          string
	}{
		{name: "unqualified defaults to public", statement: `UPDATE t SET c = 1`, want: "db.public.t"},
		{name: "unqualified uses the current schema", statement: `UPDATE t SET c = 1`, currentSchema: "app", want: "db.app.t"},
		{name: "schema qualified with alias", statement: `INSERT INTO myschema.t AS x VALUES (1)`, currentSchema: "app", want: "db.myschema.t"},
		{name: "database qualified", statement: `DELETE FROM otherdb.myschema.t`, want: "otherdb.myschema.t"},
		{name: "quoted identifiers keep their case", statement: `UPDATE "MySchema"."T" SET c = 1`, want: "db.MySchema.T"},
		{name: "unquoted identifiers are lowercased", statement: `TRUNCATE MySchema.T`, want: "db.myschema.t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractChangedResources("db", tc.currentSchema, nil /* dbMetadata */, parseASTs(t, tc.statement), tc.statement)
			require.NoError(t, err)
			require.Equal(t, []string{tc.want}, getTableNames(got.ChangedResources))
		})
	}
}

func TestExtractChangedResourcesDDL(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "t", RowCount: 10},
					{Name: "t2", RowCount: 1000, Indexes: []*metadatapb.IndexMetadata{{Name: "t2_c_idx"}}},
				},
			},
			{
				Name:   "app",
				Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 100_000, Indexes: []*metadatapb.IndexMetadata{{Name: "t_c_idx"}}}},
			},
		},
	}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_COCKROACHDB, true /* caseSensitive */)
	for _, tc := range []struct {
		statement        string
		wantTables       []string
		wantAffectedRows int64
	}{
		{statement: `CREATE TABLE t3 (id INT PRIMARY KEY)`, wantTables: []string{"db.public.t3"}},
		{statement: `CREATE INDEX idx ON t2 (c)`, wantTables: []string{"db.public.t2"}},
		{statement: `CREATE MATERIALIZED VIEW mv AS SELECT * FROM t`, wantTables: []string{"db.public.mv"}},
		{statement: `CREATE VIEW v AS SELECT * FROM t`},
		{statement: `IMPORT INTO t2 (c) CSV DATA ('nodelocal://1/t2.csv')`, wantTables: []string{"db.public.t2"}},
		{statement: `ALTER TABLE t2 CONFIGURE ZONE USING num_replicas = 5`, wantTables: []string{"db.public.t2"}, wantAffectedRows: 1000},
		{statement: `ALTER INDEX t2_c_idx CONFIGURE ZONE USING num_replicas = 5`, wantTables: []string{"db.public.t2"}},
		{statement: `ALTER RANGE default CONFIGURE ZONE USING num_replicas = 5`},
		{statement: `DROP INDEX t2@t2_c_idx, t2_c_idx, app.t_c_idx, missing_idx`, wantTables: []string{"db.app.t", "db.public.t2"}},
		{statement: `DROP INDEX db.public.t2_c_idx`, wantTables: []string{"db.public.t2"}},
		{statement: `ALTER INDEX app.t@t_c_idx PARTITION BY NOTHING`, wantTables: []string{"db.app.t"}},
		{statement: `ALTER INDEX t2_c_idx NOT VISIBLE`, wantTables: []string{"db.public.t2"}},
		{statement: `ALTER INDEX t2_c_idx RENAME TO t2_idx`, wantTables: []string{"db.public.t2"}},
		{statement: `ALTER TABLE t ADD COLUMN c2 INT, DROP COLUMN c`, wantTables: []string{"db.public.t"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE t RENAME COLUMN c TO c2`, wantTables: []string{"db.public.t"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE t SET LOCALITY REGIONAL BY ROW`, wantTables: []string{"db.public.t"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE t SET SCHEMA app`, wantTables: []string{"db.public.t"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE t OWNER TO u`, wantTables: []string{"db.public.t"}, wantAffectedRows: 10},
		{statement: `ALTER VIEW t2 SET SCHEMA app`},
		{statement: `ALTER MATERIALIZED VIEW t2 SET SCHEMA app`, wantTables: []string{"db.public.t2"}, wantAffectedRows: 1000},
		{statement: `ALTER MATERIALIZED VIEW t2 OWNER TO u`, wantTables: []string{"db.public.t2"}, wantAffectedRows: 1000},
		{statement: `ALTER TABLE t RENAME TO t_new`, wantTables: []string{"db.public.t", "db.public.t_new"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE app.t RENAME TO t_new`, wantTables: []string{"db.app.t", "db.app.t_new"}, wantAffectedRows: 100_000},
		{statement: `ALTER MATERIALIZED VIEW t2 RENAME TO mv`, wantTables: []string{"db.public.mv", "db.public.t2"}, wantAffectedRows: 1000},
		{statement: `ALTER VIEW t2 RENAME TO v`},
		{statement: `ALTER SEQUENCE t2 RENAME TO s`},
		{statement: `DROP TABLE t, app.t`, wantTables: []string{"db.app.t", "db.public.t"}, wantAffectedRows: 100_010},
		{statement: `TRUNCATE t2`, wantTables: []string{"db.public.t2"}, wantAffectedRows: 1000},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			got, err := extractChangedResources("db", "public", dbMetadata, parseASTs(t, tc.statement), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantTables, getTableNames(got.ChangedResources))
			require.Equal(t, tc.wantAffectedRows, got.ChangedResources.CountAffectedTableRows())
			require.Zero(t, got.DMLCount)
		})
	}

	t.Run("an index of another database is recorded by its database", func(t *testing.T) {
		const statement = `DROP INDEX otherdb.public.t2_c_idx`
		got, err := extractChangedResources("db", "public", dbMetadata, parseASTs(t, statement), statement)
		require.NoError(t, err)
		require.Empty(t, getTableNames(got.ChangedResources))
		require.Equal(t, []string{"otherdb"}, got.ChangedResources.GetDatabaseOnlyTargets())
	})
}

func TestExtractChangedResourcesExplainAnalyze(t *testing.T) {
	const statement = `EXPLAIN ANALYZE (VERBOSE) DELETE FROM t WHERE id = 1;
EXPLAIN ANALYZE UPSERT INTO t2 SELECT * FROM t;
EXPLAIN UPDATE t3 SET c = 1;`
	got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, statement), statement)
	require.NoError(t, err)
	require.Equal(t, []string{"db.public.t", "db.public.t2"}, getTableNames(got.ChangedResources))
	require.Equal(t, []string{"DELETE FROM t WHERE id = 1", "UPSERT INTO t2 SELECT * FROM t"}, got.DMLStatements)
}

func TestExtractChangedResourcesListsEveryDMLStatement(t *testing.T) {
	var statements []string
	for i := range 12 {
		statements = append(statements, fmt.Sprintf("DELETE FROM t WHERE id = %d", i))
	}
	statement := strings.Join(statements, ";\n")

	got, err := extractChangedResources("db", "public", nil /* dbMetadata */, parseASTs(t, statement), statement)
	require.NoError(t, err)
	require.Equal(t, 12, got.DMLCount)
	require.Equal(t, statements, got.DMLStatements)
}

func parseASTs(t *testing.T, statement string) []base.AST {
	t.Helper()
	stmts, err := base.ParseStatements(storepb.Engine_COCKROACHDB, statement)
	require.NoError(t, err)
	return base.ExtractASTs(stmts)
}

// getTableNames returns the changed tables as sorted database.schema.table names.
func getTableNames(changedResources *model.ChangedResources) []string {
	var names []string
	for _, database := range changedResources.Build().GetDatabases() {
		for _, schema := range database.GetSchemas() {
			for _, table := range schema.GetTables() {
				names = append(names, database.GetName()+"."+schema.GetName()+"."+table.GetName())
			}
		}
	}
	slices.Sort(names)
	return names
}
