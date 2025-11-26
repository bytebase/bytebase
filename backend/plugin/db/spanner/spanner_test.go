package spanner

import (
	"math"
	"testing"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

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
			name:    "int64 negative large value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(-9223372036854775808), // -2^63
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: math.MinInt64,
			}},
		},
		{
			name:    "int64 max value",
			colType: &sppb.Type{Code: sppb.TypeCode_INT64},
			value:   structpb.NewNumberValue(9223372036854775807), // 2^63-1
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: math.MaxInt64,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := convertSpannerValue(tt.colType, tt.value)
			a.Equal(tt.expected, result)
		})
	}
}
