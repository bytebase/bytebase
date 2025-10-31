package pg

import (
	"testing"
)

func TestAreTypesEquivalent(t *testing.T) {
	tests := []struct {
		typeA    string
		typeB    string
		expected bool
	}{
		// Exact matches
		{"int", "int", true},
		{"INTEGER", "INTEGER", true},
		{"json", "json", true},

		// Integer family
		{"int", "integer", true},
		{"INT", "INTEGER", true}, // Case insensitive
		{"int", "int4", true},
		{"integer", "int4", true},
		{"INT", "INT4", true},

		// Bigint family
		{"bigint", "int8", true},
		{"BIGINT", "INT8", true},

		// Smallint family
		{"smallint", "int2", true},
		{"SMALLINT", "INT2", true},

		// Serial types - normalized to their base types
		{"serial", "integer", true},
		{"SERIAL", "INTEGER", true},
		{"serial", "int", true},
		{"bigserial", "bigint", true},
		{"BIGSERIAL", "BIGINT", true},
		{"smallserial", "smallint", true},

		// Real family
		{"real", "float4", true},
		{"REAL", "FLOAT4", true},

		// Double precision family
		{"double precision", "float8", true},
		{"DOUBLE PRECISION", "FLOAT8", true},

		// Boolean family
		{"boolean", "bool", true},
		{"BOOLEAN", "BOOL", true},

		// VARCHAR / CHARACTER VARYING
		{"varchar", "character varying", true},
		{"VARCHAR", "CHARACTER VARYING", true},
		{"varchar(20)", "character varying(20)", true},
		{"VARCHAR(20)", "CHARACTER VARYING(20)", true},

		// Character family
		{"character(10)", "char(10)", true},
		{"CHAR(10)", "CHARACTER(10)", true},

		// Different types should not be equivalent
		{"int", "bigint", false},
		{"integer", "smallint", false},
		{"text", "varchar", false},
		{"json", "jsonb", false},

		// Different parameters should not be equivalent
		{"varchar(20)", "varchar(30)", false},
		{"char(10)", "char(20)", false},

		// Types not in same family
		{"int", "text", false},
		{"boolean", "int", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeA+"_vs_"+tt.typeB, func(t *testing.T) {
			result := areTypesEquivalent(tt.typeA, tt.typeB)
			if result != tt.expected {
				t.Errorf("areTypesEquivalent(%q, %q) = %v, want %v", tt.typeA, tt.typeB, result, tt.expected)
			}

			// Test symmetry: areTypesEquivalent should be commutative
			resultReversed := areTypesEquivalent(tt.typeB, tt.typeA)
			if resultReversed != tt.expected {
				t.Errorf("areTypesEquivalent(%q, %q) = %v, want %v (symmetry test)", tt.typeB, tt.typeA, resultReversed, tt.expected)
			}
		})
	}
}

func TestNormalizePostgreSQLType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Serial types
		{"SERIAL", "integer"},
		{"serial", "integer"},
		{"BIGSERIAL", "bigint"},
		{"bigserial", "bigint"},
		{"SMALLSERIAL", "smallint"},
		{"smallserial", "smallint"},

		// VARCHAR variants
		{"VARCHAR", "character varying"},
		{"varchar", "character varying"},
		{"VARCHAR(20)", "character varying(20)"},
		{"varchar(100)", "character varying(100)"},

		// Types that don't need normalization
		{"integer", "integer"},
		{"INT", "int"},
		{"text", "text"},
		{"JSON", "json"},
		{"jsonb", "jsonb"},

		// Whitespace handling
		{"  INTEGER  ", "integer"},
		{" VARCHAR ( 20 ) ", "character varying(20)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePostgreSQLType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePostgreSQLType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsTypeInList(t *testing.T) {
	tests := []struct {
		columnType string
		typeList   []string
		expected   bool
	}{
		// Exact match
		{"int", []string{"int", "text"}, true},
		{"json", []string{"json", "jsonb"}, true},

		// Equivalent type match
		{"integer", []string{"int", "text"}, true},
		{"INT4", []string{"int", "text"}, true},
		{"serial", []string{"int", "text"}, true},

		// VARCHAR equivalence
		{"varchar", []string{"character varying", "text"}, true},
		{"varchar(20)", []string{"character varying", "text"}, false}, // Different parameters

		// No match
		{"bigint", []string{"int", "text"}, false},
		{"jsonb", []string{"json", "text"}, false},

		// Empty list
		{"int", []string{}, false},

		// Case insensitive
		{"INTEGER", []string{"INT", "TEXT"}, true},
		{"INT", []string{"integer", "text"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.columnType, func(t *testing.T) {
			result := isTypeInList(tt.columnType, tt.typeList)
			if result != tt.expected {
				t.Errorf("isTypeInList(%q, %v) = %v, want %v", tt.columnType, tt.typeList, result, tt.expected)
			}
		})
	}
}
