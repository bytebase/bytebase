package clickhouse

import (
	"context"
	"io"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
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

	// A host and port the driver cannot dial are refused at Open, where the old
	// driver built an unusable ":" address and failed at the first statement.
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
			name:  "a zoned date-time keeps its kind",
			value: omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: timestamp},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{GoogleTimestamp: timestamppb.New(timestamp)},
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

	// An explain request leaves the plan to the API handler, as the old driver
	// did by leaving the field empty.
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
