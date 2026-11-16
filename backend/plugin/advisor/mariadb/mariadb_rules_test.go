package mariadb

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

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	// testMariaDBAdvisorExplainJSON is returned for `EXPLAIN FORMAT=JSON` queries.
	testMariaDBAdvisorExplainJSON string

	// testMariaDBAdvisorExplainErr, when set, fails every query, standing in for a
	// statement the server refuses to EXPLAIN.
	testMariaDBAdvisorExplainErr error

	// testMariaDBAdvisorSetErr, when set, fails every SET statement.
	testMariaDBAdvisorSetErr error

	// testMariaDBAdvisorQueries records every statement the fake driver is asked to
	// run, so a rule that sent raw DML instead of EXPLAIN is caught rather than
	// silently succeeding. testMariaDBAdvisorQueryConns records the connection that ran each.
	testMariaDBAdvisorQueries     []string
	testMariaDBAdvisorQueryConns  []int
	testMariaDBAdvisorConnections int
)

func init() {
	sql.Register("test_mariadb_advisor_explain", testMariaDBAdvisorExplainDriver{})
}

type testMariaDBAdvisorExplainDriver struct{}

func (testMariaDBAdvisorExplainDriver) Open(string) (driver.Conn, error) {
	testMariaDBAdvisorConnections++
	return &testMariaDBAdvisorExplainConn{id: testMariaDBAdvisorConnections}, nil
}

type testMariaDBAdvisorExplainConn struct {
	id int
}

func (*testMariaDBAdvisorExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*testMariaDBAdvisorExplainConn) Close() error {
	return nil
}

func (*testMariaDBAdvisorExplainConn) Begin() (driver.Tx, error) {
	return testMariaDBAdvisorExplainTx{}, nil
}

func (c *testMariaDBAdvisorExplainConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	testMariaDBAdvisorQueries = append(testMariaDBAdvisorQueries, query)
	testMariaDBAdvisorQueryConns = append(testMariaDBAdvisorQueryConns, c.id)
	if !strings.HasPrefix(query, "SET ") {
		return nil, errors.Errorf("unexpected exec %q", query)
	}
	if testMariaDBAdvisorSetErr != nil {
		return nil, testMariaDBAdvisorSetErr
	}
	return driver.ResultNoRows, nil
}

func (c *testMariaDBAdvisorExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	testMariaDBAdvisorQueries = append(testMariaDBAdvisorQueries, query)
	testMariaDBAdvisorQueryConns = append(testMariaDBAdvisorQueryConns, c.id)
	if testMariaDBAdvisorExplainErr != nil {
		return nil, testMariaDBAdvisorExplainErr
	}
	if strings.HasPrefix(query, "EXPLAIN FORMAT=JSON ") {
		return &testMariaDBAdvisorExplainResultRows{columns: []string{"EXPLAIN"}, rows: [][]driver.Value{{testMariaDBAdvisorExplainJSON}}}, nil
	}
	return &testMariaDBAdvisorExplainResultRows{columns: []string{"id", "select_type", "table", "type", "rows", "filtered"}}, nil
}

type testMariaDBAdvisorExplainTx struct{}

func (testMariaDBAdvisorExplainTx) Commit() error {
	return nil
}

func (testMariaDBAdvisorExplainTx) Rollback() error {
	return nil
}

type testMariaDBAdvisorExplainResultRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *testMariaDBAdvisorExplainResultRows) Columns() []string {
	return r.columns
}

func (*testMariaDBAdvisorExplainResultRows) Close() error {
	return nil
}

func (r *testMariaDBAdvisorExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

// openTestMariaDBAdvisorDB resets the fake server and opens a pool that keeps no idle
// connections, so statements that do not share a pinned connection land on different ones.
func openTestMariaDBAdvisorDB(t *testing.T, plan string) *sql.DB {
	t.Helper()
	testMariaDBAdvisorExplainJSON = plan
	testMariaDBAdvisorExplainErr = nil
	testMariaDBAdvisorSetErr = nil
	testMariaDBAdvisorQueries = nil
	testMariaDBAdvisorQueryConns = nil
	db, err := sql.Open("test_mariadb_advisor_explain", "")
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

const (
	// A real MariaDB 11 EXPLAIN FORMAT=JSON plan: it marks no multi-table DELETE
	// target, so the target is found by name.
	testMariaDBDeleteJoinPlan = `{"query_block":{"select_id":1,"nested_loop":[
		{"table":{"table_name":"s","access_type":"ALL","loops":1,"rows":100,"filtered":10}},
		{"table":{"table_name":"big","access_type":"ref","loops":10,"rows":100,"filtered":100}}
	]}}`
	testMariaDBInsertSelectPlan = `{"query_block":{"select_id":1,
		"table":{"insert":true,"table_name":"t2","access_type":"ALL"},
		"insert_from":{"table":{"table_name":"t","access_type":"ALL","rows":1000,"loops":1,"filtered":10}}
	}}`
)

// TestMariaDBRowLimitAdvisorsEstimateFromJSONPlan covers this package's own
// estimateAffectedRows: the driver's copy reads an omni MySQL node, so MariaDB
// composes the driver's tabular EXPLAIN with its own plan reader.
func TestMariaDBRowLimitAdvisorsEstimateFromJSONPlan(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ruleType  storepb.SQLReviewRule_Type
		plan      string
		setErr    error
		statement string
		// wantContent is empty when the statement stays within the limit of 5.
		wantContent string
		wantCode    code.Code
	}{
		{
			name:        "multi-table delete without explain_json_format_version",
			ruleType:    storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:        testMariaDBDeleteJoinPlan,
			setErr:      &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"},
			statement:   "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			wantContent: `"DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;" affected 1000 rows (estimated). The count exceeds 5.`,
			wantCode:    code.StatementAffectedRowExceedsLimit,
		},
		{
			name:        "insert select",
			ruleType:    storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			plan:        testMariaDBInsertSelectPlan,
			setErr:      &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"},
			statement:   "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			wantContent: `"INSERT INTO t2 SELECT * FROM t WHERE d = 3;" inserts 100 rows. The count exceeds 5.`,
			wantCode:    code.InsertTooManyRows,
		},
		{
			name:      "delete capped by LIMIT",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			plan:      testMariaDBDeleteJoinPlan,
			setErr:    &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"},
			statement: "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1 LIMIT 3;",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openTestMariaDBAdvisorDB(t, tc.plan)
			testMariaDBAdvisorSetErr = tc.setErr

			adviceList := checkTestRowLimitRule(t, db, storepb.Engine_MARIADB, tc.ruleType, storepb.SQLReviewRule_WARNING, tc.statement)
			if tc.wantContent == "" {
				require.Empty(t, adviceList)
			} else {
				require.Len(t, adviceList, 1)
				require.Equal(t, tc.wantCode.Int32(), adviceList[0].Code)
				require.Equal(t, tc.wantContent, adviceList[0].Content)
			}
		})
	}
}

// TestMariaDBRowLimitAdvisorsEstimateFromTabularFallback drives the half of
// estimateAffectedRows that reaches back into the shared driver: a plan with no
// estimate to read falls through to the driver's tabular EXPLAIN.
func TestMariaDBRowLimitAdvisorsEstimateFromTabularFallback(t *testing.T) {
	const statement = "INSERT INTO t2 SELECT id, ROW_NUMBER() OVER () FROM t;"
	db := openTestMariaDBAdvisorDB(t, `{"query_block":{"select_id":1,"table":{"insert":true,"select_id":1,"table_name":"t2","access_type":"ALL"}}}`)
	testMariaDBAdvisorSetErr = &mysql.MySQLError{Number: 1193, Message: "Unknown system variable 'explain_json_format_version'"}

	adviceList := checkTestRowLimitRule(t, db, storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_WARNING, statement)

	require.Len(t, adviceList, 1)
	require.Equal(t, code.Internal.Int32(), adviceList[0].Code)
	require.Contains(t, testMariaDBAdvisorQueries, "EXPLAIN FORMAT=TRADITIONAL "+statement)
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

// TestMariaDBPriorBackupCheckAdvisorReadsMariaDBOnlyStatements pins a change this
// package makes: the backup transform reads MariaDB-only DML, so the whole
// statement list is backed up as asked rather than drawing "Prior backup cannot
// handle MariaDB-specific syntax".
func TestMariaDBPriorBackupCheckAdvisorReadsMariaDBOnlyStatements(t *testing.T) {
	sm := sheet.NewManager()

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "DELETE FROM test WHERE b1 = 2 RETURNING b1;", nil, advisor.Context{
		DBType:            storepb.Engine_MARIADB,
		DBSchema:          advisor.MockMySQLDatabase,
		EnablePriorBackup: true,
		InstanceID:        "instance",
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
			return []string{"bbdataarchive"}, nil
		},
	})

	require.NoError(t, err)
	require.Empty(t, adviceList)
}

// TestMariaDBPriorBackupCheckAdvisorCountsSequenceDDL covers statements MySQL has
// no grammar for, which a DDL list written for MySQL has to name explicitly.
// Without them the script below passes a guard that promises to reject mixed DDL
// and DML, and the backup covers only the DELETE.
func TestMariaDBPriorBackupCheckAdvisorCountsSequenceDDL(t *testing.T) {
	for _, ddl := range []string{
		"CREATE SEQUENCE s START WITH 1",
		"ALTER SEQUENCE s RESTART",
		"DROP SEQUENCE s",
	} {
		sm := sheet.NewManager()
		adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, ddl+";\nDELETE FROM test WHERE b1 = 1;", nil, advisor.Context{
			DBType:            storepb.Engine_MARIADB,
			DBSchema:          advisor.MockMySQLDatabase,
			EnablePriorBackup: true,
			InstanceID:        "instance",
			ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
				return []string{"bbdataarchive"}, nil
			},
		})
		require.NoError(t, err, ddl)
		require.NotEmpty(t, adviceList, ddl)
		require.Equal(t, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(), adviceList[0].Title, ddl)
		require.Contains(t, adviceList[0].Content, "mixed DDL and DML", ddl)
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
