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

// TestTransactionModeSwitch pins the execution half of the `-- txn-mode`
// directive: "on" wraps the script in one transaction, "off" runs the statements
// in auto-commit. Nothing else in the repo covers that. The parsing half is
// plugin/parser/base's TestCleanDirectives, but executeInTransactionMode and
// executeInAutoCommitMode have no other test, so without this one every driver
// could silently ignore the directive and the suite would stay green.
//
// It lives beside the driver rather than in backend/tests because it boots no
// Bytebase server and drives no workflow.
//
// One engine on purpose. Every driver implements the same two branches, and what
// this guards is that our branch is wired to something that really opens a
// transaction -- not how an engine behaves once it is in one. The earlier
// four-engine version asserted identical expectations four times over, for three
// extra containers.
//
// The case worth adding, if anyone wants that spend back: DDL inside
// `txn-mode = on`, where MySQL and Oracle implicitly commit and silently defeat
// it. That is the reason the directive exists (see common/engine.go), and no
// version of this test has ever covered it.
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

	// Sequential, and they share one table: each asserts a row count over the
	// whole table and then empties it for the next.
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

	switch v := results[0].Rows[0].Values[0].Kind.(type) {
	case *v1pb.RowValue_Int32Value:
		return int(v.Int32Value)
	case *v1pb.RowValue_Int64Value:
		return int(v.Int64Value)
	default:
		t.Fatalf("unexpected count value type: %T", v)
		return 0
	}
}

func emptyTable(ctx context.Context, t *testing.T, driver db.Driver) {
	t.Helper()
	_, err := driver.Execute(ctx, "DELETE FROM test_table", db.ExecuteOptions{})
	require.NoError(t, err)
}
