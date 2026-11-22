package mariadb

import (
	"context"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestMariaDBRules runs the rules this package registers for MariaDB over the
// fixtures in test/, so a rule that stops firing on MariaDB's own AST is caught
// here rather than letting a MariaDB review pass silently.
//
// Three of the 58 registered rules are not here. BUILTIN_PRIOR_BACKUP_CHECK and
// BUILTIN_WALK_THROUGH_CHECK have their own tests in this package, and
// ADVICE_ONLINE_MIGRATION has no fixture anywhere — it had none before either.
//
// The fixtures are advisor/mysql's, which the rules share.
func TestMariaDBRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, Level: storepb.SQLReviewRule_WARNING},
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
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "_delete$"}}},
		{Type: storepb.SQLReviewRule_TABLE_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRED, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"id", "created_ts", "updated_ts", "creator_id", "updater_id"}}}},
		// The same columns listed with duplicates must report the same advice.
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRED, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"id", "created_ts", "id", "updated_ts", "creator_id", "updater_id", "updater_id"}}}},
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
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 2560}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 20}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4", "UTF8"}}}},
		{Type: storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4_0900_ai_ci"}}}},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MARIADB, false /* record */)
	}
}

func checkTestRowLimitRule(t *testing.T, prober *advisor.ProbeRecorder, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, statement string) []*storepb.Advice {
	t.Helper()
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{{
		Type:    ruleType,
		Level:   level,
		Engine:  storepb.Engine_MARIADB,
		Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
	}}, advisor.Context{
		DBType:          storepb.Engine_MARIADB,
		Prober:          prober,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}

func TestMariaDBRowLimitAdvisorSeam(t *testing.T) {
	for _, tc := range []struct {
		name        string
		ruleType    storepb.SQLReviewRule_Type
		prober      *advisor.ProbeRecorder
		statement   string
		wantContent string
		wantCode    code.Code
	}{
		{
			name:      "affected row limit within limit",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			prober:    &advisor.ProbeRecorder{Rows: 3},
			statement: "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1 LIMIT 3;",
		},
		{
			name:        "affected row limit exceeded",
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			prober:      &advisor.ProbeRecorder{Rows: 1000},
			statement:   "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			wantContent: `"DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;" affected 1000 rows (estimated). The count exceeds 5.`,
			wantCode:    code.StatementAffectedRowExceedsLimit,
		},
		{
			name:        "insert row limit exceeded",
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			prober:      &advisor.ProbeRecorder{Rows: 100},
			statement:   "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			wantContent: `"INSERT INTO t2 SELECT * FROM t WHERE d = 3;" inserts 100 rows. The count exceeds 5.`,
			wantCode:    code.InsertTooManyRows,
		},
		{
			name:        "refused estimate reports internal count error",
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			prober:      &advisor.ProbeRecorder{EstimateErr: errors.New(`table "t2" in the plan has no row estimate`)},
			statement:   "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;",
			wantContent: `failed to get row count for "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;": table "t2" in the plan has no row estimate`,
			wantCode:    code.Internal,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adviceList := checkTestRowLimitRule(t, tc.prober, tc.ruleType, storepb.SQLReviewRule_WARNING, tc.statement)
			if tc.wantContent == "" {
				require.Empty(t, adviceList)
				return
			}
			require.Len(t, adviceList, 1)
			require.Equal(t, tc.wantCode.Int32(), adviceList[0].Code)
			require.Equal(t, tc.wantContent, adviceList[0].Content)
		})
	}

	t.Run("statements beyond explain limit emit warning", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{Rows: 1000}
		statement := strings.Repeat("UPDATE td SET c = 1;\n", common.MaximumLintExplainSize+2)
		adviceList := checkTestRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, statement)
		require.Len(t, adviceList, common.MaximumLintExplainSize+1)
		warning := adviceList[common.MaximumLintExplainSize]
		require.Equal(t, storepb.Advice_WARNING, warning.Status)
		require.Equal(t, code.StatementAffectedRowExceedsLimit.Int32(), warning.Code)
		require.Equal(t, "Only the first 10 statements were estimated; 2 more were not checked against the row limit.", warning.Content)
		require.Len(t, prober.Estimates, common.MaximumLintExplainSize)
	})
}

func TestMariaDBPriorBackupCheckAdvisor(t *testing.T) {
	sm := sheet.NewManager()
	for _, tc := range []struct {
		name        string
		statement   string
		wantSubstr  string
		wantEmptyOK bool
	}{
		{
			name:       "mixed DDL and DML",
			statement:  "CREATE TABLE t(id INT);\nUPDATE test SET c1 = 1 WHERE b1 = 1;",
			wantSubstr: "mixed DDL and DML",
		},
		{
			name:       "MariaDB sequence DDL mixed with DML",
			statement:  "CREATE SEQUENCE s START WITH 1;\nDELETE FROM test WHERE b1 = 1;",
			wantSubstr: "mixed DDL and DML",
		},
		{
			name:        "MariaDB-only DELETE RETURNING passes",
			statement:   "DELETE FROM test WHERE b1 = 2 RETURNING b1;",
			wantEmptyOK: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, tc.statement, nil, advisor.Context{
				DBType:            storepb.Engine_MARIADB,
				DBSchema:          advisor.MockMySQLDatabase,
				EnablePriorBackup: true,
				InstanceID:        "instance",
				ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
					return []string{"bbdataarchive"}, nil
				},
			})
			require.NoError(t, err)
			if tc.wantEmptyOK {
				require.Empty(t, adviceList)
				return
			}
			require.NotEmpty(t, adviceList)
			require.Equal(t, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(), adviceList[0].Title)
			require.Contains(t, adviceList[0].Content, tc.wantSubstr)
		})
	}
}

// TestMariaDBSQLReviewReviewsMariaDBOnlyStatements pins the other half: a rule
// fires on a statement only MariaDB accepts, rather than the review walking
// straight past it.
func TestMariaDBSQLReviewReviewsMariaDBOnlyStatements(t *testing.T) {
	sm := sheet.NewManager()
	finalMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "test"}, nil, nil, storepb.Engine_MARIADB, true)
	statement := "CREATE TABLE t (a INT);\nCREATE OR REPLACE TABLE u (a INT);"

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, statement, []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_ERROR, Engine: storepb.Engine_MARIADB},
	}, advisor.Context{
		DBType:        storepb.Engine_MARIADB,
		DBSchema:      advisor.MockMySQLDatabase,
		FinalMetadata: finalMetadata,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 2)
	var flagged []string
	for _, advice := range adviceList {
		require.Equal(t, code.TableNoPK.Int32(), advice.Code)
		flagged = append(flagged, advice.Content)
	}
	require.ElementsMatch(t, []string{"Table `t` requires PRIMARY KEY", "Table `u` requires PRIMARY KEY"}, flagged)
}
