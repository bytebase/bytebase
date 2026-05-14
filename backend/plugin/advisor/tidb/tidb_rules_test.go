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
		{
			// Cumulative #30 Codex-fix-1b: aliased single-table
			// UPDATE with unqualified SET. The lookup map contains
			// 2 entries (alias + bare name) but only 1 distinct
			// base table. Pre-fix-1b code gated on `len(lookup) == 1`
			// → skipped attribution → following DELETE on same
			// table wasn't flagged as mixed. Post-fix-1b: gate on
			// `len(distinctBases) == 1` → attribution correctly
			// records UPDATE on tech_book → DELETE → mixed-DML fires.
			name:            "aliased single-table UPDATE with unqualified SET (Codex-fix-1b)",
			statement:       "UPDATE tech_book AS t SET id = 1 WHERE id = 2;\nDELETE FROM tech_book WHERE id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
			},
		},
		{
			// Cumulative #30 Codex-fix-2 (revised): qualified vs
			// unqualified split when CurrentDatabase is unset
			// (plancheck path). Pre-fix used CurrentDatabase as the
			// default fallback, which is empty in
			// statement_advise_executor.go:168-180. Post-fix uses
			// DBSchema.Name as the default (the schema being checked,
			// reliably populated across review paths). Mock catalog's
			// DBSchema.Name is "test", so unqualified `tech_book`
			// resolves to `test.tech_book` and matches qualified
			// `test.tech_book` (case-insensitive).
			name:            "qualified vs unqualified same-table mixing (Codex-fix-2 revised)",
			statement:       "UPDATE test.tech_book SET id = 1 WHERE id = 2;\nDELETE FROM tech_book WHERE id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
			},
		},
		{
			// Cumulative #30 Codex-fix-1c: DROP VIEW must be classified
			// as DDL. Pingcap parses `DROP VIEW v` as *ast.DropTableStmt
			// (already in DDL list); omni splits to *ast.DropViewStmt
			// which initial port excluded → regression. Post-fix:
			// DropViewStmt added to DDL set.
			name:            "DROP VIEW + UPDATE fires mixed DDL+DML (Codex-fix-1c)",
			statement:       "DROP VIEW v;\nUPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DDL and DML",
			},
		},
		{
			// Cumulative #30 Codex-fix-1c: ALTER DATABASE must be
			// classified as DDL. Pingcap implements DDLNode via
			// ddlNode struct embedding (ast/base.go:81); peer's
			// compile-time `_ DDLNode = ...` grep missed this →
			// initial port excluded AlterDatabaseStmt → regression.
			// Empirical verification: parse + .(ast.DDLNode) returns
			// true for AlterDatabaseStmt.
			name:            "ALTER DATABASE + UPDATE fires mixed DDL+DML (Codex-fix-1c)",
			statement:       "ALTER DATABASE test COLLATE = utf8mb4_bin;\nUPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DDL and DML",
			},
		},
		{
			// Cumulative #30 Codex-fix-1d: schema-qualified UPDATE
			// target disambiguation. Joined same-bare-name tables
			// across schemas (`db1.tech_book JOIN db2.tech_book`):
			// pre-fix lookup-by-bare-name overwrote the first entry
			// with the second, so `SET db1.tech_book.id = 1` resolved
			// to db2.tech_book (wrong). Following `DELETE FROM
			// db1.tech_book` → DELETE on db1; UPDATE went to db2 →
			// no mixing detected → false-negative. Post-fix: separate
			// bySchemaName lookup keyed by "schema.name" disambiguates.
			//
			// Note: this fixture uses two parsed-but-non-existent
			// databases (db1, db2) — the advisor's grouping logic
			// runs purely on AST-level table identifiers, not on
			// catalog schema existence. The mixed-DML detection
			// fires regardless of whether the named databases exist.
			name:            "schema-qualified UPDATE target disambiguation (Codex-fix-1d)",
			statement:       "UPDATE db1.tech_book JOIN db2.tech_book ON db1.tech_book.id = db2.tech_book.id SET db1.tech_book.id = 1;\nDELETE FROM db1.tech_book WHERE id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
				"db1.tech_book",
			},
		},
		{
			// Cumulative #30 Codex-fix-1g: count-cap with multi-table
			// gate. The backup transformer at parser/tidb/backup.go:
			// 96-110 routes >5 DML statements into single-table-only
			// path which errors on multi-table inputs. The advisor
			// must catch this pre-execution.
			//
			// 6 UPDATEs across two tables (tech_book × 3 + orders × 3)
			// → must fire "more than 5 DML statements across different
			// tables".
			name: "6+ DML across multiple tables fires count-cap (Codex-fix-1g)",
			statement: "UPDATE tech_book SET id = 1 WHERE id = 1;\n" +
				"UPDATE tech_book SET id = 2 WHERE id = 2;\n" +
				"UPDATE tech_book SET id = 3 WHERE id = 3;\n" +
				"UPDATE orders SET order_id = 4 WHERE order_id = 4;\n" +
				"UPDATE orders SET order_id = 5 WHERE order_id = 5;\n" +
				"UPDATE orders SET order_id = 6 WHERE order_id = 6;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"more than 5 DML statements across different tables",
			},
		},
		{
			// Cumulative #30 Codex-fix-1g: single-table batches >5
			// are still OK — the transformer's
			// generateSQLForSingleTable handles them successfully.
			// 6 UPDATEs on tech_book only → no count-cap advice.
			name: "6+ DML on single table — no count-cap advice (Codex-fix-1g)",
			statement: "UPDATE tech_book SET id = 1 WHERE id = 1;\n" +
				"UPDATE tech_book SET id = 2 WHERE id = 2;\n" +
				"UPDATE tech_book SET id = 3 WHERE id = 3;\n" +
				"UPDATE tech_book SET id = 4 WHERE id = 4;\n" +
				"UPDATE tech_book SET id = 5 WHERE id = 5;\n" +
				"UPDATE tech_book SET id = 6 WHERE id = 6;",
			backupDBPresent: true,
			wantNoneSubstr: []string{
				"more than 5 DML statements",
			},
		},
		{
			// Cumulative #30 Codex-fix-1f: Tier-4-deferred DDL
			// (CREATE/ALTER/DROP SEQUENCE) is omni-rejected at
			// parse time. Pre-fix-1f used omniIsDDLStmt on
			// omni-parsed stmts → soft-fail skipped the SEQUENCE
			// stmt → DDL detection missed → no mixed-DDL advice
			// despite the DML+DDL mixing. Post-fix-1f: DDL
			// detection uses pingcap's DDLNode interface via
			// `getTiDBNodes` → pingcap parses CreateSequenceStmt
			// successfully → DDL detected → advice fires.
			name:            "CREATE SEQUENCE + UPDATE fires mixed DDL+DML via pingcap path (Codex-fix-1f)",
			statement:       "CREATE SEQUENCE seq1 START 1 INCREMENT BY 1;\nUPDATE tech_book SET id = 1 WHERE id = 2;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DDL and DML",
			},
		},
		{
			// Cumulative #30 Codex-fix-1f: FLASHBACK family is
			// Tier-4-deferred grammar in omni. Pingcap handles
			// FLASHBACK TABLE / FLASHBACK DATABASE as DDL via
			// the DDLNode interface. Post-fix-1f: pingcap path
			// catches it; pre-fix-1f silently skipped.
			//
			// Note: pingcap classifies FLASHBACK TABLE / DATABASE
			// as DDL via ddlNode struct embedding. Verified by
			// parse-test in batch 19 reshape investigation.
			name:            "FLASHBACK TABLE + DELETE fires mixed DDL+DML via pingcap path (Codex-fix-1f)",
			statement:       "FLASHBACK TABLE tech_book TO tech_book_old;\nDELETE FROM orders WHERE order_id = 5;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DDL and DML",
			},
		},
		{
			// Cumulative #30 Codex-fix-1h: multi-match fallback to
			// all matched tables (mirrors transformer's
			// resolveUnqualifiedColumns at backup.go:539-576).
			// `name` column exists on BOTH tech_book and orders in
			// MockMySQLDatabase. `UPDATE tech_book JOIN orders SET
			// name = 'x'` → resolver returns BOTH tables. Then
			// `DELETE FROM tech_book` makes tech_book mixed
			// (UPDATE+DELETE) → mixed-DML fires on tech_book.
			// Pre-fix-1h: resolver returned nil for multi-match →
			// no UPDATE targets → no mixing detected → false-negative.
			name:            "multi-match unqualified SET attributes to all (Codex-fix-1h)",
			statement:       "UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET name = 'x';\nDELETE FROM tech_book WHERE id = 5;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
				"tech_book",
			},
		},
		{
			// Cumulative #30 Codex-fix-1h: zero-match fallback to
			// all distinctBases. `notarealcolumn` doesn't exist on
			// any joined table; transformer falls back to all
			// tables. Six unqualified-SET multi-table UPDATEs with
			// no-catalog-match column → count-cap fires (each
			// contributes 2 distinctBases entries; total > 5 with
			// distinct tables > 1).
			//
			// Pre-fix-1h: resolver returned nil → 0 dmlRefs → no
			// count-cap fires → advisor approves SQL that the
			// transformer would reject at runtime.
			//
			// Note: omni's parser permits unknown column names in
			// SET (lazy column validation); the catalog walk simply
			// fails to find them and falls back.
			name: "zero-match unqualified SET falls back to all (Codex-fix-1h count-cap path)",
			statement: "UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET notarealcolumn = 1 WHERE tech_book.id = 1;\n" +
				"UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET notarealcolumn = 2 WHERE tech_book.id = 2;\n" +
				"UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET notarealcolumn = 3 WHERE tech_book.id = 3;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"more than 5 DML statements across different tables",
			},
		},
		{
			// Cumulative #30 Codex-fix-1e: schema-aware column
			// resolution for unqualified SET in multi-table UPDATE.
			// `UPDATE tech_book JOIN orders SET customer_name = 'x'`
			// has unqualified `customer_name` — which exists on
			// `orders` only (per MockMySQLDatabase). Pre-fix-1e:
			// multi-target UPDATE with unqualified SET → skip → no
			// UPDATE target recorded → following DELETE on orders
			// would not fire as mixed-DML (false-negative).
			// Post-fix-1e: omniResolveUnqualifiedSETColumn walks
			// dbMetadata, finds customer_name on orders → UPDATE
			// attributed to orders → DELETE on orders → mixed-DML
			// fires.
			name:            "unqualified SET in multi-table UPDATE resolves via catalog (Codex-fix-1e)",
			statement:       "UPDATE tech_book INNER JOIN orders ON tech_book.id = orders.order_id SET customer_name = 'x';\nDELETE FROM orders WHERE order_id = 5;",
			backupDBPresent: true,
			wantContentSubstr: []string{
				"mixed DML statements on the same table",
				"orders",
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
