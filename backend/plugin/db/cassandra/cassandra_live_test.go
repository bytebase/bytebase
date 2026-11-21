package cassandra

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestCassandraDriver exercises the adapter against a live cluster: ping, the
// instance and keyspace sync, a script with its task-run log, the editor's
// query path and its SELECT-only rule, truncation, the no-op dump, and the
// keyspace that goes away.
func TestCassandraDriver(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedCassandraContainer(t)

	open := func(keyspace string) *Driver {
		t.Helper()
		driver, err := newDriver().Open(ctx, storepb.Engine_CASSANDRA, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:     container.GetHost(),
				Port:     container.GetPort(),
				Username: container.GetUsername(),
			},
			Password: container.GetPassword(),
			ConnectionContext: db.ConnectionContext{
				DatabaseName: keyspace,
			},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = driver.Close(ctx) })
		cassandra, ok := driver.(*Driver)
		require.True(t, ok, "the registered driver is the Cassandra adapter")
		return cassandra
	}

	// A cluster handle with no keyspace, which is where a keyspace is created
	// and dropped.
	unbound := open("")
	require.NoError(t, unbound.Ping(ctx))

	const keyspace = "bb_wave0_test"
	_, err := unbound.Execute(ctx, "CREATE KEYSPACE IF NOT EXISTS bb_wave0_test WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}", db.ExecuteOptions{})
	require.NoError(t, err)

	database := open(keyspace)

	t.Run("sync instance", func(t *testing.T) {
		instance, err := database.SyncInstance(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, instance.Version)
		require.Contains(t, instance.Version, ".")
		var names []string
		for _, database := range instance.Databases {
			names = append(names, database.Name)
		}
		require.Contains(t, names, keyspace)
		require.NotContains(t, names, "system")
		require.NotContains(t, names, "system_schema")
	})

	t.Run("execute reports the script's steps", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		}}
		affected, err := database.Execute(ctx, `
			CREATE TABLE IF NOT EXISTS t (
				id int PRIMARY KEY,
				name text,
				score float,
				at timestamp
			);
			INSERT INTO t (id, name, score, at) VALUES (1, 'a', 1.5, '2026-11-18T07:30:00Z');
			INSERT INTO t (id, name, score, at) VALUES (2, 'b', 2.5, '2026-11-18T08:30:00Z');
		`, opts)
		require.NoError(t, err)
		require.Equal(t, int64(0), affected, "Cassandra reports no affected rows")

		var types []storepb.TaskRunLog_Type
		for _, log := range logs {
			types = append(types, log.Type)
		}
		require.Equal(t, []storepb.TaskRunLog_Type{
			storepb.TaskRunLog_COMMAND_EXECUTE, storepb.TaskRunLog_COMMAND_RESPONSE,
			storepb.TaskRunLog_COMMAND_EXECUTE, storepb.TaskRunLog_COMMAND_RESPONSE,
			storepb.TaskRunLog_COMMAND_EXECUTE, storepb.TaskRunLog_COMMAND_RESPONSE,
		}, types)
		commandRange := logs[0].GetCommandExecute().GetRange()
		require.NotNil(t, commandRange, "a task run log carries the command's range in the script")
		require.GreaterOrEqual(t, commandRange.GetStart(), int32(0))
		require.Greater(t, commandRange.GetEnd(), commandRange.GetStart())
	})

	t.Run("sync keyspace", func(t *testing.T) {
		schema, err := database.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Len(t, schema.Schemas, 1)
		require.Len(t, schema.Schemas[0].Tables, 1)
		table := schema.Schemas[0].Tables[0]
		require.Equal(t, "t", table.Name)
		var columns []string
		for _, column := range table.Columns {
			columns = append(columns, column.Name)
		}
		// The primary key first, then the regular columns alphabetically, which is
		// also the order masking relies on.
		require.Equal(t, []string{"id", "at", "name", "score"}, columns)
		require.Equal(t, "int", table.Columns[0].Type)
	})

	t.Run("query through an ssh tunnel", func(t *testing.T) {
		sshServer := testcontainer.NewSSHServer(t)
		driver, err := newDriver().Open(ctx, storepb.Engine_CASSANDRA, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:          container.GetHost(),
				Port:          container.GetPort(),
				SshHost:       sshServer.GetHost(),
				SshPort:       sshServer.GetPort(),
				SshUser:       sshServer.GetUser(),
				SshPrivateKey: sshServer.GetPrivateKey(),
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: keyspace},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = driver.Close(ctx) })

		require.NoError(t, driver.Ping(ctx))
		instance, err := driver.SyncInstance(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, instance.Version)
		results, err := driver.QueryConn(ctx, "SELECT id FROM t WHERE id = 1;", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, int64(1), results[0].RowsCount)
	})

	t.Run("query", func(t *testing.T) {
		results, err := database.QueryConn(ctx, "SELECT id, name, score, at FROM t WHERE id = 1;", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		result := results[0]
		require.Equal(t, "SELECT id, name, score, at FROM t WHERE id = 1", result.Statement)
		require.Equal(t, []string{"id", "name", "score", "at"}, result.ColumnNames)
		// gocql names a text column varchar, and Bytebase showed that name.
		require.Equal(t, []string{"int", "varchar", "float", "timestamp"}, result.ColumnTypeNames)
		require.Equal(t, int64(1), result.RowsCount)
		require.Empty(t, result.Error)
		require.Equal(t, []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: "a"}},
			{Kind: &v1pb.RowValue_FloatValue{FloatValue: 1.5}},
			{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(time.Date(2026, 11, 18, 7, 30, 0, 0, time.UTC)),
				Accuracy:        3,
			}}},
		}, result.Rows[0].Values)
	})

	t.Run("query limit", func(t *testing.T) {
		results, err := database.QueryConn(ctx, "SELECT id FROM t;", db.QueryContext{Limit: 1})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, int64(1), results[0].RowsCount)
	})

	t.Run("query result size cap", func(t *testing.T) {
		results, err := database.QueryConn(ctx, "SELECT id, name, score, at FROM t;", db.QueryContext{MaximumSQLResultSize: 1})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, util.FormatMaximumSQLResultSizeMessage(1), results[0].Error)
	})

	t.Run("query path keeps the inherited SELECT-only rule", func(t *testing.T) {
		// Cassandra registers no editor validator, so the rule the driver has
		// always applied is a no-op and a write goes through the query path.
		results, err := database.QueryConn(ctx, "INSERT INTO t (id, name) VALUES (3, 'c');", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Empty(t, results[0].Error)
		require.Equal(t, int64(0), results[0].RowsCount)
	})

	t.Run("dump writes nothing", func(t *testing.T) {
		schema, err := database.SyncDBSchema(ctx)
		require.NoError(t, err)
		var out bytes.Buffer
		require.NoError(t, database.Dump(ctx, &out, schema))
		require.Empty(t, out.String())
	})

	t.Run("a dropped keyspace is not an empty schema", func(t *testing.T) {
		_, err := unbound.Execute(ctx, "CREATE KEYSPACE IF NOT EXISTS bb_wave0_dropped WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}", db.ExecuteOptions{})
		require.NoError(t, err)
		dropped := open("bb_wave0_dropped")
		_, err = dropped.SyncDBSchema(ctx)
		require.NoError(t, err)

		_, err = unbound.Execute(ctx, "DROP KEYSPACE bb_wave0_dropped", db.ExecuteOptions{})
		require.NoError(t, err)
		_, err = dropped.SyncDBSchema(ctx)
		require.True(t, errors.Is(err, omnidriver.ErrNotFound), "want driver.ErrNotFound, got %v", err)
	})

	t.Run("a script that fails reports the failed statement", func(t *testing.T) {
		_, err := database.Execute(ctx, "INSERT INTO missing (id) VALUES (1);\nINSERT INTO t (id) VALUES (9);", db.ExecuteOptions{})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "failed to execute"), "want the driver's wrapping, got %v", err)
	})
}
