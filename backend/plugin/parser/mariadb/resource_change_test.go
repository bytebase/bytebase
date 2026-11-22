package mariadb

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
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
	INSERT INTO t1 SET c1 = 7;
	UPDATE t1 SET c1 = 5;
	CREATE PROCEDURE getUser(id INT) SELECT * FROM hello WHERE uid = id;
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
			"UPDATE t1 SET c1 = 5;",
		},
		DMLCount:    1,
		InsertCount: 3,
	}

	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
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

	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesLoadData(t *testing.T) {
	const statement = `LOAD DATA INFILE '/tmp/f.csv' INTO TABLE other_db.t (a, b);`

	want := model.NewChangedResources(nil /* dbMetadata */)
	want.AddTable("other_db", "", &storepb.ChangedResourceTable{Name: "t"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesObjectDDLDatabaseOnly(t *testing.T) {
	// Qualified non-table object DDL → a database-only target on the qualifier.
	for _, statement := range []string{
		`CREATE VIEW other_db.v AS SELECT 1;`,
		`DROP VIEW other_db.v;`,
		`ALTER PROCEDURE other_db.p COMMENT 'c';`,
	} {
		stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
		require.NoError(t, err, statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
		require.NoError(t, err, statement)
		require.Equal(t, []string{"other_db"}, got.ChangedResources.GetDatabaseOnlyTargets(), statement)
	}
	// Unqualified → no database-only target (request-database fallback).
	const unqualified = `CREATE VIEW v AS SELECT 1;`
	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, unqualified)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, unqualified)
	require.NoError(t, err)
	require.Empty(t, got.ChangedResources.GetDatabaseOnlyTargets())
}

// TestExtractChangedResourcesMariaDBOnlyStatements is why this package exists.
// While MariaDB ran on omni's MySQL AST, every statement that parser rejected
// was counted as skipped, and a summary with any skipped statement is reported
// as no summary at all — so MariaDB got no changed resources and no
// affected-rows estimate for the whole script.
func TestExtractChangedResourcesMariaDBOnlyStatements(t *testing.T) {
	const statement = `CREATE OR REPLACE TABLE other_db.u (a INT);
	ALTER TABLE t DROP COLUMN IF EXISTS b;
	CREATE SEQUENCE s START WITH 1;
	DELETE FROM d WHERE a = 1 RETURNING a;`

	want := model.NewChangedResources(nil /* dbMetadata */)
	// CREATE OR REPLACE drops the existing table, so u's rows are affected.
	want.AddTable("other_db", "", &storepb.ChangedResourceTable{Name: "u"}, true)
	want.AddTable("db", "", &storepb.ChangedResourceTable{Name: "t"}, true)
	want.AddTable("db", "", &storepb.ChangedResourceTable{Name: "d"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "", nil /* dbMetadata */, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
	require.Zero(t, got.SkippedStatements)
	require.Equal(t, []string{"DELETE FROM d WHERE a = 1 RETURNING a;"}, got.DMLStatements)
	require.Equal(t, 1, got.DMLCount)
}

// TestExtractChangedResourcesRejectsForeignAST guards the mistake this split
// can make: a type assertion on the wrong engine's node makes every statement
// fail rather than skip.
func TestExtractChangedResourcesRejectsForeignAST(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_MYSQL, "CREATE TABLE t (a INT);")
	require.NoError(t, err)
	_, err = extractChangedResources("db", "", nil /* dbMetadata */, base.ExtractASTs(stmts), "CREATE TABLE t (a INT);")
	require.ErrorContains(t, err, "expected AST for MariaDB")
}

// TestExtractChangedResourcesCreateOrReplaceCountsExistingRows covers a form
// MySQL has no grammar for, so an extractor written for MySQL has no flag to
// read. CREATE OR REPLACE TABLE drops the existing table first, so its rows are
// affected and the summary must count them; approval rules read that count as
// statement.affected_rows.
func TestExtractChangedResourcesCreateOrReplaceCountsExistingRows(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "", Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 1000}}},
		},
	}, nil, nil, storepb.Engine_MARIADB, false)

	for _, tc := range []struct {
		statement string
		wantRows  int64
	}{
		{"CREATE TABLE t (a INT);", 0},
		{"CREATE OR REPLACE TABLE t (a INT);", 1000},
		{"CREATE OR REPLACE TABLE t LIKE s;", 1000},
		{"CREATE OR REPLACE TABLE t AS SELECT * FROM s;", 1000},
	} {
		stmts, err := base.ParseStatements(storepb.Engine_MARIADB, tc.statement)
		require.NoError(t, err, tc.statement)
		got, err := extractChangedResources("db", "", dbMetadata, base.ExtractASTs(stmts), tc.statement)
		require.NoError(t, err, tc.statement)
		require.Equal(t, tc.wantRows, got.ChangedResources.CountAffectedTableRows(), tc.statement)
	}
}
