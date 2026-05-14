package tidb

import (
	"context"
	"io"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type restoreCase struct {
	Input            string
	BackupDatabase   string
	BackupTable      string
	OriginalDatabase string
	OriginalTable    string
	Result           string
}

func TestRestore(t *testing.T) {
	tests := []restoreCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_restore.yaml"
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
		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         false,
		}, t.Input, &store.PriorBackupDetail_Item{
			SourceTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/" + t.OriginalDatabase,
				Table:    t.OriginalTable,
			},
			TargetTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/" + t.BackupDatabase,
				Table:    t.BackupTable,
			},
			StartPosition: &store.Position{
				Line:   0,
				Column: 0,
			},
			EndPosition: &store.Position{
				Line:   math.MaxInt32,
				Column: 0,
			},
		})
		a.NoError(err)

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

func TestTiDBGenerateRestoreSQLRegistration(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	result, err := base.GenerateRestoreSQL(context.Background(), store.Engine_TIDB, base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE b1 = 1;", &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_test",
		},
		StartPosition: &store.Position{
			Line:   0,
			Column: 0,
		},
		EndPosition: &store.Position{
			Line:   math.MaxInt32,
			Column: 0,
		},
	})

	require.NoError(t, err)
	require.Equal(t, "/*\nOriginal SQL:\nDELETE FROM test WHERE b1 = 1;\n*/\nINSERT INTO `db`.`test` SELECT * FROM `bbarchive`.`prefix_test`;", result)
}

// TestGenerateRestoreSQLCaseInsensitiveDatabaseQualifier pins the
// case-insensitive database-qualifier match contract in
// tableExprReferences. TiDB/MySQL identifier comparisons are typically
// case-insensitive in practice; the table-name side already uses
// EqualFold. Codex P2 catch on PR #20345 — the database side previously
// used == which silently missed schema-qualified references where the
// SQL's qualifier case differed from the backup item's stored case.
//
// Construct: SQL uses uppercase `DB.test`; backupItem stores lowercase
// `db`. Pre-fix, containsTable returned false, extractStatement returned
// empty, and the customer saw "no DML statement found" instead of the
// expected rollback SQL.
func TestGenerateRestoreSQLCaseInsensitiveDatabaseQualifier(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM DB.test WHERE c = 1;", &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.NoError(t, err,
		"case-mismatched database qualifier (DB vs db) must still match — TiDB identifier comparison is case-insensitive in practice")
	require.Contains(t, result, "INSERT INTO `db`.`test`",
		"generated rollback should target the lowercase database name from the backup item")
}

// TestGenerateRestoreSQLSelfJoinUpdate pins deterministic alias resolution
// for self-join UPDATEs. Per Codex P1 catch on PR #20345.
//
// Pre-fix bug: extractSingleTablesFromTableExprs returns
// map[alias]*TableReference; the loop in generateUpdateRestore picked
// matchedTable by ranging over that map (Go map iteration is randomized).
// For a self-join `UPDATE test t1 JOIN test t2 ...`, both t1 and t2
// satisfy the `table.Table == originalTable` check, so either could be
// picked. If t2 was picked but the SET clause only references t1, then
// extractUpdateColumns returned EMPTY (col.Table="t1" doesn't match
// matchedTable.Alias="t2", and col.Table doesn't equal-fold matchedTable.
// Table="test" either). Empty updateColumns produces invalid rollback
// SQL: `... ON DUPLICATE KEY UPDATE ;` (semicolon directly after UPDATE).
//
// Fix: extractUpdateColumns now takes the full singleTables map; for
// each qualified SET column it looks up the qualifier directly to
// determine whether the qualifier resolves to the original table —
// no need to pre-pick a single matchedTable.
//
// We loop the test 50 times to expose the nondeterminism if the fix
// regresses; one iteration is sufficient post-fix (deterministic).
func TestGenerateRestoreSQLSelfJoinUpdate(t *testing.T) {
	const input = "UPDATE test t1 JOIN test t2 ON t1.c = t2.c SET t1.a = 1 WHERE t1.c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	for i := 0; i < 50; i++ {
		result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         false,
		}, input, &store.PriorBackupDetail_Item{
			SourceTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/db",
				Table:    "test",
			},
			TargetTable: &store.PriorBackupDetail_Item_Table{
				Database: "instances/i1/databases/bbarchive",
				Table:    "prefix_1_test",
			},
			StartPosition: &store.Position{Line: 0, Column: 0},
			EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
		})

		require.NoError(t, err, "iteration %d: GenerateRestoreSQL must succeed for self-join UPDATE", i)
		require.Contains(t, result, "`a` = VALUES(`a`)",
			"iteration %d: ODKU must mention `a` (from SET t1.a = 1); empty ODKU produces invalid SQL", i)
		require.NotContains(t, result, "ON DUPLICATE KEY UPDATE ;",
			"iteration %d: ODKU clause must not be empty (would be invalid TiDB SQL)", i)
	}
}

// TestGenerateRestoreSQLSelfJoinMixedCaseAlias pins case-insensitive
// alias-map matching. Per Codex P1 follow-on catch on PR #20345 (review
// of the Fix #4 self-join determinism patch).
//
// Pre-fix bug: collectTableRefs stored map keys verbatim from
// omni AST (preserving the user's case), and extractUpdateColumns did
// an exact-string map lookup. TiDB ACCEPTS statements like
// `UPDATE test T1 JOIN test T2 ON T1.c = T2.c SET t1.a = 1`
// (alias referenced with different case in SET), but the map lookup
// `singleTables["t1"]` against entries keyed "T1"/"T2" missed.
// Fallback `EqualFold(col.Table, originalTable)` also missed
// (`EqualFold("t1","test") == false`). Result: empty updateColumns
// → invalid `... ON DUPLICATE KEY UPDATE ;`.
//
// Codex validated TiDB acceptance by executing the input against
// pingcap/tidb:v8.5.5 directly; mixed-case aliases are valid TiDB
// SQL.
//
// Fix: normalize map keys to lowercase on insert AND lookup.
func TestGenerateRestoreSQLSelfJoinMixedCaseAlias(t *testing.T) {
	const input = "UPDATE test T1 JOIN test T2 ON T1.c = T2.c SET t1.a = 1 WHERE T1.c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.NoError(t, err)
	require.Contains(t, result, "`a` = VALUES(`a`)",
		"ODKU must mention `a` (from SET t1.a = 1) — alias-map lookup must be case-insensitive")
	require.NotContains(t, result, "ON DUPLICATE KEY UPDATE ;",
		"ODKU clause must not be empty (would be invalid TiDB SQL)")
}

// TestGenerateRestoreSQLJoinedNotMutated pins that an UPDATE which
// references the target table only via JOIN (not as a SET-clause
// qualifier) does NOT generate rollback SQL for that table. Per peer
// review on PR #20345.
//
// Pre-fix bug: containsTable matched the target if it appeared in
// n.Tables (which includes JOIN-only refs). For
// `UPDATE test2 JOIN test ON ... SET test2.a = ...` with backupItem
// targeting `test`, the match incorrectly fired; extractUpdateColumns
// then correctly returned no `test`-table columns; doGenerate emitted
// invalid `... ON DUPLICATE KEY UPDATE ;` for `test`.
//
// Post-fix: containsTable's UpdateStmt arm uses updateMutatesTable,
// which checks SET-clause qualifiers (resolved through the alias map)
// — JOIN-only refs no longer trigger rollback for the wrong table.
func TestGenerateRestoreSQLJoinedNotMutated(t *testing.T) {
	const input = "UPDATE test2 JOIN test ON test2.c = test.c SET test2.a = 1 WHERE test2.c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test", // backup targets test, but test is JOINED not UPDATED
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	// The UPDATE doesn't mutate `test` (only joins it), so no DML matches
	// the backup_item's target. Pre-fix, this produced invalid SQL with
	// empty ODKU; post-fix, the "no DML statement found" error is the
	// correct outcome (there's genuinely no DML to roll back for `test`).
	require.Error(t, err,
		"backup item for joined-only-not-mutated table must not generate (invalid) rollback SQL")
	require.Contains(t, err.Error(), "no DML statement found",
		"expected no-DML error since test is only joined, not mutated")
}

// TestGenerateRestoreSQLGeneratedColumnUKSkipped pins that
// hasDisjointUniqueKey skips unique keys whose Expressions reference
// generated columns (or non-column expressions like functional indexes).
// Per peer review on PR #20345 (Finding 4).
//
// Pre-fix bug: disjoint() did naive string comparison between
// index.Expressions and the SET-columns map. For a UK on c_generated
// (where c_generated = a + b), updating columns {a, b} appeared
// "disjoint" from the UK's Expressions ["c_generated"] — string `c_generated`
// is not in {a, b}. But updating a or b CHANGES c_generated's value,
// so the UK is NOT safe for ON DUPLICATE KEY UPDATE matching: pre-fix
// would generate rollback SQL that silently fails to match rows.
//
// Post-fix: hasDisjointUniqueKey filters out UKs whose Expressions
// don't all map to regular (non-generated) columns. For this input,
// PK on b and UK on a both overlap with SET {a, b}; the UK on
// c_generated is skipped (generated). No disjoint UK remains, so
// the function returns the "no disjoint unique key found" error
// instead of generating unsafe rollback SQL.
//
// Mock setup: t_generated has PK on b, UK on a, AND a new UK on
// c_generated (added in backup_test.go for this test).
func TestGenerateRestoreSQLGeneratedColumnUKSkipped(t *testing.T) {
	const input = "UPDATE t_generated SET a = 1, b = 2 WHERE c_generated = 3;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "t_generated",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_t_generated",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err,
		"UPDATE that overlaps every regular UK must surface a no-disjoint-key error — the c_generated UK is not safe (its value depends on the SET'd columns)")
	require.Contains(t, err.Error(), "no disjoint unique key found",
		"error must come from hasDisjointUniqueKey, not a different failure mode")
}

// TestGenerateRestoreSQLDeleteUsingOnlyTable pins that a DELETE which
// references the target table only via USING (not as a delete-target in
// n.Tables) does NOT generate rollback SQL for that table. Per Codex
// P1 follow-on catch on PR #20345 — symmetric to the UPDATE-side
// joined-not-mutated fix in commit c4042da055.
//
// Pre-fix bug: containsTable's DeleteStmt arm matched n.Tables OR
// n.Using. For `DELETE test FROM test, test2 as t2 ...` with
// backupItem targeting `test2`, the match incorrectly fired (test2 is
// only in the USING-equivalent filter set). doGenerate emitted
// `INSERT INTO test2 SELECT * FROM bbarchive.prefix_test2` — which
// would re-introduce stale data if applied as rollback (test2 was
// never actually deleted).
//
// Post-fix: n.Tables is the explicit delete-target set; n.Using is
// filter-only and no longer triggers a match.
func TestGenerateRestoreSQLDeleteUsingOnlyTable(t *testing.T) {
	const input = "DELETE test FROM test, test2 as t2 where test.id = t2.id;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test2", // backup targets USING-only table
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test2",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err,
		"backup item for USING-only-not-deleted table must not generate (re-stale) rollback SQL")
	require.Contains(t, err.Error(), "no DML statement found",
		"expected no-DML error since test2 is only joined for filter, not deleted")
}

// TestGenerateRestoreSQLUnqualifiedSetNonTargetColumn pins that an
// unqualified SET column that does NOT belong to the target table does
// NOT trigger rollback for the target. Per Codex P1 catch on PR #20345
// — follow-on to the Bug 6 fix.
//
// Pre-fix bug: updateMutatesTable's unqualified branch returned true
// for any unqualified SET when the target was in scope, regardless of
// whether the column actually existed on the target. For
// `UPDATE test JOIN t1 ... SET name = 1` (where `name` exists on the
// joined table but not on `test`), the over-classification made
// containsTable return true; extractUpdateColumns correctly returned
// no `test`-table columns; doGenerate emitted invalid
// `... ON DUPLICATE KEY UPDATE ;` for `test`.
//
// Post-fix: updateMutatesTable resolves unqualified SET columns
// against the target table's actual normal column set (fetched once
// at the top of GenerateRestoreSQL via getNormalColumnsLower). Non-
// target columns no longer trigger a match.
func TestGenerateRestoreSQLUnqualifiedSetNonTargetColumn(t *testing.T) {
	// `name` doesn't exist on `test` (mock has only a, b, c).
	const input = "UPDATE test JOIN t1 ON test.c = t1.c SET name = 1 WHERE test.c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err,
		"unqualified SET on non-target-table column must not generate (invalid) rollback SQL")
	require.Contains(t, err.Error(), "no DML statement found",
		"expected no-DML error since SET column doesn't exist on target")
}

// TestGenerateRestoreSQLUnqualifiedSetTargetColumn pins the positive-
// path counterpart: when the unqualified SET column DOES exist on the
// target table, rollback is correctly generated (column resolves
// through targetCols and `extractUpdateColumns` finds it in
// normalColumns).
func TestGenerateRestoreSQLUnqualifiedSetTargetColumn(t *testing.T) {
	// Single-table UPDATE with unqualified SET — the common case.
	// `a` exists on `test` (mock has a, b, c). targetCols includes
	// "a" → updateMutatesTable returns true; extractUpdateColumns
	// returns ["a"]; ODKU UPDATE `a` = VALUES(`a`).
	const input = "UPDATE test SET a = 1 WHERE c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.NoError(t, err)
	require.Contains(t, result, "`a` = VALUES(`a`)",
		"unqualified SET on target's column must produce ODKU clause for that column")
}

// TestGenerateRestoreSQLCrossDatabaseAliasResolution pins that the
// alias-map qualifier resolution checks BOTH Database AND Table when
// matching a SET clause against the backup target. Per Codex P1 catch
// on PR #20345.
//
// Pre-fix bug: the alias-map lookup compared only entry.Table. For a
// cross-database join with homonymous tables —
//
//	UPDATE db.test t1 JOIN otherdb.test t2 ON t1.c = t2.c SET t2.a = 1 WHERE t1.c = 1;
//
// — alias `t2` resolves to {Database: "otherdb", Table: "test"}.
// updateMutatesTable's condition `entry.Table == "test"` matched
// (because the table NAME is "test"), so a backup item targeting
// db.test mis-classified the UPDATE as mutating db.test even though
// the SET only touched otherdb.test. extractUpdateColumns had the
// same bug, so the rollback would have been generated for the wrong
// table.
//
// Post-fix: both branches require `entry.Database == database` AND
// `entry.Table == table` (case-insensitive). Cross-DB SETs no longer
// match.
func TestGenerateRestoreSQLCrossDatabaseAliasResolution(t *testing.T) {
	const input = "UPDATE db.test t1 JOIN otherdb.test t2 ON t1.c = t2.c SET t2.a = 1 WHERE t1.c = 1;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db", // backup targets db.test
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err,
		"cross-DB alias (t2 → otherdb.test) must not match a backup item targeting db.test")
	require.Contains(t, err.Error(), "no DML statement found",
		"expected no-DML error since SET is on otherdb.test, not db.test")
}

// TestGenerateRestoreSQLNilBackupItemGuards pins that GenerateRestoreSQL
// returns a clean error (not a panic) when backupItem or its sub-fields
// are nil. Per Codex P2 catch on PR #20345 — Bug 9's plumbing refactor
// moved metadata-fetching ahead of extractStatement (which previously
// held the nil guard), causing the function to panic on nil input.
//
// Three independent nil paths covered: backupItem itself, SourceTable,
// TargetTable.
func TestGenerateRestoreSQLNilBackupItemGuards(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	rCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}

	cases := []struct {
		name       string
		backupItem *store.PriorBackupDetail_Item
		wantErr    string
	}{
		{
			name:       "nil backupItem",
			backupItem: nil,
			wantErr:    "backup item is nil",
		},
		{
			name: "nil SourceTable",
			backupItem: &store.PriorBackupDetail_Item{
				SourceTable: nil,
				TargetTable: &store.PriorBackupDetail_Item_Table{
					Database: "instances/i1/databases/bbarchive",
					Table:    "prefix_1_test",
				},
			},
			wantErr: "backup item source table is nil",
		},
		{
			name: "nil TargetTable",
			backupItem: &store.PriorBackupDetail_Item{
				SourceTable: &store.PriorBackupDetail_Item_Table{
					Database: "instances/i1/databases/db",
					Table:    "test",
				},
				TargetTable: nil,
			},
			wantErr: "backup item target table is nil",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := GenerateRestoreSQL(context.Background(), rCtx, "DELETE FROM test;", tc.backupItem)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}, "must return error, not panic, on nil input")
		})
	}
}

// TestGenerateRestoreSQLEmptyExpressionsUKSkipped pins that
// hasDisjointUniqueKey skips unique keys with empty Expressions. Per
// Codex P1 catch on PR #20345.
//
// Pre-fix bug: `disjoint([], anyMap)` returns vacuously true (the
// for-loop iterates zero times). So a UK with empty Expressions
// passed through the disjoint check as "safe" — even though it has
// no actual columns to match against. TiDB metadata produces empty
// Expressions for some expression-based index parts (per
// backend/plugin/schema/tidb/get_database_metadata.go's
// getIndexColumnsInfo, parts without key.Column aren't appended).
// Pre-fix the empty-Expressions UK would short-circuit
// hasDisjointUniqueKey to true → unsafe rollback SQL generated.
//
// Post-fix: explicit empty-Expressions check at the top of each UK
// iteration; such UKs are skipped (treated as overlapping).
//
// Mock setup (backup_test.go): t_generated has PK on b, UK on a,
// UK on c_generated (skipped per Bug 7), AND a new UK with empty
// Expressions. The test UPDATE overlaps the regular UKs (a and b
// in SET); pre-fix the empty-Expressions UK would false-positive
// as disjoint; post-fix all UKs are correctly classified as
// overlapping/unsafe → "no disjoint unique key found" error.
func TestGenerateRestoreSQLEmptyExpressionsUKSkipped(t *testing.T) {
	const input = "UPDATE t_generated SET a = 1, b = 2 WHERE c_generated = 3;"

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "t_generated",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_t_generated",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err,
		"empty-Expressions UK must NOT be treated as disjoint (vacuous truth bug)")
	require.Contains(t, err.Error(), "no disjoint unique key found",
		"all UKs must be classified as overlapping/unsafe — empty-Expressions UK skipped explicitly")
}

// TestGenerateRestoreSQLEndPositionExclusive pins the boundary semantic
// of backupItem.EndPosition: it is the EXCLUSIVE end (per
// base/statement.go:16-18, "points to the position AFTER the last
// character of the statement"). extractStatement's slice MUST exclude
// stmts whose Start position equals EndPosition — those belong to the
// NEXT backup item. Per Codex P1 catch on PR #20345.
//
// Why this matters now: pre-Fix-1 (multi-DML union), findFirstDML
// returned only the first DML, so even when extractStatement bled into
// the next stmt, only the first stmt's columns reached the rollback SQL.
// Post-Fix-1, findMatchingDMLs returns ALL matching DMLs from the
// extraction — so the boundary bleed now contributes columns from the
// NEXT backup item to the union, producing wrong rollback SQL (extra
// ODKU columns OR false "no disjoint unique key" errors).
//
// Construct: 2 same-table UPDATEs in one input. Use SplitSQL to get the
// actual position where stmt[0] ends; set backupItem.EndPosition to
// exactly that (mixed-DML mode where each backup item maps to ONE
// stmt). Assert the rollback ODKU mentions ONLY stmt[0]'s column (`a`)
// — not stmt[1]'s column (`b`).
func TestGenerateRestoreSQLEndPositionExclusive(t *testing.T) {
	const input = "UPDATE test SET a = 1 WHERE c = 1;\nUPDATE test SET b = 2 WHERE c = 2;"

	// Get actual stmt boundaries from the splitter.
	stmts, err := SplitSQL(input)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(stmts), 2, "splitter must produce at least 2 stmts")
	require.NotNil(t, stmts[0].End, "stmt[0].End must be set by splitter")

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		// Mixed-DML mode: this backup item covers ONLY stmt[0]. Setting
		// EndPosition to stmt[0].End (exclusive) MUST exclude stmt[1].
		EndPosition: stmts[0].End,
	})

	require.NoError(t, err)
	require.Contains(t, result, "`a` = VALUES(`a`)",
		"ODKU must contain stmt[0]'s column `a`")
	require.NotContains(t, result, "`b` = VALUES(`b`)",
		"ODKU must NOT contain stmt[1]'s column `b` — that stmt belongs to the NEXT backup item; including it would generate rollback for the wrong scope")
}

// TestGenerateRestoreSQLNoDisjointUniqueKey pins the negative path that the
// yaml golden tests cannot cover (their format is success-only — no
// WantError field). Rolling back an UPDATE requires at least one unique
// key whose columns are NOT in the SET clause; otherwise ON DUPLICATE KEY
// UPDATE has nothing to match against and the rollback is unsafe.
//
// Construct: the `test` table has PK on `c` and a unique key on `a`. An
// UPDATE that touches BOTH `a` and `c` overlaps every unique key, so
// hasDisjointUniqueKey returns false and generateUpdateRestore must
// surface the "no disjoint unique key found" error.
//
// The mysql analog (mysql/restore_test.go) currently lacks symmetric
// coverage; worth a small follow-up to mirror this test there for parity.
func TestGenerateRestoreSQLNoDisjointUniqueKey(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	_, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "UPDATE test SET a = 1, c = 2 WHERE b = 3;", &store.PriorBackupDetail_Item{
		SourceTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/db",
			Table:    "test",
		},
		TargetTable: &store.PriorBackupDetail_Item_Table{
			Database: "instances/i1/databases/bbarchive",
			Table:    "prefix_1_test",
		},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
	})

	require.Error(t, err, "UPDATE that touches every unique key must surface a no-disjoint-key error")
	require.Contains(t, err.Error(), "no disjoint unique key found",
		"error must be the no-disjoint-key error from generateUpdateRestore, not a different failure mode")
}
