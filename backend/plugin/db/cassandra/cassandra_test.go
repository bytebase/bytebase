package cassandra

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	_ "github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestConvertValueComplexTypes tests whether complex types are converted to ValueValue
// (which causes the protobuf structure display bug) or to proper RowValue types.
func TestConvertValueComplexTypes(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		expectKind  string // "valueValue" means bug exists, "stringValue" means bug fixed
		description string
	}{
		{
			name:        "list of integers",
			value:       []any{1, 2, 3},
			expectKind:  "stringValue",
			description: "Cassandra LIST type becomes []any",
		},
		{
			name:        "map of string to int",
			value:       map[string]any{"a": 1, "b": 2},
			expectKind:  "stringValue",
			description: "Cassandra MAP type becomes map[string]any",
		},
		{
			name:        "nested list",
			value:       []any{[]any{1, 2}, []any{3, 4}},
			expectKind:  "stringValue",
			description: "Nested Cassandra LIST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := convertRowValue(tt.value)

			switch tt.expectKind {
			case "valueValue":
				// Bug exists - using ValueValue
				if result.Kind == nil {
					t.Fatal("result.Kind is nil")
				}
				if _, ok := result.Kind.(*v1pb.RowValue_ValueValue); ok {
					t.Logf("✓ Confirmed bug exists: %s returns ValueValue (protobuf structure will be displayed)", tt.description)
				} else {
					t.Logf("✗ Bug might be fixed already: got %T instead of ValueValue", result.Kind)
				}
			case "stringValue":
				// Bug fixed - using StringValue with JSON
				if result.Kind == nil {
					t.Fatal("result.Kind is nil")
				}
				if sv, ok := result.Kind.(*v1pb.RowValue_StringValue); ok {
					t.Logf("✓ Bug is fixed: %s returns StringValue: %s", tt.description, sv.StringValue)
				} else {
					t.Fatalf("Expected StringValue but got %T", result.Kind)
				}
			default:
				t.Fatalf("Unknown expectKind: %s", tt.expectKind)
			}

			// Always check that it's not nil
			a.NotNil(result)
			a.NotNil(result.Kind)
		})
	}
}
