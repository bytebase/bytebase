package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	// testMySQLAdvisorExplainJSON is returned for `EXPLAIN FORMAT=JSON` queries.
	testMySQLAdvisorExplainJSON string

	// testMySQLAdvisorExplainErr, when set, fails every query, standing in for a
	// statement the server refuses to EXPLAIN.
	testMySQLAdvisorExplainErr error

	// testMySQLAdvisorSetErr, when set, fails every SET statement.
	testMySQLAdvisorSetErr error

	// testMySQLAdvisorQueries records every statement the fake driver is asked to
	// run, so a rule that sent raw DML instead of EXPLAIN is caught rather than
	// silently succeeding. testMySQLAdvisorQueryConns records the connection that ran each.
	testMySQLAdvisorQueries     []string
	testMySQLAdvisorQueryConns  []int
	testMySQLAdvisorConnections int
)

func init() {
	sql.Register("test_mysql_advisor_explain", testMySQLAdvisorExplainDriver{})
}

type testMySQLAdvisorExplainDriver struct{}

func (testMySQLAdvisorExplainDriver) Open(string) (driver.Conn, error) {
	testMySQLAdvisorConnections++
	return &testMySQLAdvisorExplainConn{id: testMySQLAdvisorConnections}, nil
}

type testMySQLAdvisorExplainConn struct {
	id int
}

func (*testMySQLAdvisorExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*testMySQLAdvisorExplainConn) Close() error {
	return nil
}

func (*testMySQLAdvisorExplainConn) Begin() (driver.Tx, error) {
	return testMySQLAdvisorExplainTx{}, nil
}

func (c *testMySQLAdvisorExplainConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	testMySQLAdvisorQueries = append(testMySQLAdvisorQueries, query)
	testMySQLAdvisorQueryConns = append(testMySQLAdvisorQueryConns, c.id)
	if !strings.HasPrefix(query, "SET ") {
		return nil, errors.Errorf("unexpected exec %q", query)
	}
	if testMySQLAdvisorSetErr != nil {
		return nil, testMySQLAdvisorSetErr
	}
	return driver.ResultNoRows, nil
}

func (c *testMySQLAdvisorExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	testMySQLAdvisorQueries = append(testMySQLAdvisorQueries, query)
	testMySQLAdvisorQueryConns = append(testMySQLAdvisorQueryConns, c.id)
	if testMySQLAdvisorExplainErr != nil {
		return nil, testMySQLAdvisorExplainErr
	}
	if strings.HasPrefix(query, "EXPLAIN FORMAT=JSON ") {
		return &testMySQLAdvisorExplainResultRows{columns: []string{"EXPLAIN"}, rows: [][]driver.Value{{testMySQLAdvisorExplainJSON}}}, nil
	}
	return &testMySQLAdvisorExplainResultRows{columns: []string{"id", "select_type", "table", "type", "rows", "filtered"}}, nil
}

type testMySQLAdvisorExplainTx struct{}

func (testMySQLAdvisorExplainTx) Commit() error {
	return nil
}

func (testMySQLAdvisorExplainTx) Rollback() error {
	return nil
}

type testMySQLAdvisorExplainResultRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *testMySQLAdvisorExplainResultRows) Columns() []string {
	return r.columns
}

func (*testMySQLAdvisorExplainResultRows) Close() error {
	return nil
}

func (r *testMySQLAdvisorExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

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
		{Type: storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 2560}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, Level: storepb.SQLReviewRule_WARNING, Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 20}}},
		{Type: storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, Level: storepb.SQLReviewRule_WARNING},
		{Type: storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, Level: storepb.SQLReviewRule_WARNING},
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

// openTestMySQLAdvisorDB resets the fake server and opens a pool that keeps no idle
// connections, so statements that do not share a pinned connection land on different ones.
func openTestMySQLAdvisorDB(t *testing.T, plan string) *sql.DB {
	t.Helper()
	testMySQLAdvisorExplainJSON = plan
	testMySQLAdvisorExplainErr = nil
	testMySQLAdvisorSetErr = nil
	testMySQLAdvisorQueries = nil
	testMySQLAdvisorQueryConns = nil
	db, err := sql.Open("test_mysql_advisor_explain", "")
	require.NoError(t, err)
	db.SetMaxIdleConns(0)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func checkTestRowLimitRule(t *testing.T, db *sql.DB, engine storepb.Engine, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, statement string) []*storepb.Advice {
	t.Helper()
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{
		{
			Type:    ruleType,
			Level:   level,
			Engine:  engine,
			Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
		},
	}, advisor.Context{
		DBType:          engine,
		Driver:          db,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}

const (
	// The flagged UPDATE target is scanned first; the join filter on s comes after it.
	testMySQLUpdateTargetFirstPlan = `{"query_block":{"select_id":1,"nested_loop":[
		{"table":{"update":true,"table_name":"big","access_type":"ALL","rows_examined_per_scan":11330,"rows_produced_per_join":11330,"filtered":"100.00"}},
		{"table":{"table_name":"s","access_type":"eq_ref","rows_examined_per_scan":1,"rows_produced_per_join":1133,"filtered":"10.00"}}
	]}}`
	// MariaDB does not flag a multi-table DELETE target.
	testMariaDBDeleteJoinPlan = `{"query_block":{"select_id":1,"nested_loop":[
		{"table":{"table_name":"s","access_type":"ALL","loops":1,"rows":100,"filtered":10}},
		{"table":{"table_name":"big","access_type":"ref","loops":10,"rows":100,"filtered":100}}
	]}}`
	// The plan of the SELECT that a single-table UPDATE of td is explained as.
	testMySQLSingleTableUpdatePlan = `{"query_block":{"select_id":1,"table":{"table_name":"td","access_type":"ALL","rows_examined_per_scan":1000,"rows_produced_per_join":1000,"filtered":"100.00"}}}`
	testMySQLInsertSelectPlan      = `{"query_block":{"select_id":1,
		"table":{"insert":true,"table_name":"t2","access_type":"ALL"},
		"insert_from":{"table":{"table_name":"t","access_type":"ALL","rows_examined_per_scan":1000,"rows_produced_per_join":100,"filtered":"10.00"}}
	}}`
	testMySQLInsertTablePlan = `{"query_block":{"select_id":1,
		"table":{"insert":true,"table_name":"t2","access_type":"ALL"},
		"insert_from":{"table":{"table_name":"t","access_type":"ALL","rows_examined_per_scan":1000,"rows_produced_per_join":1000,"filtered":"100.00"}}
	}}`
)

func TestMySQLRowLimitAdvisorsEstimateFromJSONPlan(t *testing.T) {
	for _, tc := range []struct {
		name      string
		engine    storepb.Engine
		ruleType  storepb.SQLReviewRule_Type
		plan      string
		setErr    error
		statement string
		// explained is the statement the rule explains, when it is not the statement itself.
		explained string
		// wantContent is empty when the statement stays within the limit of 5.
		wantContent string
		wantCode    code.Code
		// fallback is set when the JSON plan has no estimate to read, so the rule also runs the tabular EXPLAIN.
		fallback bool
	}{
		{
			name:        "update bounded by the join after the target",
			engine:      storepb.Engine_MYSQL,
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:        testMySQLUpdateTargetFirstPlan,
			statement:   "UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;",
			wantContent: `"UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;" affected 1133 rows (estimated). The count exceeds 5.`,
			wantCode:    code.StatementAffectedRowExceedsLimit,
		},
		{
			name:        "MariaDB multi-table delete without explain_json_format_version",
			engine:      storepb.Engine_MARIADB,
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:        testMariaDBDeleteJoinPlan,
			setErr:      &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"},
			statement:   "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			wantContent: `"DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;" affected 1000 rows (estimated). The count exceeds 5.`,
			wantCode:    code.StatementAffectedRowExceedsLimit,
		},
		{
			name:      "update capped by LIMIT",
			engine:    storepb.Engine_MYSQL,
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:      testMySQLSingleTableUpdatePlan,
			statement: "UPDATE td SET c = 1 WHERE c = 0 LIMIT 3;",
			explained: "SELECT 1 FROM td WHERE c = 0 LIMIT 3",
		},
		{
			name:        "target the plan does not name",
			engine:      storepb.Engine_MYSQL,
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:        `{"query_block":{"select_id":1,"table":{"table_name":"other","access_type":"ALL","rows_examined_per_scan":1000,"filtered":"100.00"}}}`,
			statement:   "UPDATE td SET c = 1;",
			explained:   "SELECT 1 FROM td",
			wantContent: `"UPDATE td SET c = 1;" affected 1000 rows (estimated). The count exceeds 5.`,
			wantCode:    code.StatementAffectedRowExceedsLimit,
		},
		{
			name:        "plan without an estimate falls back to the tabular plan",
			engine:      storepb.Engine_MYSQL,
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			plan:        `{"query_block":{"select_id":1,"table":{"insert":true,"select_id":1,"table_name":"t2","access_type":"ALL"}}}`,
			statement:   "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;",
			wantContent: `failed to get row count for "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;": table "t2" in the plan has no row estimate`,
			wantCode:    code.Internal,
			fallback:    true,
		},
		{
			name:        "insert select reads the source plan",
			engine:      storepb.Engine_MYSQL,
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			plan:        testMySQLInsertSelectPlan,
			statement:   "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			wantContent: `"INSERT INTO t2 SELECT * FROM t WHERE d = 3;" inserts 100 rows. The count exceeds 5.`,
			wantCode:    code.InsertTooManyRows,
		},
		{
			name:        "insert table",
			engine:      storepb.Engine_MYSQL,
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			plan:        testMySQLInsertTablePlan,
			statement:   "INSERT INTO t2 TABLE t;",
			wantContent: `"INSERT INTO t2 TABLE t;" inserts 1000 rows. The count exceeds 5.`,
			wantCode:    code.InsertTooManyRows,
		},
		{
			name:      "insert select capped by LIMIT",
			engine:    storepb.Engine_MYSQL,
			ruleType:  storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			plan:      testMySQLInsertSelectPlan,
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3 LIMIT 3;",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openTestMySQLAdvisorDB(t, tc.plan)
			testMySQLAdvisorSetErr = tc.setErr

			adviceList := checkTestRowLimitRule(t, db, tc.engine, tc.ruleType, storepb.SQLReviewRule_WARNING, tc.statement)
			if tc.wantContent == "" {
				require.Empty(t, adviceList)
			} else {
				require.Len(t, adviceList, 1)
				require.Equal(t, tc.wantCode.Int32(), adviceList[0].Code)
				require.Equal(t, tc.wantContent, adviceList[0].Content)
			}

			explained := tc.explained
			if explained == "" {
				explained = tc.statement
			}
			// The rule sets the JSON format version on the connection that runs the JSON EXPLAIN and
			// resets it where the server has the variable.
			wantQueries := []string{"SET SESSION explain_json_format_version = 1", "EXPLAIN FORMAT=JSON " + explained}
			if tc.setErr == nil {
				wantQueries = append(wantQueries, "SET SESSION explain_json_format_version = DEFAULT")
			}
			jsonQueries := len(wantQueries)
			if tc.fallback {
				wantQueries = append(wantQueries, "EXPLAIN FORMAT=TRADITIONAL "+explained)
			}
			require.Equal(t, wantQueries, testMySQLAdvisorQueries)
			for _, conn := range testMySQLAdvisorQueryConns[:jsonQueries] {
				require.Equal(t, testMySQLAdvisorQueryConns[0], conn)
			}
		})
	}
}

func TestMySQLRowLimitAdvisorsWarnAboutStatementsBeyondExplainLimit(t *testing.T) {
	t.Run("affected row limit", func(t *testing.T) {
		db := openTestMySQLAdvisorDB(t, testMySQLSingleTableUpdatePlan)
		statement := strings.Repeat("UPDATE td SET c = 1;\n", common.MaximumLintExplainSize+2)

		adviceList := checkTestRowLimitRule(t, db, storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, statement)
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
		require.Equal(t, common.MaximumLintExplainSize, countTestExplainQueries())
	})

	t.Run("insert row limit keeps counting VALUES rows", func(t *testing.T) {
		db := openTestMySQLAdvisorDB(t, testMySQLInsertSelectPlan)
		statement := strings.Repeat("INSERT INTO t2 SELECT * FROM t WHERE d = 3;\n", common.MaximumLintExplainSize+1) +
			"INSERT INTO t2 (id) VALUES (1), (2), (3), (4), (5), (6);"

		adviceList := checkTestRowLimitRule(t, db, storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_WARNING, statement)
		require.Len(t, adviceList, common.MaximumLintExplainSize+2)
		require.Equal(t, `"INSERT INTO t2 (id) VALUES (1), (2), (3), (4), (5), (6);" inserts 6 rows. The count exceeds 5.`, adviceList[common.MaximumLintExplainSize].Content)
		warning := adviceList[common.MaximumLintExplainSize+1]
		require.Equal(t, storepb.Advice_WARNING, warning.Status)
		require.Equal(t, code.InsertTooManyRows.Int32(), warning.Code)
		require.Equal(t, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT.String(), warning.Title)
		require.Equal(t, "Only the first 10 statements were estimated; 1 more were not checked against the row limit.", warning.Content)
		require.Equal(t, int32(common.MaximumLintExplainSize+1), warning.StartPosition.GetLine())
		require.Equal(t, common.MaximumLintExplainSize, countTestExplainQueries())
	})
}

func countTestExplainQueries() int {
	count := 0
	for _, query := range testMySQLAdvisorQueries {
		if strings.HasPrefix(query, "EXPLAIN ") {
			count++
		}
	}
	return count
}

func TestMariaDBPriorBackupCheckAdvisor(t *testing.T) {
	sm := sheet.NewManager()

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "CREATE TABLE t(id INT);\nUPDATE test SET c1 = 1 WHERE b1 = 1;", nil, advisor.Context{
		DBType:            storepb.Engine_MARIADB,
		DBSchema:          advisor.MockMySQLDatabase,
		EnablePriorBackup: true,
		InstanceID:        "instance",
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
			return []string{"bbdataarchive"}, nil
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, adviceList)
	require.Equal(t, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(), adviceList[0].Title)
	require.Contains(t, adviceList[0].Content, "mixed DDL and DML")
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

// TestMySQLDMLDryRunAdvisor covers STATEMENT_DML_DRY_RUN, which the yaml harness
// cannot: RunSQLReviewRuleTest builds its context with Driver: nil and this rule
// does nothing without one, which is why test/statement_dml_dry_run.yaml carries
// statements and no `want:` at all.
func TestMySQLDMLDryRunAdvisor(t *testing.T) {
	const stmt = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
	rules := []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_MYSQL,
	}}
	sm := sheet.NewManager()

	db, err := sql.Open("test_mysql_advisor_explain", "")
	require.NoError(t, err)
	defer db.Close()

	// Sequential: the second case arms a package-level failure hook.
	t.Run("a statement whose EXPLAIN succeeds raises nothing, and the DML never runs", func(t *testing.T) {
		testMySQLAdvisorQueries = nil
		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, stmt, rules, advisor.Context{
			DBType:          storepb.Engine_MYSQL,
			Driver:          db,
			NoAppendBuiltin: true,
		})
		require.NoError(t, err)
		require.Empty(t, adviceList)

		// The driver is the only way out of the rule, so assert nothing left it but
		// an EXPLAIN: a dry run that sent the raw UPDATE would pass otherwise.
		require.NotEmpty(t, testMySQLAdvisorQueries, "the rule must actually reach the driver")
		for _, q := range testMySQLAdvisorQueries {
			require.True(t, strings.HasPrefix(q, "EXPLAIN "),
				"the dry run must only ever EXPLAIN, got %q", q)
		}
	})

	t.Run("a statement whose EXPLAIN fails is reported", func(t *testing.T) {
		testMySQLAdvisorExplainErr = errors.New("Table 'test.tech_book' doesn't exist")
		defer func() { testMySQLAdvisorExplainErr = nil }()

		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, stmt, rules, advisor.Context{
			DBType:          storepb.Engine_MYSQL,
			Driver:          db,
			NoAppendBuiltin: true,
		})
		require.NoError(t, err)
		require.Len(t, adviceList, 1)
		require.Equal(t, code.StatementDMLDryRunFailed.Int32(), adviceList[0].Code)
		require.Contains(t, adviceList[0].Content, "dry runs failed")
	})
}
