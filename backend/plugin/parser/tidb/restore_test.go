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
