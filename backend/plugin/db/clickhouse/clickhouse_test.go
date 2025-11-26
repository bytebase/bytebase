package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestConvertValueComplexTypes tests whether ClickHouse TUPLE/ARRAY/MAP types
// are properly converted to JSON strings (no bug) or use ValueValue (bug exists).
func TestConvertValueComplexTypes(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		expectKind  string // "valueValue" means bug exists, "stringValue" means no bug
		description string
	}{
		{
			name: "array type (via *any)",
			value: func() *any {
				var v any = []any{1, 2, 3}
				return &v
			}(),
			expectKind:  "stringValue",
			description: "ClickHouse ARRAY type",
		},
		{
			name: "map type (via *any)",
			value: func() *any {
				var v any = map[string]any{"a": 1, "b": 2}
				return &v
			}(),
			expectKind:  "stringValue",
			description: "ClickHouse MAP type",
		},
		{
			name: "tuple type (via *any)",
			value: func() *any {
				var v any = []any{"Alice", 30}
				return &v
			}(),
			expectKind:  "stringValue",
			description: "ClickHouse TUPLE type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			// ClickHouse convertValue expects columnType but doesn't use it for complex types
			result := convertValue("", nil, tt.value)

			if tt.expectKind == "valueValue" {
				// Bug exists - using ValueValue
				if result.Kind == nil {
					t.Fatalf("result.Kind is nil")
				}
				if _, ok := result.Kind.(*v1pb.RowValue_ValueValue); ok {
					t.Logf("✗ Bug exists: %s returns ValueValue (protobuf structure will be displayed)", tt.description)
				} else {
					t.Logf("✓ Bug does NOT exist: got %T instead of ValueValue", result.Kind)
				}
			} else if tt.expectKind == "stringValue" {
				// Bug fixed - using StringValue with JSON
				if result.Kind == nil {
					t.Fatalf("result.Kind is nil")
				}
				if sv, ok := result.Kind.(*v1pb.RowValue_StringValue); ok {
					t.Logf("✓ Confirmed no bug: %s returns StringValue (JSON): %s", tt.description, sv.StringValue)
				} else {
					t.Fatalf("Expected StringValue but got %T", result.Kind)
				}
			}

			// Always check that it's not nil
			a.NotNil(result)
			a.NotNil(result.Kind)
		})
	}
}
