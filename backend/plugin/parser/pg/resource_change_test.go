package pg

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	statement :=
		`CREATE TABLE t1 (c1 INT);
						DROP TABLE t1;
						ALTER TABLE t1 ADD COLUMN c1 INT;
						ALTER TABLE t1 RENAME TO t2;
						COMMENT ON TABLE t1 IS 'comment';
						INSERT INTO t1 (c1) VALUES (1), (5);
						UPDATE t1 SET c1 = 5;
			`
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t2",
		},
		false,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         1,
		DMLStatements:    []string{"UPDATE t1 SET c1 = 5;"},
		InsertCount:      2,
	}

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesReplaysSearchPath(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: `"$user", public`,
		Schemas:    []*metadatapb.SchemaMetadata{{Name: "public"}, {Name: "app"}},
	}, nil, nil, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `UPDATE t SET c = 1;
SET search_path TO app;
UPDATE t SET c = 2;
SET search_path TO public;
UPDATE t SET c = 3;
SET search_path TO app;
RESET search_path;
UPDATE t SET c = 4;
SET search_path TO app;
SET search_path TO DEFAULT;
UPDATE t SET c = 5;
BEGIN;
SET LOCAL search_path TO app;
UPDATE t SET c = 6;
COMMIT;
UPDATE t SET c = 7;
BEGIN;
SET search_path TO app;
ROLLBACK;
UPDATE t SET c = 8;
BEGIN;
SET search_path TO app;
COMMIT AND CHAIN;
ROLLBACK;
UPDATE t SET c = 9;
DISCARD ALL;
UPDATE t SET c = 10;
SET search_path TO app;
BEGIN;
SET search_path TO other;
BEGIN;
SET search_path TO third;
ROLLBACK;
UPDATE t SET c = 11;
SET search_path TO other;
SET LOCAL search_path TO app;
ROLLBACK;
UPDATE t SET c = 12;
BEGIN;
SET LOCAL search_path TO app;
SET search_path FROM CURRENT;
COMMIT;
UPDATE t SET c = 13;`
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	got, err := extractChangedResources("db", "", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, []string{
		"UPDATE t SET c = 1;",
		"SET LOCAL search_path TO \"app\";\nUPDATE t SET c = 2;",
		"UPDATE t SET c = 3;",
		"UPDATE t SET c = 4;",
		"UPDATE t SET c = 5;",
		"SET LOCAL search_path TO \"app\";\nUPDATE t SET c = 6;",
		"UPDATE t SET c = 7;",
		"UPDATE t SET c = 8;",
		"SET LOCAL search_path TO \"app\";\nUPDATE t SET c = 9;",
		"UPDATE t SET c = 10;",
		"SET LOCAL search_path TO \"app\";\nUPDATE t SET c = 11;",
		"SET LOCAL search_path TO \"other\";\nUPDATE t SET c = 12;",
		"SET LOCAL search_path TO \"app\";\nUPDATE t SET c = 13;",
	}, got.DMLStatements)
}

func TestExtractChangedResourcesSelectedSchemaFallsBackToPublicForExistingTarget(t *testing.T) {
	const statement = `INSERT INTO customer VALUES (1);`

	for _, tc := range []struct {
		name       string
		appTables  []*metadatapb.TableMetadata
		appFuncs   []*metadatapb.FunctionMetadata
		wantSchema string
	}{
		{
			name:       "falls back to public when selected schema lacks the table",
			wantSchema: "public",
		},
		{
			name: "selected schema still takes precedence",
			appTables: []*metadatapb.TableMetadata{{
				Name: "customer",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int"},
				},
			}},
			wantSchema: "app",
		},
		{
			name: "relation lookup skips function in selected schema",
			appFuncs: []*metadatapb.FunctionMetadata{{
				Name: "customer",
			}},
			wantSchema: "public",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				Name: "db",
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name:      "app",
						Tables:    tc.appTables,
						Functions: tc.appFuncs,
					},
					{
						Name: "public",
						Tables: []*metadatapb.TableMetadata{{
							Name: "customer",
							Columns: []*metadatapb.ColumnMetadata{
								{Name: "id", Type: "int"},
							},
						}},
					},
				},
			}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)

			want := model.NewChangedResources(dbMetadata)
			want.AddTable("db", tc.wantSchema, &storepb.ChangedResourceTable{Name: "customer"}, false)

			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
			require.NoError(t, err)
			asts := base.ExtractASTs(stmts)
			got, err := extractChangedResources("db", "app", dbMetadata, asts, statement)
			require.NoError(t, err)
			require.Equal(t, want, got.ChangedResources)
		})
	}
}

func TestExtractChangedResourcesTruncate(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `TRUNCATE TABLE public.t1, myschema.t2;`

	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t1"}, true)
	want.AddTable("db", "myschema", &storepb.ChangedResourceTable{Name: "t2"}, true)

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesMerge(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `MERGE INTO myschema.tgt t USING src s ON t.id = s.id WHEN MATCHED THEN UPDATE SET a = s.a;`

	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db", "myschema", &storepb.ChangedResourceTable{Name: "tgt"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesDMLCounts(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	for _, tc := range []struct {
		name            string
		statement       string
		wantDMLCount    int
		wantInsertCount int
		wantTables      []string
	}{
		{
			name:            "insert values",
			statement:       `INSERT INTO t VALUES (1), (2), (3);`,
			wantInsertCount: 3,
			wantTables:      []string{"t"},
		},
		{
			name:            "insert values on conflict",
			statement:       `INSERT INTO t VALUES (1), (2) ON CONFLICT (id) DO UPDATE SET c = 1;`,
			wantInsertCount: 2,
			wantTables:      []string{"t"},
		},
		{
			name:            "insert default values",
			statement:       `INSERT INTO t DEFAULT VALUES;`,
			wantInsertCount: 1,
			wantTables:      []string{"t"},
		},
		{
			name:         "insert select",
			statement:    `INSERT INTO t SELECT * FROM t2 ON CONFLICT DO NOTHING;`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "insert select with read-only CTE",
			statement:    `WITH x AS (SELECT id FROM t2) INSERT INTO t SELECT id FROM x;`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "merge",
			statement:    `MERGE INTO t USING t2 ON t.id = t2.id WHEN MATCHED THEN UPDATE SET c = t2.c;`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "data-modifying CTE feeding insert",
			statement:    `WITH d AS (DELETE FROM t WHERE id > 0 RETURNING *) INSERT INTO t2 SELECT * FROM d;`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "data-modifying CTE feeding insert values",
			statement:    `WITH i AS (INSERT INTO t VALUES (1) RETURNING id) INSERT INTO t2 VALUES (2), (3);`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2"},
		},
		{
			name:         "data-modifying CTE in select",
			statement:    `WITH d AS (DELETE FROM t RETURNING 1) SELECT count(*) FROM d;`,
			wantDMLCount: 1,
			wantTables:   []string{"t"},
		},
		{
			name:         "data-modifying CTEs feeding delete",
			statement:    `WITH u AS (UPDATE t SET c = 1 RETURNING id), m AS (MERGE INTO t3 USING t2 ON t3.id = t2.id WHEN MATCHED THEN DELETE) DELETE FROM t2 WHERE id IN (SELECT id FROM u);`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t2", "t3"},
		},
		{
			name:         "data-modifying CTE in create table as",
			statement:    `CREATE TABLE t_moved AS WITH d AS (DELETE FROM t RETURNING *) SELECT * FROM d;`,
			wantDMLCount: 1,
			wantTables:   []string{"t", "t_moved"},
		},
		{
			name:      "read-only CTE in select",
			statement: `WITH x AS (SELECT id FROM t) SELECT * FROM x;`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := model.NewChangedResources(dbMetadata)
			for _, table := range tc.wantTables {
				want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: table}, false)
			}
			var wantSamples []string
			if tc.wantDMLCount > 0 {
				wantSamples = []string{tc.statement}
			}

			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, tc.statement)
			require.NoError(t, err)
			got, err := extractChangedResources("db", "public", dbMetadata, base.ExtractASTs(stmts), tc.statement)
			require.NoError(t, err)
			require.Equal(t, &base.ChangeSummary{
				ChangedResources: want,
				DMLCount:         tc.wantDMLCount,
				DMLStatements:    wantSamples,
				InsertCount:      tc.wantInsertCount,
			}, got)
		})
	}
}

func TestExtractChangedResourcesExplainAnalyze(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `EXPLAIN (ANALYZE, BUFFERS) WITH d AS (DELETE FROM t RETURNING id) INSERT INTO t2 SELECT id FROM d;
EXPLAIN ANALYZE MERGE INTO t3 USING t ON t3.id = t.id WHEN MATCHED THEN DELETE;
EXPLAIN UPDATE t4 SET c = 1;
EXPLAIN (ANALYZE FALSE) DELETE FROM t5;
EXPLAIN ANALYZE WITH d AS (DELETE FROM t6 RETURNING id) SELECT count(*) FROM d ORDER BY 1;`

	want := model.NewChangedResources(dbMetadata)
	for _, table := range []string{"t", "t2", "t3", "t6"} {
		want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: table}, false)
	}
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	got, err := extractChangedResources("db", "public", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, &base.ChangeSummary{
		ChangedResources: want,
		DMLCount:         3,
		DMLStatements: []string{
			"WITH d AS (DELETE FROM t RETURNING id) INSERT INTO t2 SELECT id FROM d;",
			"MERGE INTO t3 USING t ON t3.id = t.id WHEN MATCHED THEN DELETE;",
			"WITH d AS (DELETE FROM t6 RETURNING id) SELECT count(*) FROM d ORDER BY 1;",
		},
	}, got)
}

func TestExtractChangedResourcesRename(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{Name: "t", RowCount: 10},
				{Name: "t2", RowCount: 1000},
				{Name: "t3", RowCount: 1_000_000},
			},
		}},
	}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	for _, tc := range []struct {
		statement        string
		wantTables       []string
		wantAffectedRows int64
	}{
		{statement: `ALTER TABLE t RENAME TO t_new;`, wantTables: []string{"t", "t_new"}, wantAffectedRows: 10},
		{statement: `ALTER TABLE t2 RENAME COLUMN c TO t3;`, wantTables: []string{"t2"}, wantAffectedRows: 1000},
		{statement: `ALTER TABLE t2 RENAME c TO t3;`, wantTables: []string{"t2"}, wantAffectedRows: 1000},
		{statement: `ALTER TABLE t2 RENAME CONSTRAINT t2_pkey TO t3;`, wantTables: []string{"t2"}, wantAffectedRows: 1000},
		{statement: `ALTER TRIGGER trg ON t2 RENAME TO t3;`, wantTables: []string{"t2"}, wantAffectedRows: 1000},
		{statement: `ALTER INDEX t2_idx RENAME TO t3;`},
		{statement: `ALTER VIEW v RENAME COLUMN c TO t3;`},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, tc.statement)
			require.NoError(t, err)
			got, err := extractChangedResources("db", "public", dbMetadata, base.ExtractASTs(stmts), tc.statement)
			require.NoError(t, err)

			var gotTables []string
			for _, database := range got.ChangedResources.Build().GetDatabases() {
				for _, schema := range database.GetSchemas() {
					for _, table := range schema.GetTables() {
						gotTables = append(gotTables, table.GetName())
					}
				}
			}
			require.Equal(t, tc.wantTables, gotTables)
			require.Equal(t, tc.wantAffectedRows, got.ChangedResources.CountAffectedTableRows())
		})
	}
}

func TestExtractChangedResourcesCTAS(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	for _, tc := range []struct {
		statement, schema, table string
	}{
		{`CREATE TABLE myschema.t1 AS SELECT * FROM src;`, "myschema", "t1"},
		{`SELECT * INTO myschema.t2 FROM src;`, "myschema", "t2"},
		// INTO sits on the first arm of a set operation, not the root.
		{`SELECT a INTO myschema.t3 FROM src UNION SELECT a FROM src2;`, "myschema", "t3"},
	} {
		want := model.NewChangedResources(dbMetadata)
		want.AddTable("db", tc.schema, &storepb.ChangedResourceTable{Name: tc.table}, false)

		stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, tc.statement)
		require.NoError(t, err, tc.statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("db", "public", dbMetadata, asts, tc.statement)
		require.NoError(t, err, tc.statement)
		require.Equal(t, want, got.ChangedResources, tc.statement)
	}
}
