package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	DROP TABLE t1;
	ALTER TABLE t1 ADD COLUMN c1 INT;
	RENAME TABLE t1 TO t2;
	INSERT INTO t1 (c1) VALUES (1), (5);
	UPDATE t1 SET c1 = 5;
	`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"db",
		"",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"",
		&storepb.ChangedResourceTable{
			Name: "t2",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements: []string{
			"UPDATE t1 SET c1 = 5",
		},
		DMLCount:    1,
		InsertCount: 2,
	}

	stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesTruncate(t *testing.T) {
	const statement = `TRUNCATE TABLE t;`

	want := model.NewChangedResources(nil /* dbMetadata */)
	want.AddTable("db", "", &storepb.ChangedResourceTable{Name: "t"}, true)

	stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesLoadDataImport(t *testing.T) {
	for _, statement := range []string{
		`LOAD DATA INFILE '/tmp/f.csv' INTO TABLE other_db.t;`,
		`IMPORT INTO other_db.t FROM '/tmp/f.csv';`,
	} {
		want := model.NewChangedResources(nil /* dbMetadata */)
		want.AddTable("other_db", "", &storepb.ChangedResourceTable{Name: "t"}, false)

		stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
		require.NoError(t, err, statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
		require.NoError(t, err, statement)
		require.Equal(t, want, got.ChangedResources, statement)
	}
}

func TestExtractChangedResourcesObjectDDLDatabaseOnly(t *testing.T) {
	// Qualified non-table object DDL (view/sequence) → a database-only target on the qualifier.
	for _, statement := range []string{
		`CREATE VIEW other_db.v AS SELECT 1;`,
		`DROP VIEW other_db.v;`,
		`CREATE SEQUENCE other_db.s;`,
	} {
		stmts, err := base.ParseStatements(storepb.Engine_TIDB, statement)
		require.NoError(t, err, statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
		require.NoError(t, err, statement)
		require.Equal(t, []string{"other_db"}, got.ChangedResources.GetDatabaseOnlyTargets(), statement)
	}
	// Unqualified → no database-only target (request-database fallback).
	const unqualified = `CREATE VIEW v AS SELECT 1;`
	stmts, err := base.ParseStatements(storepb.Engine_TIDB, unqualified)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, unqualified)
	require.NoError(t, err)
	require.Empty(t, got.ChangedResources.GetDatabaseOnlyTargets())
}

func TestExtractChangedResourcesDMLCounts(t *testing.T) {
	for _, tc := range []struct {
		name        string
		statement   string
		dmlCount    int
		insertCount int
		samples     []string
		tables      []string
	}{
		{
			name:        "insert_set_is_one_row",
			statement:   `INSERT INTO t SET c = 1, d = 2;`,
			insertCount: 1,
			tables:      []string{"t"},
		},
		{
			name:      "batch_samples_inner_dml",
			statement: `BATCH ON id LIMIT 1000 DELETE FROM t WHERE c = 1;`,
			dmlCount:  1,
			samples:   []string{"DELETE FROM t WHERE c = 1"},
			tables:    []string{"t"},
		},
		{
			name:      "batch_with_an_executable_comment_is_counted_without_a_sample",
			statement: `BATCH ON id LIMIT 1000 DELETE FROM t WHERE /*T![clustered_index] c = 0 AND */ id < 10000000000;`,
			dmlCount:  1,
			tables:    []string{"t"},
		},
		{
			name:      "batch_dry_run_changes_nothing",
			statement: `BATCH ON id LIMIT 1000 DRY RUN DELETE FROM t WHERE c = 1;`,
		},
		{
			// pingcap cannot parse the row alias, which omni accepts.
			name:        "statement_without_pingcap_ast",
			statement:   "INSERT INTO t VALUES (1, 2) AS new ON DUPLICATE KEY UPDATE c = new.c;\nUPDATE t2 SET c = 1 WHERE id = 1;",
			dmlCount:    1,
			insertCount: 1,
			samples:     []string{"UPDATE t2 SET c = 1 WHERE id = 1"},
			tables:      []string{"t", "t2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_TIDB, tc.statement)
			require.NoError(t, err)
			got, err := extractChangedResources("db", "", nil /* dbMetadata */, base.ExtractASTs(stmts), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.dmlCount, got.DMLCount)
			require.Equal(t, tc.insertCount, got.InsertCount)
			require.Equal(t, tc.samples, got.DMLStatements)
			var tables []string
			for _, database := range got.ChangedResources.Build().GetDatabases() {
				for _, schema := range database.GetSchemas() {
					for _, table := range schema.GetTables() {
						tables = append(tables, table.GetName())
					}
				}
			}
			require.Equal(t, tc.tables, tables)
		})
	}
}
