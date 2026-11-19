//nolint:revive
package util

import (
	"fmt"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omnidriver "github.com/bytebase/omni/driver"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// OmniResultToQueryResult converts one result of the driver contract
// (github.com/bytebase/omni/driver) into the API's QueryResult. Every adapter
// over an omni driver returns its results through this, so the conversion lives
// here rather than in each engine's package.
//
// maxResultBytes is the cap the caller passed as QueryOptions.MaxResultBytes;
// the contract stops reading at it and sets Truncated, and the message the API
// shows for that cap is the caller's to format.
func OmniResultToQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	if result == nil {
		return &v1pb.QueryResult{}
	}
	out := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	if len(result.Columns) > 0 {
		out.ColumnNames = make([]string, len(result.Columns))
		out.ColumnTypeNames = make([]string, len(result.Columns))
		for i, column := range result.Columns {
			out.ColumnNames[i] = column.Name
			out.ColumnTypeNames[i] = column.TypeName
		}
	}
	for _, row := range result.Rows {
		values := make([]*v1pb.RowValue, len(row))
		for i, value := range row {
			values[i] = omniRowValue(value)
		}
		out.Rows = append(out.Rows, &v1pb.QueryRow{Values: values})
	}
	// A statement that returns no columns reports a count instead, which the API
	// shows as one "Affected Rows" row. The contract leaves the count nil for a
	// statement the engine states none for, which is not the same as zero.
	if len(result.Columns) == 0 && result.AffectedRows != nil {
		affected := BuildAffectedRowsResult(*result.AffectedRows, nil)
		affected.Statement = out.Statement
		affected.Latency = out.Latency
		out = affected
	}
	for _, message := range result.Messages {
		out.Messages = append(out.Messages, &v1pb.QueryResult_Message{
			Level:   v1pb.QueryResult_Message_Level(v1pb.QueryResult_Message_Level_value[message.Level]),
			Content: message.Content,
		})
	}
	if result.Plan != nil {
		out.QueryPlan = &v1pb.QueryResult_QueryPlan{
			Format:   v1pb.QueryOption_ExplainFormat(v1pb.QueryOption_ExplainFormat_value[result.Plan.Format]),
			Executed: result.Plan.Executed,
			Content:  result.Plan.Content,
		}
	}
	if result.Err != nil {
		out.Error = result.Err.Error()
	} else if result.Truncated {
		out.Error = FormatMaximumSQLResultSizeMessage(maxResultBytes)
	}
	out.RowsCount = int64(len(out.Rows))
	return out
}

// omniRowValue converts one cell of a contract result into the API's RowValue.
func omniRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: float32(value.Float64)}}
	case omnidriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case omnidriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Uint32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(value.Uint64)}}
	case omnidriver.Uint64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: value.Uint64}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case omnidriver.Structured:
		structured, err := structpb.NewValue(value.Structured)
		if err != nil {
			// A value the JSON model cannot hold is reported rather than dropped.
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value.Structured)}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_ValueValue{ValueValue: structured}}
	case omnidriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
			GoogleTimestamp: timestamppb.New(value.Time),
			Accuracy:        value.Precision,
		}}}
	case omnidriver.TimestampTZ:
		zone, offset := value.Time.Zone()
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
			GoogleTimestamp: timestamppb.New(value.Time),
			Zone:            zone,
			Offset:          int32(offset),
			Accuracy:        value.Precision,
		}}}
	default:
		return NullRowValue
	}
}
