package hive

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestHiveDriver exercises the adapter against the pinned HiveServer2. The
// driver this replaced could not open a session — its SASL login was empty — and
// could not sync a table, and no test ran either path, so there is no old
// behaviour to compare against: each step here is the proof that it works.
func TestHiveDriver(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedHiveContainer(t)

	open := func(database string) *Driver {
		t.Helper()
		driver, err := newDriver().Open(ctx, storepb.Engine_HIVE, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:     container.GetHost(),
				Port:     container.GetPort(),
				Username: container.GetUsername(),
			},
			ConnectionContext: db.ConnectionContext{DatabaseName: database},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = driver.Close(ctx) })
		hive, ok := driver.(*Driver)
		require.True(t, ok, "the registered driver is the Hive adapter")
		return hive
	}

	const database = "bb_hive_test"
	// HiveServer2 refuses a session opened on a database that does not exist, so
	// the handle that prepares the fixture is opened on none. Both databases are
	// dropped first so a rerun on the same container starts from nothing.
	admin := open("")
	t.Run("open reaches the server", func(t *testing.T) {
		require.NoError(t, admin.Ping(ctx))
		require.Nil(t, admin.GetDB())
	})
	_, err := admin.Execute(ctx, "DROP DATABASE IF EXISTS "+database+" CASCADE;\nCREATE DATABASE "+database+";", db.ExecuteOptions{})
	require.NoError(t, err)

	session := open(database)
	t.Run("execute creates the fixture and reports its steps", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		}}
		_, err := session.Execute(ctx, strings.Join([]string{
			"CREATE TABLE " + database + ".orders (id BIGINT, amount DECIMAL(10,2), note STRING COMMENT 'a note') STORED AS PARQUET",
			"ALTER TABLE " + database + ".orders SET TBLPROPERTIES('EXTERNAL'='FALSE')",
			"INSERT INTO " + database + ".orders VALUES (1, 1.5, 'n1'), (2, 2.5, 'n2')",
			"CREATE TABLE " + database + ".events (id INT, body STRING) PARTITIONED BY (dt STRING) STORED AS PARQUET",
			"ALTER TABLE " + database + ".events SET TBLPROPERTIES('EXTERNAL'='FALSE')",
			"ALTER TABLE " + database + ".events ADD PARTITION (dt='2024-01-01')",
			"INSERT INTO " + database + ".events PARTITION (dt='2024-01-01') VALUES (1, 'x'), (2, 'y')",
			"CREATE EXTERNAL TABLE " + database + ".ext_users (id INT COMMENT 'the id', name STRING) ROW FORMAT DELIMITED FIELDS TERMINATED BY ',' STORED AS TEXTFILE LOCATION '/tmp/bytebase/ext_users'",
			"CREATE VIEW " + database + ".v_orders AS SELECT id, note FROM " + database + ".orders",
			"CREATE MATERIALIZED VIEW " + database + ".mv_orders DISABLE REWRITE AS SELECT id, note FROM " + database + ".orders",
		}, ";\n")+";", opts)
		require.NoError(t, err)

		var executes, responses int
		for _, log := range logs {
			switch log.Type {
			case storepb.TaskRunLog_COMMAND_EXECUTE:
				executes++
				require.NotEmpty(t, log.GetCommandExecute().GetRange())
			case storepb.TaskRunLog_COMMAND_RESPONSE:
				responses++
				require.Empty(t, log.GetCommandResponse().GetError())
			default:
				// Hive issues no transaction control, so a task run logs none.
			}
		}
		require.Equal(t, 10, executes)
		require.Equal(t, 10, responses)
	})

	t.Run("sync instance", func(t *testing.T) {
		instance, err := session.SyncInstance(ctx)
		require.NoError(t, err)
		require.Equal(t, "4.0.1", instance.Version)
		var names []string
		for _, database := range instance.Databases {
			names = append(names, database.Name)
		}
		require.Contains(t, names, database)
	})

	t.Run("sync database", func(t *testing.T) {
		metadata, err := session.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Equal(t, database, metadata.Name)
		require.Len(t, metadata.Schemas, 1)
		schema := metadata.Schemas[0]

		require.Len(t, schema.Tables, 2)
		orders := tableNamed(t, schema.Tables, "orders")
		require.Equal(t, int64(2), orders.RowCount)
		require.Len(t, orders.Columns, 3)
		require.Equal(t, "note", orders.Columns[2].Name)
		require.Equal(t, "string", orders.Columns[2].Type)
		require.Equal(t, "a note", orders.Columns[2].Comment)
		events := tableNamed(t, schema.Tables, "events")
		require.Len(t, events.Partitions, 1)
		require.Equal(t, "dt=2024-01-01", events.Partitions[0].Name)

		require.Len(t, schema.ExternalTables, 1)
		require.Equal(t, "ext_users", schema.ExternalTables[0].Name)
		require.Equal(t, "the id", schema.ExternalTables[0].Columns[0].Comment)

		require.Len(t, schema.Views, 1)
		require.Equal(t, "v_orders", schema.Views[0].Name)
		require.Contains(t, schema.Views[0].Definition, "SELECT")

		require.Len(t, schema.MaterializedViews, 1)
		require.Equal(t, "mv_orders", schema.MaterializedViews[0].Name)
		require.Contains(t, schema.MaterializedViews[0].Definition, "SELECT")
	})

	t.Run("query returns typed values", func(t *testing.T) {
		results, err := session.QueryConn(ctx, nil, "SELECT 1 AS one, DOUBLE(1.5) AS d, CAST(1.5 AS FLOAT) AS f", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 1)
		result := results[0]
		require.Equal(t, []string{"one", "d", "f"}, result.ColumnNames)
		require.Equal(t, []string{"INT", "DOUBLE", "FLOAT"}, result.ColumnTypeNames)
		require.Equal(t, int64(1), result.RowsCount)
		values := result.Rows[0].Values
		require.Equal(t, int32(1), values[0].GetInt32Value())
		require.Equal(t, 1.5, values[1].GetDoubleValue())
		require.Equal(t, float32(1.5), values[2].GetFloatValue())
	})

	t.Run("the query schema is selected before the script", func(t *testing.T) {
		// The driver is open on no database, so an unqualified name resolves
		// only if QueryContext.Schema selected one.
		results, err := admin.QueryConn(ctx, nil, "SELECT note FROM orders ORDER BY id", db.QueryContext{Schema: database})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Len(t, results[0].Rows, 2)
		require.Equal(t, "n1", results[0].Rows[0].Values[0].GetStringValue())
	})

	t.Run("a failing statement ends the script with its error", func(t *testing.T) {
		results, err := session.QueryConn(ctx, nil, "SELECT 1;\nSELECT * FROM no_such_table;\nSELECT 3", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Empty(t, results[0].Error)
		require.Contains(t, results[1].Error, "TStatus")
	})

	t.Run("a cut result reports the cap", func(t *testing.T) {
		results, err := session.QueryConn(ctx, nil, "SELECT * FROM orders ORDER BY id", db.QueryContext{MaximumSQLResultSize: 12})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, util.FormatMaximumSQLResultSizeMessage(12), results[0].Error)
		require.Less(t, results[0].RowsCount, int64(2))
	})

	t.Run("the row limit is applied even though the statement is not rewritten", func(t *testing.T) {
		results, err := session.QueryConn(ctx, nil, "SELECT * FROM orders ORDER BY id", db.QueryContext{Limit: 1})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, int64(1), results[0].RowsCount)
		require.Len(t, results[0].Rows, 1)
		require.Equal(t, "SELECT * FROM orders ORDER BY id", results[0].Statement, "the statement reaches Hive as the caller wrote it")
	})

	t.Run("explain returns the plan as rows", func(t *testing.T) {
		results, err := session.QueryConn(ctx, nil, "SELECT * FROM orders", db.QueryContext{Explain: true})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "EXPLAIN SELECT * FROM orders", results[0].Statement)
		require.NotEmpty(t, results[0].Rows)
		require.Nil(t, results[0].QueryPlan)
	})

	t.Run("dump restores", func(t *testing.T) {
		var out bytes.Buffer
		require.NoError(t, session.Dump(ctx, &out, &metadatapb.DatabaseSchemaMetadata{}))
		dumped := out.String()
		require.Contains(t, dumped, "-- Table structure for `orders`")
		require.Contains(t, dumped, "CREATE TABLE `"+database+"`.`orders`")
		require.Contains(t, dumped, "CREATE VIEW `"+database+"`.`v_orders`")
		require.Contains(t, dumped, "CREATE MATERIALIZED VIEW `"+database+"`.`mv_orders`")

		// The dump replays into a database of its own name, which is what the
		// runner does: drop, recreate, run the text.
		_, err := admin.Execute(ctx, "DROP DATABASE "+database+" CASCADE;\nCREATE DATABASE "+database+";", db.ExecuteOptions{})
		require.NoError(t, err)
		restored := open(database)
		_, err = restored.Execute(ctx, dumped, db.ExecuteOptions{})
		require.NoError(t, err)

		metadata, err := restored.SyncDBSchema(ctx)
		require.NoError(t, err)
		schema := metadata.Schemas[0]
		require.Len(t, schema.Tables, 2)
		require.Len(t, schema.ExternalTables, 1)
		require.Len(t, schema.Views, 1)
		require.Len(t, schema.MaterializedViews, 1)
		require.Len(t, tableNamed(t, schema.Tables, "orders").Columns, 3)

		var again bytes.Buffer
		require.NoError(t, restored.Dump(ctx, &again, &metadatapb.DatabaseSchemaMetadata{}))
		// SHOW CREATE states the times the server last modified an object, and a
		// replay is a modification, so the two dumps are compared with those
		// masked; nothing else about them may move.
		require.Equal(t, maskModifiedTimes(dumped), maskModifiedTimes(again.String()), "the dump does not round-trip")
	})
}

// modifiedTimeProperty matches the table properties Hive stamps with the time
// it last changed an object.
var modifiedTimeProperty = regexp.MustCompile(`'(last_modified_time|transient_lastDdlTime)'='[0-9]+'`)

// maskModifiedTimes replaces the server's modification times with a fixed value,
// so two dumps of the same schema compare equal.
func maskModifiedTimes(dump string) string {
	return modifiedTimeProperty.ReplaceAllString(dump, "'$1'='0'")
}

// tableNamed returns the table a synced schema calls name.
func tableNamed(t *testing.T, tables []*metadatapb.TableMetadata, name string) *metadatapb.TableMetadata {
	t.Helper()
	for _, table := range tables {
		if table.Name == name {
			return table
		}
	}
	t.Fatalf("no table %q in the synced schema", name)
	return nil
}
