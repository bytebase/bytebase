package redis

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestQueryConnExplainRefused holds the refusal in front of the session: an
// explain never reaches the server.
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

func TestOpenConfigValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("non-numeric database index fails before dialing", func(t *testing.T) {
		_, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
			DataSource:        &storepb.DataSource{Host: "127.0.0.1", Port: "6379"},
			ConnectionContext: db.ConnectionContext{DatabaseName: "not-a-number"},
		})
		require.ErrorContains(t, err, "failed to convert database not-a-number to int")
	})

	t.Run("unsupported redis type is rejected", func(t *testing.T) {
		_, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:      "127.0.0.1",
				Port:      "6379",
				RedisType: storepb.DataSource_RedisType(99),
			},
		})
		require.ErrorContains(t, err, "unsupported redis type")
	})

	t.Run("invalid TLS material fails at Open", func(t *testing.T) {
		_, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Host:   "127.0.0.1",
				Port:   "6379",
				UseSsl: true,
				SslKey: "orphan-key",
			},
		})
		require.ErrorContains(t, err, "redis: failed to get tls config")
	})

	t.Run("standalone, sentinel, and cluster topologies open lazily without dialing", func(t *testing.T) {
		for _, ds := range []*storepb.DataSource{
			{Host: "127.0.0.1", Port: "6379", RedisType: storepb.DataSource_STANDALONE},
			{Host: "127.0.0.1", Port: "26379", RedisType: storepb.DataSource_SENTINEL, MasterName: "mymaster"},
			{Host: "127.0.0.1", Port: "7000", RedisType: storepb.DataSource_CLUSTER},
		} {
			d, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
				DataSource:        ds,
				ConnectionContext: db.ConnectionContext{DatabaseName: "0"},
			})
			require.NoError(t, err)
			require.NoError(t, d.Close(ctx))
		}
	})
}

type fakeRedisSession struct {
	gotScript string
	gotOpts   omnidriver.QueryOptions
	results   []*omnidriver.Result
	closed    bool
}

func (s *fakeRedisSession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (s *fakeRedisSession) Close() error {
	s.closed = true
	return nil
}

type fakeRedisClusterSession struct {
	fakeRedisSession
	allNodesCalled bool
}

func (s *fakeRedisClusterSession) QueryAllNodes(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.allNodesCalled = true
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

type fakeRedisDriver struct {
	session       omnidriver.Session
	executeScript string
	executeOpts   omnidriver.ExecuteOptions
	executeFn     func(script string, opts omnidriver.ExecuteOptions) (int64, error)
	instance      *omnidriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
}

func (*fakeRedisDriver) Ping(context.Context) error { return nil }
func (*fakeRedisDriver) Close() error               { return nil }
func (f *fakeRedisDriver) Session(context.Context) (omnidriver.Session, error) {
	return f.session, nil
}

func (f *fakeRedisDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeRedisDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeRedisDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (*fakeRedisDriver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

func TestRedisAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("QueryConn dispatches SINGLE_NODE and ALL_NODES and rejects ALL_NODES on non-cluster session", func(t *testing.T) {
		clusterSess := &fakeRedisClusterSession{
			fakeRedisSession: fakeRedisSession{
				results: []*omnidriver.Result{{
					Statement: "DBSIZE",
					Columns:   []omnidriver.Column{{Name: "#", TypeName: "INT"}, {Name: "Value", TypeName: "TEXT"}},
					Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int32, Int64: 1}, {Kind: omnidriver.String, String: "2"}}},
				}},
			},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeRedisDriver{session: clusterSess}}}

		got, err := d.QueryConn(ctx, "DBSIZE", db.QueryContext{
			MaximumSQLResultSize: 1024,
			Option:               &v1pb.QueryOption{RedisRunCommandsOn: v1pb.QueryOption_ALL_NODES},
		})
		require.NoError(t, err)
		require.True(t, clusterSess.allNodesCalled)
		require.True(t, clusterSess.closed)
		require.Equal(t, int64(1024), clusterSess.gotOpts.MaxResultBytes)
		require.Len(t, got, 1)
		require.Equal(t, "2", got[0].Rows[0].Values[1].GetStringValue())

		standaloneSess := &fakeRedisSession{}
		dStandalone := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeRedisDriver{session: standaloneSess}}}
		_, err = dStandalone.QueryConn(ctx, "DBSIZE", db.QueryContext{
			Option: &v1pb.QueryOption{RedisRunCommandsOn: v1pb.QueryOption_ALL_NODES},
		})
		require.ErrorContains(t, err, "expect *redis.ClusterClient")
		require.True(t, standaloneSess.closed)
	})

	t.Run("Execute forwards CreateDatabase and logs command byte ranges", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		impl := &fakeRedisDriver{
			executeFn: func(_ string, opts omnidriver.ExecuteOptions) (int64, error) {
				opts.Log.CommandExecute(omnidriver.Command{Text: "SET a 1", ByteStart: 0, ByteEnd: 7})
				opts.Log.CommandResponse(0, nil, nil)
				opts.Log.CommandExecute(omnidriver.Command{Text: "GET a", ByteStart: 8, ByteEnd: 13})
				opts.Log.CommandResponse(0, nil, nil)
				return 0, nil
			},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

		_, err := d.Execute(ctx, "SET a 1\nGET a", db.ExecuteOptions{
			CreateDatabase: true,
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.NoError(t, err)
		require.True(t, impl.executeOpts.CreateDatabase)
		require.Len(t, logs, 4)
		require.Equal(t, &storepb.Range{Start: 0, End: 7}, logs[0].GetCommandExecute().GetRange())
		require.Equal(t, &storepb.Range{Start: 8, End: 13}, logs[2].GetCommandExecute().GetRange())
	})

	t.Run("SyncInstance, SyncDBSchema, and Dump delegate", func(t *testing.T) {
		impl := &fakeRedisDriver{
			instance: &omnidriver.Instance{
				Version:   "7.4.12",
				Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "0"}},
			},
			schema: &metadatapb.DatabaseSchemaMetadata{Name: "0"},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

		inst, err := d.SyncInstance(ctx)
		require.NoError(t, err)
		require.Equal(t, "7.4.12", inst.Version)
		require.Len(t, inst.Databases, 1)

		schema, err := d.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Equal(t, "0", schema.GetName())

		var out bytes.Buffer
		require.NoError(t, d.Dump(ctx, &out, schema))
		require.Empty(t, out.String())
	})
}
