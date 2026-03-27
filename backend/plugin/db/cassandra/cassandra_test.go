package cassandra

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestConvertRowValueComplexTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *v1pb.RowValue
	}{
		{
			name:  "nil value",
			value: nil,
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "null",
			}},
		},
		{
			name:  "list of integers",
			value: []any{1, 2, 3},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "[1,2,3]",
			}},
		},
		{
			name:  "map of string to int",
			value: map[string]any{"a": 1, "b": 2},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: `{"a":1,"b":2}`,
			}},
		},
		{
			name:  "nested list",
			value: []any{[]any{1, 2}, []any{3, 4}},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: "[[1,2],[3,4]]",
			}},
		},
		{
			name:  "array of maps",
			value: []any{map[string]any{"id": 1}, map[string]any{"id": 2}},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: `[{"id":1},{"id":2}]`,
			}},
		},
		{
			name:     "empty list",
			value:    []any{},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "[]"}},
		},
		{
			name:     "empty map",
			value:    map[string]any{},
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "{}"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := convertRowValue(tt.value)
			a.Equal(tt.expected, result)
		})
	}
}

func TestConvertRowValuePrimitives(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *v1pb.RowValue
	}{
		{
			name:     "string pointer",
			value:    new("hello"),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "hello"}},
		},
		{
			name: "nil string pointer",
			value: func() *string {
				return nil
			}(),
			expected: util.NullRowValue,
		},
		{
			name:     "int64 pointer",
			value:    new(int64(42)),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 42}},
		},
		{
			name:     "bool pointer true",
			value:    new(true),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}},
		},
		{
			name:     "bool pointer false",
			value:    new(false),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: false}},
		},
		{
			name:     "float64 pointer",
			value:    new(3.14159),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 3.14159}},
		},
		{
			name:     "bytes pointer",
			value:    new([]byte("hello")),
			expected: &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte("hello")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := convertRowValue(tt.value)
			a.Equal(tt.expected, result)
		})
	}
}
