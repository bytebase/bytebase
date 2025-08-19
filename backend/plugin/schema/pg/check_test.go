package pg

import (
	"testing"
)

func TestCheckColumnType(t *testing.T) {
	tests := []struct {
		name     string
		colType  string
		expected bool
	}{
		// Basic integer types
		{"smallint", "SMALLINT", true},
		{"integer", "INTEGER", true},
		{"bigint", "BIGINT", true},
		{"int", "INT", true},
		{"int2", "INT2", true},
		{"int4", "INT4", true},
		{"int8", "INT8", true},

		// Decimal/numeric types
		{"decimal", "DECIMAL", true},
		{"numeric", "NUMERIC", true},
		{"decimal_with_precision", "DECIMAL(10,2)", true},
		{"numeric_with_precision", "NUMERIC(15,5)", true},

		// Floating point types
		{"real", "REAL", true},
		{"double_precision", "DOUBLE PRECISION", true},
		{"float", "FLOAT", true},
		{"float4", "FLOAT4", true},
		{"float8", "FLOAT8", true},

		// Serial types
		{"serial", "SERIAL", true},
		{"bigserial", "BIGSERIAL", true},
		{"smallserial", "SMALLSERIAL", true},

		// Character types
		{"char", "CHAR", true},
		{"varchar", "VARCHAR", true},
		{"text", "TEXT", true},
		{"char_with_length", "CHAR(10)", true},
		{"varchar_with_length", "VARCHAR(255)", true},
		{"character", "CHARACTER", true},
		{"character_varying", "CHARACTER VARYING", true},

		// Boolean type
		{"boolean", "BOOLEAN", true},
		{"bool", "BOOL", true},

		// Date/time types
		{"date", "DATE", true},
		{"time", "TIME", true},
		{"timestamp", "TIMESTAMP", true},
		{"timestamptz", "TIMESTAMPTZ", true},
		{"timestamp_with_timezone", "TIMESTAMP WITH TIME ZONE", true},
		{"time_with_timezone", "TIME WITH TIME ZONE", true},
		{"interval", "INTERVAL", true},

		// Binary data types
		{"bytea", "BYTEA", true},

		// Network address types
		{"cidr", "CIDR", true},
		{"inet", "INET", true},
		{"macaddr", "MACADDR", true},
		{"macaddr8", "MACADDR8", true},

		// UUID type
		{"uuid", "UUID", true},

		// JSON types
		{"json", "JSON", true},
		{"jsonb", "JSONB", true},

		// Array types
		{"integer_array", "INTEGER[]", true},
		{"text_array", "TEXT[]", true},
		{"varchar_array", "VARCHAR(50)[]", true},

		// Range types
		{"int4range", "INT4RANGE", true},
		{"int8range", "INT8RANGE", true},
		{"numrange", "NUMRANGE", true},
		{"tsrange", "TSRANGE", true},
		{"tstzrange", "TSTZRANGE", true},
		{"daterange", "DATERANGE", true},

		// Geometric types
		{"point", "POINT", true},
		{"line", "LINE", true},
		{"lseg", "LSEG", true},
		{"box", "BOX", true},
		{"path", "PATH", true},
		{"polygon", "POLYGON", true},
		{"circle", "CIRCLE", true},

		// Full-text search types
		{"tsvector", "TSVECTOR", true},
		{"tsquery", "TSQUERY", true},

		// Other types
		{"money", "MONEY", true},
		{"bit", "BIT", true},
		{"bit_varying", "BIT VARYING", true},
		{"varbit", "VARBIT", true},
		{"xml", "XML", true},

		// Custom domain types (should be valid syntax)
		{"user_defined", "my_custom_type", true},

		// Unknown types (pg_query_go treats these as potentially valid user-defined types)
		{"unknown_type", "INVALID_TYPE_XXXXX", true},

		// Actually invalid syntax that should fail
		{"empty_string", "", false},
		{"just_parentheses", "()", false},
		{"malformed_precision", "DECIMAL(,)", false},
		{"unclosed_parenthesis", "VARCHAR(255", false},
		{"invalid_array", "TEXT[[[", false},

		// Case insensitive tests
		{"lowercase_text", "text", true},
		{"mixed_case", "TeXt", true},
		{"uppercase_varchar", "VARCHAR(100)", true},

		// Complex precision specifications
		{"time_with_precision", "TIME(6)", true},
		{"timestamp_with_precision", "TIMESTAMP(3)", true},
		{"interval_fields", "INTERVAL YEAR TO MONTH", true},
		{"interval_precision", "INTERVAL(4)", true},

		// Bit types with length
		{"bit_with_length", "BIT(8)", true},
		{"varbit_with_length", "VARBIT(64)", true},

		// Multi-dimensional arrays
		{"multidim_array", "INTEGER[][]", true},
		{"three_dim_array", "TEXT[][][]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkColumnType(tt.colType)
			if result != tt.expected {
				t.Errorf("checkColumnType(%q) = %v, expected %v", tt.colType, result, tt.expected)
			}
		})
	}
}
