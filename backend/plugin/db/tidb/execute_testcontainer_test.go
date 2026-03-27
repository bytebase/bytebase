package tidb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

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
	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	defer container.Close(ctx)

	tidbDriver := openTestDriver(ctx, t, container)
	defer func() {
		require.NoError(t, tidbDriver.Close(ctx))
	}()

	_, err := tidbDriver.Execute(ctx, `
		CREATE DATABASE IF NOT EXISTS test;
		USE test;
		DROP TABLE IF EXISTS test.execute_create_index_in_transaction;
		CREATE TABLE test.execute_create_index_in_transaction (id INT);
		BEGIN;
		CREATE INDEX idx_execute_create_index_in_transaction ON test.execute_create_index_in_transaction(id);
		COMMIT;
	`, db.ExecuteOptions{})
	require.NoError(t, err)

	var count int
	query := `
		SELECT COUNT(*)
		FROM information_schema.tidb_indexes
		WHERE table_schema = 'test'
			AND table_name = 'execute_create_index_in_transaction'
			AND key_name = 'idx_execute_create_index_in_transaction'
	`
	err = tidbDriver.db.QueryRowContext(ctx, query).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestExecutePreparedStatementFlowWithCreateIndexString(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestTiDBContainer(ctx, t)
	defer container.Close(ctx)

	tidbDriver := openTestDriver(ctx, t, container)
	defer func() {
		require.NoError(t, tidbDriver.Close(ctx))
	}()

	statement := `
		CREATE DATABASE IF NOT EXISTS test;
		USE test;
		DROP TABLE IF EXISTS test.prepare_statement_flow;
		CREATE TABLE test.prepare_statement_flow (id INT);
		SET @sql := 'CREATE INDEX idx_prepare_statement_flow ON test.prepare_statement_flow(id)';
		PREPARE stmt FROM @sql;
		EXECUTE stmt;
		DEALLOCATE PREPARE stmt;
	`
	_, err := tidbDriver.Execute(ctx, statement, db.ExecuteOptions{})
	require.NoError(t, err)

	var count int
	query := `
		SELECT COUNT(*)
		FROM information_schema.tidb_indexes
		WHERE table_schema = 'test'
			AND table_name = 'prepare_statement_flow'
			AND key_name = 'idx_prepare_statement_flow'
	`
	err = tidbDriver.db.QueryRowContext(ctx, query).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
