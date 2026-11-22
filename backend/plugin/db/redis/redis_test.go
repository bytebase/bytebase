package redis

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func openTestDriver(t *testing.T) db.Driver {
	t.Helper()
	container := testcontainer.SharedRedisClusterContainer(t)
	d, err := newDriver().Open(context.Background(), storepb.Engine_REDIS, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:      storepb.DataSourceType_ADMIN,
			Host:      container.GetHost(),
			Port:      container.GetPort(),
			RedisType: storepb.DataSource_CLUSTER,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "0"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close(context.Background())) })
	return d
}

// TestLegacyParity replays the corpus recorded from the old driver before this
// change deleted it: the same setup, sync answers, query results and all-nodes
// answers. testdata/legacy.json is that recording; the old path ran on the same
// redis:7.4.12-alpine pin, so the version and database list are comparable.
func TestLegacyParity(t *testing.T) {
	ctx := context.Background()
	d := openTestDriver(t)

	_, err := d.Execute(ctx, "FLUSHALL\nSET greeting \"hello world\"\nRPUSH queue first second", db.ExecuteOptions{})
	require.NoError(t, err)

	instance, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)

	single, err := d.QueryConn(ctx, "GET greeting\n\nGET missing\nLRANGE queue 0 -1\nDBSIZE", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	for _, result := range single {
		result.Latency = nil
	}
	all, err := d.QueryConn(ctx, "GET greeting\nDBSIZE", db.QueryContext{
		MaximumSQLResultSize: 1 << 20,
		Option:               &v1pb.QueryOption{RedisRunCommandsOn: v1pb.QueryOption_ALL_NODES},
	})
	require.NoError(t, err)
	for _, result := range all {
		result.Latency = nil
	}

	got, err := json.MarshalIndent(map[string]any{
		"version":   instance.Version,
		"databases": instance.Databases,
		"schema":    schema,
		"single":    envelope(single),
		"all":       envelope(all),
	}, "", "  ")
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join("testdata", "legacy.json"))
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
}

func envelope(results []*v1pb.QueryResult) []json.RawMessage {
	var out []json.RawMessage
	for _, result := range results {
		b, err := protojson.Marshal(proto.Clone(result).(*v1pb.QueryResult))
		if err != nil {
			panic(err)
		}
		out = append(out, b)
	}
	return out
}

func TestQueryConnScripts(t *testing.T) {
	ctx := context.Background()
	d := openTestDriver(t)
	_, err := d.Execute(ctx, "FLUSHALL\nSET a \"two words\"\nSET b value", db.ExecuteOptions{})
	require.NoError(t, err)

	results, err := d.QueryConn(ctx, "GET a\n\nGET b\n   ", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for i, want := range []string{"GET a", "GET b"} {
		require.Equal(t, want, results[i].Statement)
		require.Equal(t, []string{"#", "Value"}, results[i].ColumnNames)
		require.Equal(t, []string{"INT", "TEXT"}, results[i].ColumnTypeNames)
		require.Len(t, results[i].Rows, 1)
	}
	require.Equal(t, "two words", results[0].Rows[0].Values[1].GetStringValue())
	require.Equal(t, "value", results[1].Rows[0].Values[1].GetStringValue())

	_, err = d.QueryConn(ctx, `SET a "unclosed`, db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.ErrorContains(t, err, "failed to split command")

	// Known difference: the old shell lexer treated # as the start of a comment,
	// so this line was GET with no key; Redis has no comment syntax.
	results, err = d.QueryConn(ctx, "GET #", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "GET #", results[0].Statement)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, util.NullRowValue, results[0].Rows[0].Values[1])
}

// TestQueryConnExplainRefused holds the refusal in front of the session, as the
// old driver had it: an explain never reaches the server.
func TestQueryConnExplainRefused(t *testing.T) {
	_, err := (&Driver{}).QueryConn(context.Background(), "GET key", db.QueryContext{Explain: true})
	require.ErrorContains(t, err, "Redis does not support EXPLAIN")
}

func TestQueryConnRejectsUnknownOption(t *testing.T) {
	_, err := (&Driver{}).QueryConn(context.Background(), "GET key", db.QueryContext{
		Option: &v1pb.QueryOption{RedisRunCommandsOn: v1pb.QueryOption_RedisRunCommandsOn(99)},
	})
	require.ErrorContains(t, err, "unknown RedisRunCommandOn enum")
}

// TestExecuteRefusedStopsScript records a known difference from the pipeline:
// the old Execute sent every command and failed afterwards; the driver stops at
// the refused command, so the commands after it do not run.
func TestExecuteRefusedStopsScript(t *testing.T) {
	ctx := context.Background()
	d := openTestDriver(t)
	_, err := d.Execute(ctx, "FLUSHALL\nSET counter text", db.ExecuteOptions{})
	require.NoError(t, err)

	_, err = d.Execute(ctx, "INCR counter\nSET after value", db.ExecuteOptions{})
	require.ErrorContains(t, err, "ERR value is not an integer or out of range")

	results, err := d.QueryConn(ctx, "GET after", db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, util.NullRowValue, results[0].Rows[0].Values[1])
}

func TestExecuteCreateDatabaseAndDump(t *testing.T) {
	ctx := context.Background()
	d := openTestDriver(t)
	_, err := d.Execute(ctx, "SET a 1", db.ExecuteOptions{CreateDatabase: true})
	require.ErrorContains(t, err, "redis: cannot create database")

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, nil))
	require.Empty(t, out.String())
}

// TestExecuteLogsItsSteps covers the ExecuteLogger bridge: each command and its
// response reaches the task run's log with the command's byte range.
func TestExecuteLogsItsSteps(t *testing.T) {
	ctx := context.Background()
	d := openTestDriver(t)
	var logs []*storepb.TaskRunLog
	_, err := d.Execute(ctx, "SET a 1\nGET a", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	require.NoError(t, err)
	require.Len(t, logs, 4)
	for i, want := range []struct {
		byteStart int32
		byteEnd   int32
	}{
		{0, 7},
		{0, 0},
		{8, 13},
		{0, 0},
	} {
		if i%2 == 0 {
			require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[i].Type)
			require.Empty(t, logs[i].GetCommandExecute().Statement)
			require.Equal(t, want.byteStart, logs[i].GetCommandExecute().GetRange().GetStart())
			require.Equal(t, want.byteEnd, logs[i].GetCommandExecute().GetRange().GetEnd())
		} else {
			require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[i].Type)
		}
	}
}

func TestSyncInstanceWithoutConfigPermission(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedRedisClusterContainer(t)
	open := func(username, password string) db.Driver {
		d, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Type:      storepb.DataSourceType_ADMIN,
				Host:      container.GetHost(),
				Port:      container.GetPort(),
				Username:  username,
				RedisType: storepb.DataSource_CLUSTER,
			},
			Password: password,
		})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, d.Close(ctx)) })
		return d
	}

	_, err := open("", "").Execute(ctx, "ACL SETUSER noconfig on >noconfig-pw ~* +@all -config", db.ExecuteOptions{})
	require.NoError(t, err)

	limited := open("noconfig", "noconfig-pw")
	_, err = limited.Execute(ctx, "CONFIG GET databases", db.ExecuteOptions{})
	require.ErrorContains(t, err, "NOPERM")

	instance, err := limited.SyncInstance(ctx)
	require.NoError(t, err)
	require.Len(t, instance.Databases, 1)
	require.Equal(t, "0", instance.Databases[0].Name)
}

// TestToRowValue holds the contract value shapes Redis reports: an answer is
// text, an array position is an int, and a missing key is NULL.
func TestToRowValue(t *testing.T) {
	require.Equal(t, util.NullRowValue, toRowValue(omnidriver.Value{Kind: omnidriver.Null}))
	require.Equal(t, int32(2), toRowValue(omnidriver.Value{Kind: omnidriver.Int32, Int64: 2}).GetInt32Value())
	require.Equal(t, "value", toRowValue(omnidriver.Value{Kind: omnidriver.String, String: "value"}).GetStringValue())
}

func TestAddressList(t *testing.T) {
	require.Equal(t, "127.0.0.1:6379", joinAddress("127.0.0.1", "6379"))
	require.Equal(t, "redis.example.com:6379", joinAddress("redis.example.com", "6379"))
	require.Equal(t, "[2001:db8::1]:6379", joinAddress("2001:db8::1", "6379"))
	require.Equal(t, "[2001:db8::1]:6379", joinAddress("[2001:db8::1]", "6379"))

	ds := &storepb.DataSource{
		Host: "2001:db8::1",
		Port: "6379",
		AdditionalAddresses: []*storepb.DataSource_Address{
			{Host: "[2001:db8::2]", Port: "6380"},
			{Host: "node3.example.com", Port: "6381"},
		},
	}
	require.Equal(t, "[2001:db8::1]:6379,[2001:db8::2]:6380,node3.example.com:6381", addressList(ds))
}
