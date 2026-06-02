package tidb

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type rollbackCase struct {
	Input  string
	Result []base.BackupStatement
}

func TestBackup(t *testing.T) {
	tests := []rollbackCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_backup.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         false,
		}, t.Input, "db", "backupDB", "_rollback")
		a.NoError(err)
		slices.SortFunc(result, func(a, b base.BackupStatement) int {
			if a.TargetTableName == b.TargetTableName {
				if a.Statement < b.Statement {
					return -1
				}
				if a.Statement > b.Statement {
					return 1
				}
				return 0
			}
			if a.TargetTableName < b.TargetTableName {
				return -1
			}
			if a.TargetTableName > b.TargetTableName {
				return 1
			}
			return 0
		})

		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Input)
		}
	}
	if record {
		yamltest.Record(t, filepath, tests)
	}
}

func buildFixedMockDatabaseMetadataGetterAndLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	schemaMetadata := []*store.SchemaMetadata{
		{
			Name: "",
			Tables: []*store.TableMetadata{
				{
					Name: "t_generated",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c_generated",
							Generation: &store.GenerationMetadata{
								Expression: "a + b",
							},
						},
					},
					Indexes: []*store.IndexMetadata{
						{
							Name:    "PRIMARY",
							Primary: true,
							Unique:  true,
							Expressions: []string{
								"b",
							},
						},
						{
							Name:   "uk_a",
							Unique: true,
							Expressions: []string{
								"a",
							},
						},
						// Unique key on a generated column (c_generated = a + b).
						// Used by TestGenerateRestoreSQLGeneratedColumnUKSkipped to
						// pin that hasDisjointUniqueKey skips UKs whose
						// expressions reference generated columns. Pre-fix this
						// UK would false-positive as disjoint via naive string
						// comparison; post-fix it's correctly skipped.
						{
							Name:   "uk_c_generated",
							Unique: true,
							Expressions: []string{
								"c_generated",
							},
						},
						// Unique key with empty Expressions — represents the
						// TiDB-metadata shape for some expression/functional
						// index parts that don't populate key.Column (per
						// backend/plugin/schema/tidb/get_database_metadata.go).
						// Used by TestGenerateRestoreSQLEmptyExpressionsUKSkipped
						// to pin that hasDisjointUniqueKey skips empty-
						// Expressions UKs. Pre-fix: disjoint([]) returns
						// vacuously true, false-positive disjoint. Post-fix:
						// empty-Expressions UKs are skipped explicitly.
						{
							Name:        "uk_empty_expressions",
							Unique:      true,
							Expressions: nil,
						},
					},
				},
				{
					Name: "t1",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "t2",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "test",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
					Indexes: []*store.IndexMetadata{
						{
							Name:    "PRIMARY",
							Primary: true,
							Expressions: []string{
								"c",
							},
						},
						{
							Name:   "PRIMARY",
							Unique: true,
							Expressions: []string{
								"a",
							},
						},
					},
				},
				{
					Name: "test2",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "t3",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "d",
						},
					},
				},
			},
		},
	}

	return func(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
			return database, model.NewDatabaseMetadata(&store.DatabaseSchemaMetadata{
				Name:    database,
				Schemas: schemaMetadata,
			}, nil, nil, store.Engine_TIDB, false /* isObjectCaseSensitive */), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db", "db1", "db2"}, nil
		}
}

// TestBackupRejectsAndPreserves pins behaviors the omni port must keep
// (regressions caught in PR #20480 review):
//   - TiDB BATCH is rejected, not silently skipped (else the executor would run
//     the mutation with no backup).
//   - Cross-database mutations (DELETE and UPDATE) are rejected (a
//     BackupStatement carries no source database, so the executor would restore
//     into the wrong database).
//   - A derived table in the UPDATE FROM list is preserved (not dropped, which
//     would emit a malformed "FROM  WHERE ..." clause).
//   - A WITH (CTE) prefix is carried into the backup SELECT (else a FROM that
//     references the CTE would be invalid SQL).
func TestBackupRejectsAndPreserves(t *testing.T) {
	a := require.New(t)
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	run := func(sql string) ([]base.BackupStatement, error) {
		return TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         false,
		}, sql, "db", "backupDB", "_rollback")
	}

	_, err := run("BATCH LIMIT 2 DELETE FROM test WHERE id > 0")
	a.Error(err, "BATCH (non-transactional DML) must be rejected")

	_, err = run("DELETE FROM db1.t1 WHERE a = 1")
	a.Error(err, "cross-database DELETE must be rejected")

	_, err = run("UPDATE db1.t1 SET c1 = 1 WHERE c1 = 2")
	a.Error(err, "cross-database UPDATE must be rejected")

	result, err := run("UPDATE test, (SELECT id FROM test2) AS x SET test.c1 = 1 WHERE test.id = x.id")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal(
		"CREATE TABLE `backupDB`.`_rollback_0_test` LIKE `db`.`test`;\n"+
			"INSERT INTO `backupDB`.`_rollback_0_test` SELECT `test`.* FROM test, (SELECT id FROM test2) AS x WHERE test.id = x.id;",
		result[0].Statement,
	)

	// A WITH (CTE) prefix must be carried into the backup SELECT, else the FROM
	// (which joins the CTE) references an undefined name.
	result, err = run("WITH x AS (SELECT id FROM test2) UPDATE test JOIN x ON test.id = x.id SET test.c1 = 1")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal(
		"CREATE TABLE `backupDB`.`_rollback_0_test` LIKE `db`.`test`;\n"+
			"INSERT INTO `backupDB`.`_rollback_0_test` WITH x AS (SELECT id FROM test2) SELECT `test`.* FROM test JOIN x ON test.id = x.id;",
		result[0].Statement,
	)
	// The generated backup must be valid SQL (the CTE is defined before use).
	_, perr := ParseTiDBOmni(result[0].Statement)
	a.NoError(perr, "generated CTE backup must re-parse as valid SQL")

	// More than maxMixedDMLCount same-table statements use the UNION ALL path,
	// which cannot emit a per-arm WITH (TiDB rejects WITH after UNION ALL). A CTE
	// in that batch must be rejected rather than produce invalid SQL.
	manyWithCTE := "DELETE FROM test WHERE id = 1;\n" +
		"DELETE FROM test WHERE id = 2;\n" +
		"DELETE FROM test WHERE id = 3;\n" +
		"DELETE FROM test WHERE id = 4;\n" +
		"DELETE FROM test WHERE id = 5;\n" +
		"WITH x AS (SELECT id FROM test2) DELETE FROM test WHERE id > 0;"
	_, err = run(manyWithCTE)
	a.Error(err, "CTE in a >maxMixedDMLCount same-table batch must be rejected")

	// An unqualified SET column whose owner can't be resolved must fall back to
	// real tables only, never the CTE join source (a CTE can't be a mutation
	// target). "nonexistent_col" forces the metadata fallback path.
	result, err = run("WITH x AS (SELECT id FROM test2) UPDATE test JOIN x ON test.id = x.id SET nonexistent_col = 1")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal("test", result[0].SourceTableName, "backup target must be the real table, not the CTE")

	// A table alias colliding with a CTE name must resolve to the real table
	// (a CTE can't be a mutation target), not be filtered out.
	result, err = run("WITH x AS (SELECT id FROM test2) UPDATE test AS x JOIN test2 AS y ON x.a = y.a SET x.c = 1")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal("test", result[0].SourceTableName)

	// Multi-target mutation: EVERY mutated table must be backed up, even when an
	// alias collides with a CTE name. A partial backup (only test2) would leave
	// test unprotected.
	result, err = run("WITH x AS (SELECT id FROM test2) UPDATE test AS x JOIN test2 AS y ON x.a = y.a SET x.c = 1, y.c = 2")
	a.NoError(err)
	a.Len(result, 2)
	a.ElementsMatch([]string{"test", "test2"}, []string{result[0].SourceTableName, result[1].SourceTableName})

	// Multi-target DELETE: both deleted tables backed up despite the collision.
	result, err = run("WITH x AS (SELECT id FROM test2) DELETE x, y FROM test AS x JOIN test2 AS y WHERE x.a = y.a")
	a.NoError(err)
	a.Len(result, 2)
	a.ElementsMatch([]string{"test", "test2"}, []string{result[0].SourceTableName, result[1].SourceTableName})

	// Updating only a CTE has no real target -> must error (never silently skip).
	_, err = run("WITH x AS (SELECT id FROM test2) UPDATE test JOIN x ON test.id = x.id SET x.c = 1")
	a.Error(err, "updating only a CTE must error")

	// CTE names are matched case-insensitively (TiDB resolves a case-mismatched
	// reference to the CTE, which shadows any same-named real table). The CTE
	// "X" referenced as "x" must be excluded from the unqualified-column fallback
	// candidates, so the backup targets only the real table test (not db.x).
	result, err = run("WITH X AS (SELECT id FROM test2) UPDATE test JOIN x ON test.id = x.id SET missing_col = 1")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal("test", result[0].SourceTableName)

	// A schema-qualified physical table is never a CTE, even when its name
	// matches a CTE in scope (a CTE reference is always unqualified). Both
	// physical tables (db.test via alias p, test2 via alias y) are mutated, so
	// both must be backed up; skipping db.test would leave it unprotected
	// (verified on TiDB v8.5.0: both physical tables are updated/deleted).
	result, err = run("WITH test AS (SELECT a FROM test2) UPDATE db.test AS p JOIN test2 AS y ON p.a = y.a SET p.c = 1, y.c = 2")
	a.NoError(err)
	a.Len(result, 2)
	a.ElementsMatch([]string{"test", "test2"}, []string{result[0].SourceTableName, result[1].SourceTableName})
	for _, r := range result {
		_, perr := ParseTiDBOmni(r.Statement)
		a.NoError(perr, "generated multi-target backup must re-parse as valid SQL")
	}

	result, err = run("WITH test AS (SELECT a FROM test2) DELETE p, y FROM db.test AS p JOIN test2 AS y WHERE p.a = y.a")
	a.NoError(err)
	a.Len(result, 2)
	a.ElementsMatch([]string{"test", "test2"}, []string{result[0].SourceTableName, result[1].SourceTableName})
	for _, r := range result {
		_, perr := ParseTiDBOmni(r.Statement)
		a.NoError(perr, "generated multi-target backup must re-parse as valid SQL")
	}

	// The cross-database guard is case-insensitive (TiDB default): a different-
	// case reference to the task database is the same database, not cross-db.
	result, err = run("UPDATE DB.test SET c1 = 1 WHERE c1 = 2")
	a.NoError(err)
	a.Len(result, 1)
	a.Equal("test", result[0].SourceTableName, "case-only db difference must be treated as same database")

	// EXPLAIN ANALYZE executes the DML (modifying data) but can't be backed up,
	// so it must be rejected. Plain EXPLAIN does not execute -> no backup needed.
	_, err = run("EXPLAIN ANALYZE UPDATE test SET c1 = 1 WHERE c1 = 2")
	a.Error(err, "EXPLAIN ANALYZE of a DML must be rejected")
	result, err = run("EXPLAIN UPDATE test SET c1 = 1 WHERE c1 = 2")
	a.NoError(err, "plain EXPLAIN does not execute and needs no backup")
	a.Empty(result)

	// REPLACE and INSERT ... ON DUPLICATE KEY UPDATE overwrite existing rows but
	// have no backup support, so they must be rejected (not silently skipped). A
	// plain INSERT only adds rows and needs no backup.
	_, err = run("REPLACE INTO test VALUES (1, 2, 3)")
	a.Error(err, "REPLACE must be rejected")
	_, err = run("INSERT INTO test VALUES (1, 2, 3) ON DUPLICATE KEY UPDATE c = 5")
	a.Error(err, "INSERT ... ON DUPLICATE KEY UPDATE must be rejected")
	result, err = run("INSERT INTO test VALUES (1, 2, 3)")
	a.NoError(err, "plain INSERT only adds rows and needs no backup")
	a.Empty(result)

	// In the >maxMixedDMLCount same-table UNION path, case-only database
	// differences (db.test vs DB.test) must be treated as the same table, not
	// rejected as "different tables" — consistent with the cross-database guard.
	mixedCaseDB := "UPDATE db.test SET c1 = 1 WHERE id = 1;\n" +
		"UPDATE DB.test SET c1 = 1 WHERE id = 2;\n" +
		"UPDATE db.test SET c1 = 1 WHERE id = 3;\n" +
		"UPDATE DB.test SET c1 = 1 WHERE id = 4;\n" +
		"UPDATE db.test SET c1 = 1 WHERE id = 5;\n" +
		"UPDATE DB.test SET c1 = 1 WHERE id = 6;"
	result, err = run(mixedCaseDB)
	a.NoError(err, "case-only db differences must not be treated as different tables")
	a.Len(result, 1)
}

// TestBackupHonorsCaseSensitivity pins that the cross-database guard and
// equalTable respect the instance's IsCaseSensitive flag (consistent with
// classifyColumns), so the same comparisons fold case or not as configured.
func TestBackupHonorsCaseSensitivity(t *testing.T) {
	a := require.New(t)
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	run := func(caseSensitive bool, sql string) ([]base.BackupStatement, error) {
		return TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         caseSensitive,
		}, sql, "db", "backupDB", "_rollback")
	}

	// Cross-database guard: DB.test under task db is the same database when
	// case-insensitive (allowed) but a different database when case-sensitive
	// (rejected).
	_, err := run(false, "UPDATE DB.test SET c = 1 WHERE c = 2")
	a.NoError(err, "case-insensitive: DB.test is db.test")
	_, err = run(true, "UPDATE DB.test SET c = 1 WHERE c = 2")
	a.Error(err, "case-sensitive: DB.test is a different database than db")

	// equalTable (the >maxMixedDMLCount UNION path): test and Test merge into one
	// backup when case-insensitive, but are distinct tables (rejected) when
	// case-sensitive.
	sixMixedCase := "DELETE FROM test WHERE a = 1;\n" +
		"DELETE FROM test WHERE a = 2;\n" +
		"DELETE FROM test WHERE a = 3;\n" +
		"DELETE FROM test WHERE a = 4;\n" +
		"DELETE FROM test WHERE a = 5;\n" +
		"DELETE FROM Test WHERE a = 6;"
	_, err = run(false, sixMixedCase)
	a.NoError(err, "case-insensitive: test and Test are the same table")
	_, err = run(true, sixMixedCase)
	a.Error(err, "case-sensitive: test and Test are different tables")
}
