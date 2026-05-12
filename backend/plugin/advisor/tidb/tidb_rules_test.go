package tidb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestTiDBRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[a-z]+(_[a-z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_COLUMN, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[a-z]+(_[a-z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_UK, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^uk_{{table}}_{{column_list}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_FK, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_IDX, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^idx_{{table}}_{{column_list}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^id$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 1000}}},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "_delete$"}}},
		{Type: storepb.SQLReviewRule_TABLE_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRED, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"id", "created_ts", "updated_ts", "creator_id", "updater_id"}}}},
		{Type: storepb.SQLReviewRule_COLUMN_NO_NULL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"JSON", "BINARY_FLOAT"}}}},
		{Type: storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 20}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 20}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"serial", "bigserial", "int", "bigint"}}}},
		{Type: storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4", "UTF8"}}}},
		{Type: storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4_0900_ai_ci"}}}},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_TIDB, false /* record */)
	}
}

// TestTiDBPriorBackupCheckAdvisor exercises the BUILTIN_PRIOR_BACKUP_CHECK
// advisor's full check pipeline. Modeled on the mysql analog's
// TestMariaDBPriorBackupCheckAdvisor — the standard fixture driver
// doesn't set ListDatabaseNamesFunc, so backup-database-existence and
// per-table DML-mixing checks need dedicated context construction.
//
// Coverage:
//   - Mixed DDL + DML: triggers "mixed DDL and DML" advice
//   - Backup db missing: triggers "Need database ... does not exist"
//   - Backup db present + plain UPDATE: no DML-mixing advice
//   - Per-table DML-type mixing (UPDATE + DELETE on same table):
//     triggers "mixed DML statements on the same table" advice
//   - Size cap exceeded: triggers size-limit advice
func TestTiDBPriorBackupCheckAdvisor(t *testing.T) {
	sm := sheet.NewManager()
	// Large statement for size-cap testing: must exceed
	// common.MaxSheetCheckSize (2 * 1024 * 1024 bytes). Padded
	// with a SQL comment so the statement parses cleanly.
	largeStatement := "UPDATE tech_book SET id = 1 WHERE id = 2; -- " + strings.Repeat("x", 2*1024*1024+100)
	cases := []struct {
		name              string
		statement         string
		backupDBPresent   bool
		wantContentSubstr []string
		wantNoneSubstr    []string
	}{
		{
			name:            "mixed DDL and DML fires",
			statement:       "CREATE TABLE t(id INT);\nUPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DDL and DML",
			},
			wantNoneSubstr: []string{
				"does not exist",
			},
		},
		{
			name:            "backup db missing fires",
			statement:       "UPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: false,
			wantContentSubstr: []string{
				"does not exist",
			},
		},
		{
			name:            "single UPDATE on table — clean",
			statement:       "UPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: true,
			wantNoneSubstr: []string{
				"mixed DDL and DML",
				"mixed DML statements",
				"does not exist",
				"exceeds the maximum limit",
			},
		},
		{
			name:            "per-table DML mixing (UPDATE + DELETE on same table) fires",
			statement:       "UPDATE tech_book SET id = 1 WHERE id = 2;\nDELETE FROM tech_book WHERE id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
				"tech_book",
			},
		},
		{
			name:            "UPDATE + DELETE on DIFFERENT tables — clean",
			statement:       "UPDATE tech_book SET id = 1 WHERE id = 2;\nDELETE FROM orders WHERE order_id = 3;",
			backupDBPresent: true,
			wantNoneSubstr: []string{
				"mixed DML statements",
			},
		},
		{
			name:            "size cap exceeded fires",
			statement:       largeStatement,
			backupDBPresent: true,
			wantContentSubstr: []string{
				"exceeds the maximum limit",
				"for backup",
			},
		},
		{
			// Cumulative #30 Codex-fix-1: UPDATE JOIN should NOT
			// false-positive on the joined-only read table.
			// `UPDATE t1 JOIN t2 ON ... SET t1.col=...` mutates t1
			// only; t2 is read-only. Following DELETE FROM t2 is
			// pure DELETE — no mixing on either table. Pre-fix code
			// tagged BOTH t1 and t2 as UPDATE targets, then matched
			// t2's UPDATE+DELETE → false-positive. Post-fix:
			// SET-clause-based target extraction → only t1 tagged
			// for UPDATE → no mixing.
			name:            "UPDATE-JOIN + DELETE-on-joined-table — no false-positive (Codex-fix-1)",
			statement:       "UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET tech_book.name = 'x';\nDELETE FROM orders WHERE order_id = 5;",
			backupDBPresent: true,
			wantNoneSubstr: []string{
				"mixed DML statements",
			},
		},
		{
			// Cumulative #30 Codex-fix-2: case-insensitive grouping.
			// `UPDATE tech_book ...; DELETE FROM Tech_Book ...`
			// references the same logical table with different
			// casing; pre-fix code split into two buckets and missed
			// the mixing. Post-fix: lowercased grouping key.
			name:            "case-insensitive grouping — Tech_Book ≡ tech_book (Codex-fix-2)",
			statement:       "UPDATE tech_book SET id = 1 WHERE id = 2;\nDELETE FROM Tech_Book WHERE id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := advisor.Context{
				DBType:            storepb.Engine_TIDB,
				DBSchema:          advisor.MockMySQLDatabase, // tidb tests reuse mysql mock catalog
				EnablePriorBackup: true,
				InstanceID:        "instance",
				ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
					if tc.backupDBPresent {
						return []string{"bbdataarchive"}, nil
					}
					return nil, nil
				},
			}
			rule := &storepb.SQLReviewRule{
				Type:  storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
				Level: storepb.SQLReviewRule_WARNING,
			}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, tc.statement, []*storepb.SQLReviewRule{rule}, ctx)
			require.NoError(t, err)
			joined := ""
			for _, a := range adviceList {
				joined += a.Content + "\n"
			}
			for _, want := range tc.wantContentSubstr {
				require.True(t, strings.Contains(joined, want),
					"expected advice content containing %q, got: %s", want, joined)
			}
			for _, unwanted := range tc.wantNoneSubstr {
				require.False(t, strings.Contains(joined, unwanted),
					"expected NO advice content containing %q, got: %s", unwanted, joined)
			}
		})
	}
}
