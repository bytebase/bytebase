package tsql

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
	ALTER TABLE t1 ADD colb INT;
	INSERT INTO t1 (c1) VALUES (1), (5);
	UPDATE t1 SET c1 = 5;
	`
	changedResources := model.NewChangedResources(nil /* dbMetadata */)
	changedResources.AddTable(
		"DB",
		"dbo",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount:    1,
		InsertCount: 2,
	}

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.Len(t, asts, 5)
	got, err := extractChangedResources("DB", "dbo", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResources_DropIndex(t *testing.T) {
	statement := `DROP INDEX idx1 ON t1;`

	want := model.NewChangedResources(nil)
	want.AddTable("DB", "dbo", &storepb.ChangedResourceTable{Name: "t1"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResources_InsertDefaultValues(t *testing.T) {
	statement := `INSERT INTO t1 DEFAULT VALUES;`

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, 1, got.InsertCount)
	require.Equal(t, 0, got.DMLCount)
	require.Empty(t, got.SampleDMLS)
}

func TestExtractChangedResources_Truncate(t *testing.T) {
	const statement = `TRUNCATE TABLE dbo.t;`

	want := model.NewChangedResources(nil)
	want.AddTable("DB", "dbo", &storepb.ChangedResourceTable{Name: "t"}, true)

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResources_Merge(t *testing.T) {
	const statement = `MERGE INTO other_db.dbo.tgt AS t USING dbo.src AS s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = s.a;`

	want := model.NewChangedResources(nil)
	want.AddTable("other_db", "dbo", &storepb.ChangedResourceTable{Name: "tgt"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResources_ObjectDDLDatabaseOnly(t *testing.T) {
	// Qualified non-table object DDL → a database-only target on the qualifier.
	for _, statement := range []string{
		`CREATE VIEW other_db.dbo.v AS SELECT 1 AS x;`,
		`DROP VIEW other_db.dbo.v;`,
	} {
		stmts, err := base.ParseStatements(storepb.Engine_MSSQL, statement)
		require.NoError(t, err, statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("DB", "dbo", nil, asts, statement)
		require.NoError(t, err, statement)
		require.Equal(t, []string{"other_db"}, got.ChangedResources.GetDatabaseOnlyTargets(), statement)
	}
	// Unqualified → no database-only target (request-database fallback).
	const unqualified = `CREATE VIEW v AS SELECT 1 AS x;`
	stmts, err := base.ParseStatements(storepb.Engine_MSSQL, unqualified)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("DB", "dbo", nil, asts, unqualified)
	require.NoError(t, err)
	require.Empty(t, got.ChangedResources.GetDatabaseOnlyTargets())
}
