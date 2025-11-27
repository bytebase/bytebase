package spanner

import (
	"math"
	"testing"
	"time"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestGetDatabaseFromDSN(t *testing.T) {
	tests := []struct {
		dsn   string
		match bool
		want  string
	}{
		{
			dsn:   "projects/p/instances/i/databases/d",
			match: true,
			want:  "d",
		},
		{
			dsn:   "projects/p/instances/i/databases/",
			match: false,
			want:  "",
		},
	}
	a := require.New(t)
	for i := range tests {
		test := tests[i]
		got, err := getDatabaseFromDSN(test.dsn)
		if test.match {
			a.NoError(err)
			a.Equal(test.want, got)
		} else {
			a.Error(err)
		}
	}
}

func TestGetColumnTypeName(t *testing.T) {
	tests := []struct {
		spannerType sppb.Type
		want        string
	}{
		{
			spannerType: sppb.Type{
				Code: sppb.TypeCode_JSON,
			},
			want: "JSON",
		},
		{
			spannerType: sppb.Type{
				Code: sppb.TypeCode_DATE,
			},
			want: "DATE",
		},
		{
			spannerType: sppb.Type{
				Code: sppb.TypeCode_TIMESTAMP,
			},
			want: "TIMESTAMP",
		},
		{
			spannerType: sppb.Type{
				Code: sppb.TypeCode_ARRAY,
				ArrayElementType: &sppb.Type{
					Code: sppb.TypeCode_BYTES,
				},
			},
			want: "[]BYTES",
		},
	}
	a := require.New(t)
	for i := range tests {
		got, err := getColumnTypeName(&tests[i].spannerType)
		a.NoError(err)
		a.Equal(tests[i].want, got)
	}
}

// Helper function to create expected timestamp RowValue for tests
func timestampRowValue(rfc3339 string, accuracy int32) *v1pb.RowValue {
	t, _ := time.Parse(time.RFC3339Nano, rfc3339)
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
		TimestampValue: &v1pb.RowValue_Timestamp{
			GoogleTimestamp: timestamppb.New(t),
			Accuracy:        accuracy,
		},
	}}
}

func TestConvertSpannerValue(t *testing.T) {
	tests := []struct {
		name     string
		colType  *sppb.Type
		value    *structpb.Value
		expected *v1pb.RowValue
	}{
		{
			name:     "nil value",
			colType:  nil,
			value:    nil,
			expected: util.NullRowValue,
		},
		{
			name:     "null value",
			colType:  &sppb.Type{Code: sppb.TypeCode_STRING},
			value:    structpb.NewNullValue(),
			expected: util.NullRowValue,
		},
		{
			name:    "string value",
			colType: &sppb.Type{Code: sppb.TypeCode_STRING},
			value:   structpb.NewStringValue("hello"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "hello",
			}},
		},
		{
			name:    "bool value - true",
			colType: &sppb.Type{Code: sppb.TypeCode_BOOL},
			value:   structpb.NewBoolValue(true),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{
				BoolValue: true,
			}},
		},
		{
			name:    "bool value - false",
			colType: &sppb.Type{Code: sppb.TypeCode_BOOL},
			value:   structpb.NewBoolValue(false),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{
				BoolValue: false,
			}},
		},
		{
			name:    "float64 value",
			colType: &sppb.Type{Code: sppb.TypeCode_FLOAT64},
			value:   structpb.NewNumberValue(3.14159),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{
				DoubleValue: 3.14159,
			}},
		},
		{
			name:    "int64 small value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(42),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: 42,
			}},
		},
		{
			name:    "int64 large value - near 2^53",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(9007199254740992), // 2^53
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: 9007199254740992,
			}},
		},
		{
			name:    "int64 large value - within float64 safe range",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(9007199254740991), // 2^53 - 1 (max safe int in float64)
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: 9007199254740991,
			}},
		},
		{
			name:    "int64 negative large value - within float64 safe range",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(-9007199254740991), // -(2^53 - 1)
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: -9007199254740991,
			}},
		},
		{
			name:    "int64 as string - small value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewStringValue("42"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: 42,
			}},
		},
		{
			name:    "int64 as string - max value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewStringValue("9223372036854775807"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: math.MaxInt64,
			}},
		},
		{
			name:    "int64 as string - min value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewStringValue("-9223372036854775808"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: math.MinInt64,
			}},
		},
		{
			name:    "int64 as string - invalid falls back to string",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewStringValue("not-a-number"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "not-a-number",
			}},
		},
		{
			name:    "number without type metadata defaults to double",
			colType: nil,
			value:   structpb.NewNumberValue(42),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{
				DoubleValue: 42,
			}},
		},
		{
			name:    "array value",
			colType: &sppb.Type{Code: sppb.TypeCode_ARRAY},
			value:   structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewNumberValue(1), structpb.NewNumberValue(2), structpb.NewNumberValue(3)}}),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "[1,2,3]",
			}},
		},
		{
			name:    "struct value",
			colType: &sppb.Type{Code: sppb.TypeCode_STRUCT},
			value: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
				"name": structpb.NewStringValue("Alice"),
				"age":  structpb.NewNumberValue(30),
			}}),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: `{"age":30,"name":"Alice"}`,
			}},
		},
		{
			name:    "nested array",
			colType: &sppb.Type{Code: sppb.TypeCode_ARRAY},
			value: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewNumberValue(1), structpb.NewNumberValue(2)}}),
				structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{structpb.NewNumberValue(3), structpb.NewNumberValue(4)}}),
			}}),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "[[1,2],[3,4]]",
			}},
		},
		{
			name:    "array of structs",
			colType: &sppb.Type{Code: sppb.TypeCode_ARRAY},
			value: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"id": structpb.NewNumberValue(1)}}),
				structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"id": structpb.NewNumberValue(2)}}),
			}}),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: `[{"id":1},{"id":2}]`,
			}},
		},
		{
			name:     "timestamp value",
			colType:  &sppb.Type{Code: sppb.TypeCode_TIMESTAMP},
			value:    structpb.NewStringValue("2024-01-15T10:30:45.123456Z"),
			expected: timestampRowValue("2024-01-15T10:30:45.123456Z", 6),
		},
		{
			name:     "timestamp with nanosecond precision",
			colType:  &sppb.Type{Code: sppb.TypeCode_TIMESTAMP},
			value:    structpb.NewStringValue("2024-01-15T10:30:45.123456789Z"),
			expected: timestampRowValue("2024-01-15T10:30:45.123456789Z", 9),
		},
		{
			name:     "timestamp without fractional seconds",
			colType:  &sppb.Type{Code: sppb.TypeCode_TIMESTAMP},
			value:    structpb.NewStringValue("2024-01-15T10:30:45Z"),
			expected: timestampRowValue("2024-01-15T10:30:45Z", 6),
		},
		{
			name:    "timestamp invalid format falls back to string",
			colType: &sppb.Type{Code: sppb.TypeCode_TIMESTAMP},
			value:   structpb.NewStringValue("not-a-timestamp"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "not-a-timestamp",
			}},
		},
		{
			name:    "bytes value",
			colType: &sppb.Type{Code: sppb.TypeCode_BYTES},
			value:   structpb.NewStringValue("SGVsbG8gV29ybGQ="), // "Hello World" in base64
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{
				BytesValue: []byte("Hello World"),
			}},
		},
		{
			name:    "bytes invalid base64 falls back to string",
			colType: &sppb.Type{Code: sppb.TypeCode_BYTES},
			value:   structpb.NewStringValue("not-valid-base64!"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "not-valid-base64!",
			}},
		},
		{
			name:    "date value",
			colType: &sppb.Type{Code: sppb.TypeCode_DATE},
			value:   structpb.NewStringValue("2024-01-15"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "2024-01-15",
			}},
		},
		{
			name:    "json value",
			colType: &sppb.Type{Code: sppb.TypeCode_JSON},
			value:   structpb.NewStringValue(`{"name":"Alice","age":30}`),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: `{"name":"Alice","age":30}`,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := convertSpannerValue(tt.colType, tt.value)
			a.Equal(tt.expected, result)
		})
	}
}
