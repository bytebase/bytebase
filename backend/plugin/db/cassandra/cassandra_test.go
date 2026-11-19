package cassandra

import (
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestConvertValue(t *testing.T) {
	timestamp := time.Date(2026, 11, 18, 7, 30, 0, 0, time.UTC)
	zone := time.FixedZone("PDT", -7*60*60)
	for _, tc := range []struct {
		name  string
		value omnidriver.Value
		want  *v1pb.RowValue
	}{
		{
			name:  "null",
			value: omnidriver.Value{Kind: omnidriver.Null},
			want:  util.NullRowValue,
		},
		{
			name:  "bool",
			value: omnidriver.Value{Kind: omnidriver.Bool, Bool: true},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}},
		},
		{
			name:  "blob",
			value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte("bytes")},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte("bytes")}},
		},
		{
			// A CQL float is 32-bit, and Bytebase showed it as a FloatValue.
			name:  "float",
			value: omnidriver.Value{Kind: omnidriver.Float32, Float64: 1.5},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: 1.5}},
		},
		{
			name:  "double",
			value: omnidriver.Value{Kind: omnidriver.Float64, Float64: 3.25},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 3.25}},
		},
		{
			// smallint and tinyint widened to Int32Value, int and bigint to Int64Value.
			name:  "small int",
			value: omnidriver.Value{Kind: omnidriver.Int32, Int64: 7},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: 7}},
		},
		{
			name:  "big int",
			value: omnidriver.Value{Kind: omnidriver.Int64, Int64: 1 << 40},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1 << 40}},
		},
		{
			name:  "unsigned int",
			value: omnidriver.Value{Kind: omnidriver.Uint32, Uint64: 7},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: 7}},
		},
		{
			name:  "unsigned big int",
			value: omnidriver.Value{Kind: omnidriver.Uint64, Uint64: 1 << 40},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: 1 << 40}},
		},
		{
			// text, ascii, varchar, uuid, inet, decimal, varint, time and duration.
			name:  "string",
			value: omnidriver.Value{Kind: omnidriver.String, String: "text"},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "text"}},
		},
		{
			name:  "structured",
			value: omnidriver.Value{Kind: omnidriver.Structured, Structured: []any{1.0, "a"}},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: `[1,"a"]`}},
		},
		{
			name:  "timestamp",
			value: omnidriver.Value{Kind: omnidriver.Timestamp, Time: timestamp, Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(timestamp),
				Accuracy:        3,
			}}},
		},
		{
			name:  "timestamp with zone",
			value: omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: timestamp.In(zone), Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(timestamp.In(zone)),
				Zone:            "PDT",
				Offset:          -7 * 60 * 60,
				Accuracy:        3,
			}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, convertValue(tc.value))
		})
	}
}

func TestToQueryResult(t *testing.T) {
	result := toQueryResult(&omnidriver.Result{
		Statement: "SELECT * FROM t",
		Columns: []omnidriver.Column{
			{Name: "id", TypeName: "int"},
			{Name: "name", TypeName: "text"},
		},
		Rows: [][]omnidriver.Value{
			{{Kind: omnidriver.Int32, Int64: 1}, {Kind: omnidriver.String, String: "a"}},
			{{Kind: omnidriver.Int32, Int64: 2}, {Kind: omnidriver.Null}},
		},
	}, db.QueryContext{})

	require.Equal(t, "SELECT * FROM t", result.Statement)
	require.Equal(t, []string{"id", "name"}, result.ColumnNames)
	require.Equal(t, []string{"int", "text"}, result.ColumnTypeNames)
	require.Equal(t, int64(2), result.RowsCount)
	require.Empty(t, result.Error)
	require.Equal(t, []*v1pb.QueryRow{
		{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_Int32Value{Int32Value: 1}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: "a"}},
		}},
		{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_Int32Value{Int32Value: 2}},
			util.NullRowValue,
		}},
	}, result.Rows)
}

func TestToQueryResultReportsTruncationAndErrors(t *testing.T) {
	truncated := toQueryResult(&omnidriver.Result{Statement: "SELECT * FROM t", Truncated: true}, db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(1<<20), truncated.Error)

	failed := toQueryResult(&omnidriver.Result{Statement: "SELECT * FROM t", Err: errTest}, db.QueryContext{})
	require.Equal(t, "failed: refused", failed.Error)
}

var errTest = &testError{}

type testError struct{}

func (*testError) Error() string { return "failed: refused" }

func TestExecuteLoggerReportsEachStep(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	}
	logger := executeLogger{opts: opts}
	logger.CommandExecute(omnidriver.Command{Text: "SELECT 1", ByteStart: 0, ByteEnd: 8})
	logger.CommandResponse(0, nil, errTest)
	logger.TransactionControl(omnidriver.Begin, nil)
	logger.Retry(errTest, 1, 3)

	require.Len(t, logs, 4)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, "SELECT 1", logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
	require.Equal(t, "failed: refused", logs[1].GetCommandResponse().GetError())
	require.Equal(t, storepb.TaskRunLog_TRANSACTION_CONTROL, logs[2].Type)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[2].GetTransactionControl().GetType())
	require.Equal(t, storepb.TaskRunLog_RETRY_INFO, logs[3].Type)
	require.Equal(t, int32(1), logs[3].GetRetryInfo().GetRetryCount())
}

func TestExecuteLoggerLogsTheCommandRange(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
		logs = append(logs, log)
		return nil
	}}
	executeLogger{opts: opts}.CommandExecute(omnidriver.Command{Text: "SELECT 1", ByteStart: 3, ByteEnd: 11})

	require.Len(t, logs, 1)
	require.Equal(t, &storepb.Range{Start: 3, End: 11}, logs[0].GetCommandExecute().GetRange())
}

func TestExecuteLoggerIsSilentWithoutACallback(t *testing.T) {
	require.NotPanics(t, func() {
		logger := executeLogger{}
		logger.CommandExecute(omnidriver.Command{Text: "SELECT 1"})
		logger.CommandResponse(0, nil, nil)
		logger.TransactionControl(omnidriver.Commit, nil)
		logger.Retry(nil, 1, 1)
	})
}

func TestFormatAddress(t *testing.T) {
	require.Equal(t, "node1.example.com:9042", formatAddress("node1.example.com", "9042"))
	require.Equal(t, "node1.example.com", formatAddress("node1.example.com", ""))
	require.Equal(t, "[2001:db8::1]:9042", formatAddress("2001:db8::1", "9042"))
	require.Equal(t, "[2001:db8::1]:9042", formatAddress("[2001:db8::1]", "9042"))
}
