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
		SampleDMLS: []string{
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
