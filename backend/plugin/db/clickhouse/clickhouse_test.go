package clickhouse

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestClickHouseConfig(t *testing.T) {
	dataSource := &storepb.DataSource{
		Host:                      "db.example.com",
		Port:                      "9000",
		Username:                  "default",
		ExtraConnectionParameters: map[string]string{"max_threads": "3"},
	}
	cfg, err := clickhouseConfig(dataSource, "secret", "sales")
	require.NoError(t, err)
	require.Equal(t, "db.example.com", cfg.Host)
	require.Equal(t, "9000", cfg.Port)
	require.Equal(t, "sales", cfg.Database)
	require.Equal(t, "default", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, map[string]string{"max_threads": "3"}, cfg.Params)
	require.Nil(t, cfg.TLS, "a data source without SSL leaves the client's own negotiation")

	dataSource.UseSsl = true
	cfg, err = clickhouseConfig(dataSource, "secret", "sales")
	require.NoError(t, err)
	require.NotNil(t, cfg.TLS)
	tlsConfig, err := cfg.TLS("db.example.com")
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	// A host and port the driver cannot dial are refused at Open.
	_, err = newDriver().Open(context.Background(), storepb.Engine_CLICKHOUSE, db.ConnectionConfig{DataSource: &storepb.DataSource{}})
	require.ErrorContains(t, err, "host must be set")
}

func TestToRowValue(t *testing.T) {
	timestamp := time.Date(2026, 11, 18, 7, 30, 0, 123000000, time.UTC)
	for _, tc := range []struct {
		name  string
		value omnidriver.Value
		want  *v1pb.RowValue
	}{
		{
			name:  "null",
			value: omnidriver.Value{Kind: omnidriver.Null},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}},
		},
		{
			name:  "int8 stays 32-bit",
			value: omnidriver.Value{Kind: omnidriver.Int32, Int64: -1},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: -1}},
		},
		{
			name:  "int64",
			value: omnidriver.Value{Kind: omnidriver.Int64, Int64: 1},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
		},
		{
			name:  "uint8 stays 32-bit",
			value: omnidriver.Value{Kind: omnidriver.Uint32, Uint64: 2},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: 2}},
		},
		{
			name:  "float32 stays a float",
			value: omnidriver.Value{Kind: omnidriver.Float32, Float64: 1.5},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: 1.5}},
		},
		{
			name:  "float64",
			value: omnidriver.Value{Kind: omnidriver.Float64, Float64: 2.5},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 2.5}},
		},
		{
			name:  "a date-time is typed with its precision",
			value: omnidriver.Value{Kind: omnidriver.Timestamp, Time: timestamp, Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{GoogleTimestamp: timestamppb.New(timestamp), Accuracy: 3},
			}},
		},
		{
			name:  "a zoned date-time keeps its zone and offset",
			value: omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: timestamp, Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{
				TimestampTzValue: &v1pb.RowValue_TimestampTZ{
					GoogleTimestamp: timestamppb.New(timestamp),
					Zone:            "UTC",
					Accuracy:        3,
				},
			}},
		},
		{
			name:  "a time is a clock string",
			value: omnidriver.Value{Kind: omnidriver.String, String: "07:30:00"},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "07:30:00"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toRowValue(tc.value))
		})
	}
}

func TestToQueryResult(t *testing.T) {
	toQueryResult := func(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
		return util.OmniResultToQueryResultWith(result, queryContext.MaximumSQLResultSize, clickhouseQueryResultOptions(queryContext.Explain))
	}
	result := &omnidriver.Result{
		Statement: "SELECT 1",
		Columns:   []omnidriver.Column{{Name: "1", TypeName: "UINT8"}},
		Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Uint32, Uint64: 1}}},
		Latency:   2 * time.Second,
	}
	converted := toQueryResult(result, db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.Equal(t, "SELECT 1", converted.Statement)
	require.Equal(t, []string{"1"}, converted.ColumnNames)
	require.Equal(t, []string{"UINT8"}, converted.ColumnTypeNames)
	require.Equal(t, int64(1), converted.RowsCount)
	require.Equal(t, time.Duration(2*time.Second), converted.Latency.AsDuration())
	require.Empty(t, converted.Error)
	require.Nil(t, converted.QueryPlan)

	// A caller's own EXPLAIN carries the typed plan of #1394's fix.
	explain := toQueryResult(&omnidriver.Result{
		Statement: "EXPLAIN SELECT 1",
		Plan:      &omnidriver.Plan{Format: "TEXT", Executed: true},
	}, db.QueryContext{})
	require.Equal(t, &v1pb.QueryResult_QueryPlan{
		Format:   v1pb.QueryOption_TEXT,
		Executed: true,
	}, explain.QueryPlan)

	// An explain request leaves the plan to the API handler, which describes
	// the request's format.
	request := toQueryResult(&omnidriver.Result{
		Statement: "EXPLAIN SELECT 1",
		Plan:      &omnidriver.Plan{Format: "TEXT"},
	}, db.QueryContext{Explain: true})
	require.Nil(t, request.QueryPlan)

	truncated := toQueryResult(&omnidriver.Result{
		Statement: "SELECT 1",
		Columns:   []omnidriver.Column{{Name: "1", TypeName: "UINT8"}},
		Truncated: true,
	}, db.QueryContext{MaximumSQLResultSize: 1})
	require.Contains(t, truncated.Error, "exceeds max allowed output size")

	failed := toQueryResult(&omnidriver.Result{Statement: "bad", Err: io.ErrUnexpectedEOF}, db.QueryContext{})
	require.Equal(t, io.ErrUnexpectedEOF.Error(), failed.Error)
	require.Nil(t, failed.Rows)
}

type fakeClickHouseSession struct {
	queryFn func(ctx context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error)
	closed  bool
}

func (s *fakeClickHouseSession) Query(ctx context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	if s.queryFn != nil {
		return s.queryFn(ctx, script, opts)
	}
	return []*omnidriver.Result{{Statement: script}}, nil
}

func (s *fakeClickHouseSession) Close() error {
	s.closed = true
	return nil
}

type fakeClickHouseDriver struct {
	sessions      []*fakeClickHouseSession
	sessionCalls  int
	executeScript string
	executeOpts   omnidriver.ExecuteOptions
	executeFn     func(script string, opts omnidriver.ExecuteOptions) (int64, error)
	instance      *omnidriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
	dumpDDL       string
	closed        bool
}

func (*fakeClickHouseDriver) Ping(context.Context) error { return nil }
func (f *fakeClickHouseDriver) Close() error {
	f.closed = true
	return nil
}

func (f *fakeClickHouseDriver) Session(context.Context) (omnidriver.Session, error) {
	idx := f.sessionCalls
	f.sessionCalls++
	if idx < len(f.sessions) {
		return f.sessions[idx], nil
	}
	return &fakeClickHouseSession{}, nil
}

func (f *fakeClickHouseDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeClickHouseDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeClickHouseDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (f *fakeClickHouseDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func TestClickHouseAdapterSeam(t *testing.T) {
	ctx := context.Background()

	t.Run("held session survives server exception but discards poisoned or canceled session", func(t *testing.T) {
		var gotOpts omnidriver.QueryOptions
		healthy := &fakeClickHouseSession{
			queryFn: func(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				if script == "SELECT 1" {
					return []*omnidriver.Result{{Statement: script}}, nil
				}
				gotOpts = opts
				return []*omnidriver.Result{{Statement: script, Err: errors.New("Unknown expression identifier")}}, nil
			},
		}
		poisoned := &fakeClickHouseSession{
			queryFn: func(_ context.Context, script string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				if script == "SELECT 1" {
					return nil, errors.New("driver: bad connection")
				}
				return []*omnidriver.Result{{Statement: script, Err: errors.New("AggregateFunction decode error")}}, nil
			},
		}
		fresh := &fakeClickHouseSession{}

		impl := &fakeClickHouseDriver{sessions: []*fakeClickHouseSession{healthy, poisoned, fresh}}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl, WithInstanceMetadata: true}}

		// Server error whose SELECT 1 probe succeeds keeps the held session.
		res, err := d.QueryConn(ctx, "SELECT nope FROM t", db.QueryContext{
			Limit:                10,
			Explain:              true,
			Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
			MaximumSQLResultSize: 1024,
		})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Contains(t, res[0].Error, "Unknown expression identifier")
		require.Equal(t, omnidriver.QueryOptions{
			Limit:          10,
			Explain:        true,
			ExplainFormat:  "JSON",
			MaxResultBytes: 1024,
		}, gotOpts)
		require.False(t, healthy.closed)
		require.Same(t, healthy, d.Session())

		// Canceled context discards the held session.
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = d.QueryConn(canceledCtx, "SELECT 1", db.QueryContext{})
		require.NoError(t, err)
		require.True(t, healthy.closed)
		require.Nil(t, d.Session())

		// Decode/protocol error where SELECT 1 probe fails discards the session so the next call opens a new one.
		res, err = d.QueryConn(ctx, "SELECT * FROM agg", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Contains(t, res[0].Error, "AggregateFunction")
		require.True(t, poisoned.closed)
		require.Nil(t, d.Session())

		_, err = d.QueryConn(ctx, "SELECT 1", db.QueryContext{})
		require.NoError(t, err)
		require.Equal(t, 3, impl.sessionCalls)
		require.Same(t, fresh, d.Session())

		require.NoError(t, d.Close(ctx))
		require.True(t, fresh.closed)
		require.True(t, impl.closed)
	})

	t.Run("execute parses transaction directive and logs steps", func(t *testing.T) {
		var logs []*storepb.TaskRunLog
		impl := &fakeClickHouseDriver{
			executeFn: func(script string, opts omnidriver.ExecuteOptions) (int64, error) {
				opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
				opts.Log.CommandResponse(0, nil, nil)
				return 0, nil
			},
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

		_, err := d.Execute(ctx, "-- txn-mode = off\nINSERT INTO t VALUES (1)", db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.NoError(t, err)
		require.Equal(t, "INSERT INTO t VALUES (1)", impl.executeScript)
		require.Equal(t, omnidriver.AutoCommit, impl.executeOpts.Transaction)
		require.Len(t, logs, 2)
		require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
		require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
	})

	t.Run("sync and dump delegate through OmniAdapter", func(t *testing.T) {
		impl := &fakeClickHouseDriver{
			instance: &omnidriver.Instance{
				Version:   "25.1.1",
				Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "analytics"}},
				Roles:     []*omnidriver.Role{{Name: "default"}},
			},
			schema:  &metadatapb.DatabaseSchemaMetadata{Name: "analytics"},
			dumpDDL: "CREATE TABLE analytics.t (id Int64) ENGINE = MergeTree ORDER BY id;\n",
		}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl, WithInstanceMetadata: true}}

		inst, err := d.SyncInstance(ctx)
		require.NoError(t, err)
		require.Equal(t, "25.1.1", inst.Version)
		require.Len(t, inst.Databases, 1)
		require.Len(t, inst.Metadata.GetRoles(), 1)
		require.Equal(t, "default", inst.Metadata.GetRoles()[0].GetName())

		schema, err := d.SyncDBSchema(ctx)
		require.NoError(t, err)
		require.Equal(t, "analytics", schema.GetName())

		var out bytes.Buffer
		require.NoError(t, d.Dump(ctx, &out, schema))
		require.Contains(t, out.String(), "CREATE TABLE analytics.t")
	})
}
