package tidb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetTiDBConnectionUsesExtraConnectionParameters(t *testing.T) {
	d := &Driver{}
	dsn, err := d.getTiDBConnection(db.ConnectionConfig{
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
			DatabaseName: "test",
		},
	})
	require.NoError(t, err)
	require.Contains(t, dsn, "readTimeout=30s")
	require.Contains(t, dsn, "writeTimeout=45s")
}

func TestGetTiDBConnectionRejectsAllowAllFiles(t *testing.T) {
	d := &Driver{}
	_, err := d.getTiDBConnection(db.ConnectionConfig{
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

func TestBuildExecuteCommandsNormalizesDelimiter(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n"

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.NotContains(t, commands[0].Text, "DELIMITER")
	require.Contains(t, commands[0].Text, "CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND")
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForTooManyCommands(t *testing.T) {
	var statement strings.Builder
	statement.WriteString("DELIMITER //\n")
	statement.WriteString("CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\n")
	statement.WriteString("DELIMITER ;\n")
	statement.WriteString("/*!50003 SET @OLD_SQL_MODE=@@SQL_MODE */;\n")
	for i := 0; i < common.MaximumCommands; i++ {
		statement.WriteString("SELECT 1;\n")
	}

	commands, err := buildExecuteCommands(statement.String())
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement.String(), commands[0].Text)
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForLargeSheet(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n" +
		strings.Repeat(" ", common.MaxSheetCheckSize)

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement, commands[0].Text)
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
	err = tidbDriver.db.QueryRowContext(ctx, query).Scan(&count)
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
	err = tidbDriver.db.QueryRowContext(ctx, query).Scan(&count)
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

	_, err := driver.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.plan_target (id INT PRIMARY KEY, note VARCHAR(64))", database))
	require.NoError(t, err)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	statement := fmt.Sprintf("SELECT note FROM %s.plan_target WHERE id > 100", database)
	explain := func(s string, option *v1pb.QueryOption, asExplain bool) *v1pb.QueryResult {
		results, err := driver.QueryConn(ctx, conn, s, db.QueryContext{
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
