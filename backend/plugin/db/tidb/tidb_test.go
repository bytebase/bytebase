package tidb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

func openTestDriver(ctx context.Context, t *testing.T, container *testcontainer.Container) *Driver {
	t.Helper()

	driver := &Driver{}
	d, err := driver.Open(ctx, storepb.Engine_TIDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "root",
			Host:     container.GetHost(),
			Port:     container.GetPort(),
		},
		ConnectionContext: db.ConnectionContext{},
	})
	require.NoError(t, err)

	tidbDriver, ok := d.(*Driver)
	require.True(t, ok)
	return tidbDriver
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
