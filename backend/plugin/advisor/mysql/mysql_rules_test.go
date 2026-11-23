package mysql

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

func TestMySQLRules(t *testing.T) {
	rules := []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_NAMING_TABLE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[a-z]+(_[a-z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_COLUMN, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^[a-z]+(_[a-z]+)*$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_UK, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^uk_{{table}}_{{column_list}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_FK, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_INDEX_IDX, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^$|^idx_{{table}}_{{column_list}}$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "^id$", MaxLength: 64}}},
		{Type: storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, Level: storepb.SQLReviewRule_WARNING},
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
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 2}}},
		{Type: storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_PK, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NamingPayload{NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{Format: "_delete$"}}},
		{Type: storepb.SQLReviewRule_TABLE_COMMENT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_CommentConventionPayload{CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{Required: true, MaxLength: 10}}},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_DDL, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"t", "u"}}}},
		{Type: storepb.SQLReviewRule_TABLE_DISALLOW_DML, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"t", "u"}}}},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION, Level: storepb.SQLReviewRule_WARNING},
		// The mock tables report 0 rows, so a limit of 0 is the one that fires.
		{Type: storepb.SQLReviewRule_TABLE_LIMIT_SIZE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 0}}},
		{Type: storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 1000}}},
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
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}}},
		{Type: storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"serial", "bigserial", "int", "bigint"}}}},
		{Type: storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"BTREE", "HASH"}}}},
		{Type: storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4", "UTF8"}}}},
		{Type: storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"utf8mb4_0900_ai_ci"}}}},
		{Type: storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_StringArrayPayload{StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"rand", "uuid", "sleep"}}}},
	}

	for _, rule := range rules {
		advisor.RunSQLReviewRuleTest(t, rule, storepb.Engine_MYSQL, false /* record */)
	}
}

func checkTestRowLimitRule(t *testing.T, prober *advisor.ProbeRecorder, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, statement string) []*storepb.Advice {
	t.Helper()
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{{
		Type:    ruleType,
		Level:   level,
		Engine:  storepb.Engine_MYSQL,
		Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
	}}, advisor.Context{
		DBType:          storepb.Engine_MYSQL,
		Prober:          prober,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}

func TestMySQLRowLimitAdvisorSeam(t *testing.T) {
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
			statement: "UPDATE td SET c = 1 WHERE c = 0 LIMIT 3;",
		},
		{
			name:        "affected row limit exceeded",
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			prober:      &advisor.ProbeRecorder{Rows: 1000},
			statement:   "UPDATE td SET c = 1;",
			wantContent: `"UPDATE td SET c = 1;" affected 1000 rows (estimated). The count exceeds 5.`,
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
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			prober:      &advisor.ProbeRecorder{EstimateErr: errors.New("Table 'test.tech_book' doesn't exist")},
			statement:   "UPDATE tech_book SET name = 'xz' WHERE id = 1;",
			wantContent: `failed to get row count for "UPDATE tech_book SET name = 'xz' WHERE id = 1;": Table 'test.tech_book' doesn't exist`,
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
}

func TestMySQLRowLimitAdvisorsWarnAboutStatementsBeyondExplainLimit(t *testing.T) {
	t.Run("affected row limit", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{Rows: 1000}
		statement := strings.Repeat("UPDATE td SET c = 1;\n", common.MaximumLintExplainSize+2)

		adviceList := checkTestRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, statement)
		require.Len(t, adviceList, common.MaximumLintExplainSize+1)
		for _, advice := range adviceList[:common.MaximumLintExplainSize] {
			require.Equal(t, storepb.Advice_ERROR, advice.Status)
			require.Contains(t, advice.Content, "affected 1000 rows")
		}
		warning := adviceList[common.MaximumLintExplainSize]
		require.Equal(t, storepb.Advice_WARNING, warning.Status)
		require.Equal(t, code.StatementAffectedRowExceedsLimit.Int32(), warning.Code)
		require.Equal(t, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT.String(), warning.Title)
		require.Equal(t, "Only the first 10 statements were estimated; 2 more were not checked against the row limit.", warning.Content)
		require.Equal(t, int32(common.MaximumLintExplainSize+1), warning.StartPosition.GetLine())
		require.Len(t, prober.Estimates, common.MaximumLintExplainSize)
	})

	t.Run("insert row limit keeps counting VALUES rows", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{Rows: 100}
		statement := strings.Repeat("INSERT INTO t2 SELECT * FROM t WHERE d = 3;\n", common.MaximumLintExplainSize+1) +
			"INSERT INTO t2 (id) VALUES (1), (2), (3), (4), (5), (6);"

		adviceList := checkTestRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_WARNING, statement)
		require.Len(t, adviceList, common.MaximumLintExplainSize+2)
		require.Equal(t, `"INSERT INTO t2 (id) VALUES (1), (2), (3), (4), (5), (6);" inserts 6 rows. The count exceeds 5.`, adviceList[common.MaximumLintExplainSize].Content)
		warning := adviceList[common.MaximumLintExplainSize+1]
		require.Equal(t, storepb.Advice_WARNING, warning.Status)
		require.Equal(t, code.InsertTooManyRows.Int32(), warning.Code)
		require.Equal(t, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT.String(), warning.Title)
		require.Equal(t, "Only the first 10 statements were estimated; 1 more were not checked against the row limit.", warning.Content)
		require.Equal(t, int32(common.MaximumLintExplainSize+1), warning.StartPosition.GetLine())
		require.Len(t, prober.Estimates, common.MaximumLintExplainSize)
	})
}

func TestMySQLBuiltinWalkThroughCheckTableExists(t *testing.T) {
	sm := sheet.NewManager()
	finalMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "user",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_MYSQL, true)

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "CREATE TABLE user(id INT);", []*storepb.SQLReviewRule{
		{Type: storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_MYSQL},
	}, advisor.Context{
		DBType:        storepb.Engine_MYSQL,
		FinalMetadata: finalMetadata,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.TableExists.Int32(), adviceList[0].Code)
	require.Equal(t, "Table `user` already exists", adviceList[0].Content)
}

func TestMySQLDMLDryRunAdvisor(t *testing.T) {
	const stmt = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
	rules := []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_MYSQL,
	}}
	sm := sheet.NewManager()

	t.Run("failed EXPLAIN is reported", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{ProbeErr: errors.New("Table 'test.tech_book' doesn't exist")}
		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, stmt, rules, advisor.Context{
			DBType:          storepb.Engine_MYSQL,
			Prober:          prober,
			NoAppendBuiltin: true,
		})
		require.NoError(t, err)
		require.Len(t, adviceList, 1)
		require.Equal(t, code.StatementDMLDryRunFailed.Int32(), adviceList[0].Code)
		require.Contains(t, adviceList[0].Content, "dry runs failed")
	})

	t.Run("batch mixing DDL is skipped", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{}
		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "CREATE TABLE t2 (id INT);\n"+stmt, rules, advisor.Context{
			DBType:          storepb.Engine_MYSQL,
			Prober:          prober,
			NoAppendBuiltin: true,
		})
		require.NoError(t, err)
		require.Empty(t, adviceList)
		require.Empty(t, prober.Probes)
	})

	t.Run("only INSERT, UPDATE and DELETE are probed", func(t *testing.T) {
		prober := &advisor.ProbeRecorder{}
		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "SELECT 1;\nINSERT INTO tech_book VALUES (1, 'a');\n"+stmt+"\nDELETE FROM tech_book;", rules, advisor.Context{
			DBType:          storepb.Engine_MYSQL,
			Prober:          prober,
			NoAppendBuiltin: true,
		})
		require.NoError(t, err)
		require.Empty(t, adviceList)
		require.Equal(t, []advisor.ProbeCall{
			{Statement: "EXPLAIN INSERT INTO tech_book VALUES (1, 'a');"},
			{Statement: "EXPLAIN " + stmt},
			{Statement: "EXPLAIN DELETE FROM tech_book;"},
		}, prober.Probes)
	})
}
