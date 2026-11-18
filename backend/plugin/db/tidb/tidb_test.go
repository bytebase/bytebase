package tidb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestBuildConfigMapsExtraConnectionParameters pins the connection parameters a
// data source carries reaching the contract's Params, where the omni driver
// appends them to the DSN beside multiStatements and maxAllowedPacket.
func TestBuildConfigMapsExtraConnectionParameters(t *testing.T) {
	config, sshClient, err := buildConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"readTimeout":  "30s",
				"writeTimeout": "45s",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName:  "test",
			EngineVersion: "8.0.11-TiDB-v8.5.5",
		},
	})
	require.NoError(t, err)
	require.Nil(t, sshClient)
	require.Equal(t, map[string]string{"readTimeout": "30s", "writeTimeout": "45s"}, config.Params)
	require.Equal(t, "test", config.Database)
	require.Equal(t, "secret", config.Password)
	require.Equal(t, "8.0.11-TiDB-v8.5.5", config.EngineVersion)
	// The application name keeps the /*app=bytebase*/ comment the old driver
	// prepended.
	require.Equal(t, "bytebase", config.ApplicationName)
}

// TestOpenRejectsAllowAllFiles keeps the refusal the old driver made before it
// built a DSN.
func TestOpenRejectsAllowAllFiles(t *testing.T) {
	driver := newDriver()
	_, err := driver.Open(context.Background(), storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "4000",
			Username: "root",
			ExtraConnectionParameters: map[string]string{
				"allowAllFiles": "true",
			},
		},
		Password: "secret",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "test",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

// TestOpenRequiresAUser pins the one thing the old DSN silently sent: the
// contract refuses a connection with no login.
func TestOpenRequiresAUser(t *testing.T) {
	driver := newDriver()
	_, err := driver.Open(context.Background(), storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host: "127.0.0.1",
			Port: "4000",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "test"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "user must be set")
}

// TestExecuteSplitsDelimiterScript proves the splitter the old driver used
// survives the cutover: a DELIMITER directive groups statements for the split
// without reaching the server.
func TestExecuteSplitsDelimiterScript(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	_, err := driver.Execute(ctx, `
		DELIMITER //
		CREATE TABLE delimiter_one (id INT)//
		CREATE TABLE delimiter_two (id INT)//
		DELIMITER ;
	`, db.ExecuteOptions{})
	require.NoError(t, err)

	for _, table := range []string{"delimiter_one", "delimiter_two"} {
		var count int
		err := container.GetDB().QueryRowContext(ctx, fmt.Sprintf(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'", database, table,
		)).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, table)
	}
}

// recordTaskRunLogs collects the task run log entries an Execute call writes.
func recordTaskRunLogs(t *testing.T) (db.ExecuteOptions, *[]*storepb.TaskRunLog) {
	t.Helper()
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	return opts, &logs
}

func logTypes(logs []*storepb.TaskRunLog) []string {
	types := make([]string, len(logs))
	for i, entry := range logs {
		types[i] = entry.Type.String()
	}
	return types
}

// TestExecuteHonoursTheTransactionDirective proves the product's directive still
// decides whether a script runs in one transaction, and that the task run log
// still records the transaction control entries and the byte range of each
// command.
func TestExecuteHonoursTheTransactionDirective(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		directive      string
		wantReverted   bool
		wantControl    []string
		failOn         string
		wantLogActions []string
	}{
		{
			name:         "transactional script rolls back",
			directive:    "",
			wantReverted: true,
			wantControl: []string{
				storepb.TaskRunLog_TransactionControl_BEGIN.String(),
				storepb.TaskRunLog_TransactionControl_ROLLBACK.String(),
			},
			failOn: "INSERT INTO missing_table VALUES (1);",
			wantLogActions: []string{
				storepb.TaskRunLog_TRANSACTION_CONTROL.String(),
				storepb.TaskRunLog_COMMAND_EXECUTE.String(),
				storepb.TaskRunLog_COMMAND_RESPONSE.String(),
				storepb.TaskRunLog_COMMAND_EXECUTE.String(),
				storepb.TaskRunLog_COMMAND_RESPONSE.String(),
				storepb.TaskRunLog_TRANSACTION_CONTROL.String(),
			},
		},
		{
			name:         "auto-commit script keeps the statements before the failure",
			directive:    "-- txn-mode = off\n",
			wantReverted: false,
			wantControl:  nil,
			failOn:       "INSERT INTO missing_table VALUES (1);",
			wantLogActions: []string{
				storepb.TaskRunLog_COMMAND_EXECUTE.String(),
				storepb.TaskRunLog_COMMAND_RESPONSE.String(),
				storepb.TaskRunLog_COMMAND_EXECUTE.String(),
				storepb.TaskRunLog_COMMAND_RESPONSE.String(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			container, database := testcontainer.NewTiDBDatabase(t)
			driver := openTestDriverOn(ctx, t, container, database)
			t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

			_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.txn_probe (id INT)", database))
			require.NoError(t, err)

			opts, logs := recordTaskRunLogs(t)
			script := tc.directive + fmt.Sprintf("INSERT INTO %s.txn_probe VALUES (1);\n%s\n", database, tc.failOn)
			_, err = driver.Execute(ctx, script, opts)
			require.Error(t, err)

			var count int
			require.NoError(t, container.GetDB().QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.txn_probe", database)).Scan(&count))
			if tc.wantReverted {
				require.Equal(t, 0, count, "a transactional script rolls back")
			} else {
				require.Equal(t, 1, count, "an auto-commit script keeps the statement before the failure")
			}

			var control []string
			for _, entry := range *logs {
				if entry.GetTransactionControl() != nil {
					control = append(control, entry.GetTransactionControl().GetType().String())
				}
			}
			require.Equal(t, tc.wantControl, control)
			require.Equal(t, tc.wantLogActions, logTypes(*logs))

			// The command's byte range covers the statement and its semicolon,
			// as Bytebase's splitter reported it. Directives are removed before
			// the script is split, so the range is relative to the cleaned script.
			_, cleaned := base.ParseTransactionConfig(script)
			var executed []string
			for _, entry := range *logs {
				if command := entry.GetCommandExecute(); command != nil {
					require.NotNil(t, command.GetRange())
					start, end := int(command.GetRange().GetStart()), int(command.GetRange().GetEnd())
					require.True(t, start >= 0 && end <= len(cleaned) && start < end, "range %d:%d", start, end)
					executed = append(executed, cleaned[start:end])
				}
			}
			require.Len(t, executed, 2)
			require.Equal(t, fmt.Sprintf("INSERT INTO %s.txn_probe VALUES (1);", database), strings.TrimSpace(executed[0]))
			require.Equal(t, tc.failOn, strings.TrimSpace(executed[1]))
		})
	}
}

func TestExecuteCreateIndexInTransaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)

	tidbDriver := openTestDriver(ctx, t, container)
	defer func() {
		require.NoError(t, tidbDriver.Close(ctx))
	}()

	_, err := tidbDriver.Execute(ctx, fmt.Sprintf(`
		USE %[1]s;
		CREATE TABLE %[1]s.execute_create_index_in_transaction (id INT);
		BEGIN;
		CREATE INDEX idx_execute_create_index_in_transaction ON %[1]s.execute_create_index_in_transaction(id);
		COMMIT;
	`, database), db.ExecuteOptions{})
	require.NoError(t, err)

	var count int
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.tidb_indexes
		WHERE table_schema = '%s'
			AND table_name = 'execute_create_index_in_transaction'
			AND key_name = 'idx_execute_create_index_in_transaction'
	`, database)
	err = container.GetDB().QueryRowContext(ctx, query).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestExecutePreparedStatementFlowWithCreateIndexString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)

	tidbDriver := openTestDriver(ctx, t, container)
	defer func() {
		require.NoError(t, tidbDriver.Close(ctx))
	}()

	statement := fmt.Sprintf(`
		USE %[1]s;
		CREATE TABLE %[1]s.prepare_statement_flow (id INT);
		SET @sql := 'CREATE INDEX idx_prepare_statement_flow ON %[1]s.prepare_statement_flow(id)';
		PREPARE stmt FROM @sql;
		EXECUTE stmt;
		DEALLOCATE PREPARE stmt;
	`, database)
	_, err := tidbDriver.Execute(ctx, statement, db.ExecuteOptions{})
	require.NoError(t, err)

	var count int
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.tidb_indexes
		WHERE table_schema = '%s'
			AND table_name = 'prepare_statement_flow'
			AND key_name = 'idx_prepare_statement_flow'
	`, database)
	err = container.GetDB().QueryRowContext(ctx, query).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// TestQueryConnExplainJSON pins the structured plan the query-plan viewer draws.
// TiDB spells it FORMAT=tidb_json and rejects a quoted format in the statement
// Bytebase validates, so what the driver builds has to parse here as well.
func TestQueryConnExplainJSON(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	defer func() { require.NoError(t, driver.Close(ctx)) }()

	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.plan_target (id INT PRIMARY KEY, note VARCHAR(64))", database))
	require.NoError(t, err)

	statement := fmt.Sprintf("SELECT note FROM %s.plan_target WHERE id > 100", database)
	explain := func(s string, option *v1pb.QueryOption, asExplain bool) *v1pb.QueryResult {
		results, err := driver.QueryConn(ctx, nil, s, db.QueryContext{
			Explain:              asExplain,
			Limit:                5000,
			MaximumSQLResultSize: 1 << 30,
			Option:               option,
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].GetError())
		return results[0]
	}
	// The root operator's id, which every node of a tidb_json plan carries.
	rootOperator := func(result *v1pb.QueryResult) string {
		t.Helper()
		require.Len(t, result.GetColumnNames(), 1)
		require.Len(t, result.GetRows(), 1)
		var plan []struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal([]byte(result.GetRows()[0].GetValues()[0].GetStringValue()), &plan))
		require.NotEmpty(t, plan)
		return plan[0].ID
	}

	jsonOption := &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON}
	plan := explain(statement, jsonOption, true)
	require.Equal(t, "EXPLAIN FORMAT=tidb_json "+statement, plan.GetStatement())
	require.NotEmpty(t, rootOperator(plan))
	// Known difference: unchanged. An explain request reports no typed plan,
	// because its rows are the plan the caller asked the driver to build.
	require.Nil(t, plan.GetQueryPlan())

	// The default plan stays the table TiDB has always returned.
	text := explain(statement, nil, true)
	require.Equal(t, "EXPLAIN "+statement, text.GetStatement())
	require.Greater(t, len(text.GetColumnNames()), 1)

	// A caller's own EXPLAIN is replanned in the format asked for rather than
	// wrapped in a second one.
	typed := explain("EXPLAIN FORMAT=brief "+statement, jsonOption, true)
	require.Equal(t, "EXPLAIN FORMAT=tidb_json "+statement, typed.GetStatement())
	require.NotEmpty(t, rootOperator(typed))

	// Run on that statement reports the plan it returned, so the viewer draws it
	// without explaining a second time.
	ran := explain("EXPLAIN FORMAT=tidb_json "+statement, nil, false)
	require.Equal(t, v1pb.QueryOption_JSON, ran.GetQueryPlan().GetFormat())
	require.False(t, ran.GetQueryPlan().GetExecuted())
	require.NotEmpty(t, rootOperator(ran))

	// TiDB has no session setting that changes what a plain EXPLAIN returns, so
	// its format is known to be text.
	ranText := explain("EXPLAIN "+statement, nil, false)
	require.Equal(t, v1pb.QueryOption_TEXT, ranText.GetQueryPlan().GetFormat())
}

// TestQueryConnRefusesAZeroResultCap states the open decision the adapter takes:
// the product reads MaximumSQLResultSize 0 as "no rows" and the contract reads
// it as "no cap", so the adapter refuses rather than read an unbounded result.
func TestQueryConnRefusesAZeroResultCap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	defer func() { require.NoError(t, driver.Close(ctx)) }()

	_, err := driver.QueryConn(ctx, nil, fmt.Sprintf("SELECT 1 FROM %s.zero_cap", database), db.QueryContext{})
	require.Error(t, err)
	require.Equal(t, "maximum SQL result size must be positive", err.Error())
}

// TestQueryConnKeepsOneSession proves the statements of one driver share a
// session, as they shared the caller's connection before: a session variable set
// by one call is visible to the next.
func TestQueryConnKeepsOneSession(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, _ := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	defer func() { require.NoError(t, driver.Close(ctx)) }()

	query := func(statement string) *v1pb.QueryResult {
		t.Helper()
		results, err := driver.QueryConn(ctx, nil, statement, db.QueryContext{MaximumSQLResultSize: 1 << 20})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].GetError())
		return results[0]
	}
	query("SET @bytebase_session_probe = 42")
	result := query("SELECT @bytebase_session_probe")
	require.Len(t, result.GetRows(), 1)
	require.Equal(t, int64(42), result.GetRows()[0].GetValues()[0].GetInt64Value())
}

// TestQueryConnDiscardsTheSessionAfterACanceledQuery pins the session lifetime
// on the path that ends a statement early: the client closes the connection
// when the statement's context ends, so the stream's next statement must be
// given a new session rather than the dead one.
func TestQueryConnDiscardsTheSessionAfterACanceledQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, _ := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	defer func() { require.NoError(t, driver.Close(ctx)) }()
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	_, err := driver.QueryConn(ctx, nil, "SELECT 1", queryContext)
	require.NoError(t, err)
	held := driver.session
	require.NotNil(t, held, "a statement of its own leaves the session in place")

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, _ = driver.QueryConn(canceledCtx, nil, "SELECT 1", queryContext)
	require.Nil(t, driver.session, "a statement whose context ended is not one to keep")
	require.NotNil(t, held)
}

// TestQueryConnReportsAParseErrorAsAKnownDifference records the error text the
// product now shows for invalid SQL: omni's parser message rather than
// Bytebase's parser adapter's "Syntax error at line ..." rendering.
func TestQueryConnReportsAParseErrorAsAKnownDifference(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, _ := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	defer func() { require.NoError(t, driver.Close(ctx)) }()

	_, err := driver.QueryConn(ctx, nil, "SELECT FROM WHERE", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.Error(t, err)
	// Before, through Bytebase's parser adapter:
	//   Syntax error at line 1:8: syntax error at or near "FROM"
	//   related text: SELECT FROM WHERE
	// After, omni's parser error as the driver reports it:
	//   syntax error at or near "FROM" (line 1, column 8)
	//   related text: SELECT FROM WHERE
	require.Contains(t, err.Error(), `syntax error at or near "FROM" (line 1, column 8)`)
	require.NotContains(t, err.Error(), "Syntax error at line")
}
