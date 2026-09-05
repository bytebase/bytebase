package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"

	// Register the driver this test opens.
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestTransactionModeSwitch covers the execution half of the `-- txn-mode`
// directive: "on" wraps the script in one transaction, "off" runs it in
// auto-commit. It is the only coverage of that half -- executeInTransactionMode
// and executeInAutoCommitMode have no other test -- so it needs a replacement
// before it goes. Parsing is covered by plugin/parser/base.
//
// Not covered: DDL inside `txn-mode = on`, where MySQL and Oracle implicitly
// commit and defeat it, which is what the directive exists for.
func TestTransactionModeSwitch(t *testing.T) {
	ctx := context.Background()

	container := testcontainer.GetTestPgContainer(ctx, t)
	defer container.Close(ctx)

	driver, err := db.Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: "postgres",
			Database: "postgres",
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
		Password:          "root-password",
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	_, err = driver.Execute(ctx, "CREATE TABLE test_table (id INTEGER PRIMARY KEY, value VARCHAR(100));", db.ExecuteOptions{})
	require.NoError(t, err)

	// The third insert violates the primary key, so the script fails either way.
	// What differs is how much of it survives.
	const script = `
		INSERT INTO test_table (id, value) VALUES (1, 'test1');
		INSERT INTO test_table (id, value) VALUES (2, 'test2');
		INSERT INTO test_table (id, value) VALUES (1, 'duplicate');
	`

	// Sequential: both count rows across the one table, so each empties it after.
	t.Run("on rolls the whole script back", func(t *testing.T) {
		_, err := driver.Execute(ctx, "-- txn-mode = on\n"+script, db.ExecuteOptions{})
		require.Error(t, err, "the duplicate key must fail the script")
		require.Equal(t, 0, rowCount(ctx, t, driver), "txn-mode = on must roll back the statements that succeeded")
		emptyTable(ctx, t, driver)
	})

	t.Run("off keeps the statements that succeeded", func(t *testing.T) {
		_, err := driver.Execute(ctx, "-- txn-mode = off\n"+script, db.ExecuteOptions{})
		require.Error(t, err, "the duplicate key must fail the script")
		require.Equal(t, 2, rowCount(ctx, t, driver), "txn-mode = off must leave the two inserts that ran before the failure")
		emptyTable(ctx, t, driver)
	})
}

func rowCount(ctx context.Context, t *testing.T, driver db.Driver) int {
	t.Helper()
	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	results, err := driver.QueryConn(ctx, conn, "SELECT COUNT(*) FROM test_table", db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Len(t, results[0].Rows[0].Values, 1)

	count, ok := results[0].Rows[0].Values[0].Kind.(*v1pb.RowValue_Int64Value)
	require.True(t, ok, "COUNT(*) should come back as int64, got %T", results[0].Rows[0].Values[0].Kind)
	return int(count.Int64Value)
}

func emptyTable(ctx context.Context, t *testing.T, driver db.Driver) {
	t.Helper()
	_, err := driver.Execute(ctx, "DELETE FROM test_table", db.ExecuteOptions{})
	require.NoError(t, err)
}
