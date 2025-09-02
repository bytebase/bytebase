package ast

import (
	"testing"
)

func TestPostgreSQLExpressionComparer_CompareExpressions(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	tests := []struct {
		name     string
		expr1    string
		expr2    string
		expected bool
		error    bool
	}{
		// Basic equality tests
		{
			name:     "identical expressions",
			expr1:    "column_name",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "case insensitive identifiers",
			expr1:    "Column_Name",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "different identifiers",
			expr1:    "column1",
			expr2:    "column2",
			expected: false,
		},

		// Schema prefix tests
		{
			name:     "public schema ignored",
			expr1:    "public.table.column",
			expr2:    "table.column",
			expected: true,
		},
		{
			name:     "both have public schema",
			expr1:    "public.table.column",
			expr2:    "public.table.column",
			expected: true,
		},
		{
			name:     "non-public schema preserved",
			expr1:    "schema1.table.column",
			expr2:    "schema2.table.column",
			expected: false,
		},

		// Operator normalization tests
		{
			name:     "inequality operators",
			expr1:    "a != b",
			expr2:    "a <> b",
			expected: true,
		},
		{
			name:     "logical operators",
			expr1:    "a && b",
			expr2:    "a AND b",
			expected: true,
		},

		// Commutative operator tests
		{
			name:     "commutative equality",
			expr1:    "a = b",
			expr2:    "b = a",
			expected: true,
		},
		{
			name:     "commutative addition",
			expr1:    "a + b",
			expr2:    "b + a",
			expected: true,
		},
		{
			name:     "non-commutative subtraction",
			expr1:    "a - b",
			expr2:    "b - a",
			expected: false,
		},

		// Parentheses tests
		{
			name:     "irrelevant parentheses ignored",
			expr1:    "(column_name)",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "nested parentheses",
			expr1:    "((column_name))",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "where condition with parentheses",
			expr1:    "text_col IS NOT NULL",
			expr2:    "(text_col IS NOT NULL)",
			expected: true,
		},
		{
			name:     "complex where condition with parentheses",
			expr1:    "column > 0 AND status = 'active'",
			expr2:    "(column > 0 AND status = 'active')",
			expected: true,
		},

		// Function call tests - basic cases
		{
			name:     "function calls case insensitive",
			expr1:    "UPPER(column)",
			expr2:    "upper(column)",
			expected: true,
		},
		{
			name:     "function with schema",
			expr1:    "public.UPPER(column)",
			expr2:    "UPPER(column)",
			expected: true,
		},
		{
			name:     "function with different arguments",
			expr1:    "UPPER(col1)",
			expr2:    "UPPER(col2)",
			expected: false,
		},

		// Quoted vs unquoted function names - PostgreSQL identifier normalization
		{
			name:     "unquoted LEFT vs quoted lowercase left - EQUAL",
			expr1:    "LEFT(text_col, 50)",
			expr2:    "\"left\"(text_col, 50)",
			expected: true,
		},
		{
			name:     "unquoted left vs quoted lowercase left - EQUAL",
			expr1:    "left(text_col, 50)",
			expr2:    "\"left\"(text_col, 50)",
			expected: true,
		},
		{
			name:     "unquoted LEFT vs quoted uppercase LEFT - NOT EQUAL",
			expr1:    "LEFT(text_col, 50)",
			expr2:    "\"LEFT\"(text_col, 50)",
			expected: false,
		},
		{
			name:     "unquoted left vs quoted uppercase LEFT - NOT EQUAL",
			expr1:    "left(text_col, 50)",
			expr2:    "\"LEFT\"(text_col, 50)",
			expected: false,
		},
		{
			name:     "unquoted LEFT vs quoted mixed case Left - NOT EQUAL",
			expr1:    "LEFT(text_col, 50)",
			expr2:    "\"Left\"(text_col, 50)",
			expected: false,
		},
		{
			name:     "quoted function with complex expression",
			expr1:    "SUBSTRING(column FROM 1 FOR 10)",
			expr2:    "\"substring\"(column FROM 1 FOR 10)",
			expected: true,
		},

		// Additional identifier normalization tests
		{
			name:     "unquoted Column_Name vs quoted lowercase column_name - EQUAL",
			expr1:    "Column_Name",
			expr2:    "\"column_name\"",
			expected: true,
		},
		{
			name:     "unquoted Column_Name vs quoted original Column_Name - NOT EQUAL",
			expr1:    "Column_Name",
			expr2:    "\"Column_Name\"",
			expected: false,
		},

		// Complex expression tests
		{
			name:     "complex equality expression",
			expr1:    "table1.col1 = table2.col2 AND status = 'active'",
			expr2:    "table1.col1 = table2.col2 AND status = 'active'",
			expected: true,
		},
		{
			name:     "complex expression with whitespace differences",
			expr1:    "table1.col1=table2.col2 AND status='active'",
			expr2:    "table1.col1 = table2.col2 AND status = 'active'",
			expected: true,
		},

		// PostgreSQL type cast tests
		{
			name:     "JSONB implicit cast",
			expr1:    "'{}'",
			expr2:    "'{}'::jsonb",
			expected: true,
		},
		{
			name:     "JSONB explicit cast both ways",
			expr1:    "'{\"key\": \"value\"}'::jsonb",
			expr2:    "'{\"key\": \"value\"}'::jsonb",
			expected: true,
		},
		{
			name:     "Array type cast case insensitive",
			expr1:    "ARRAY[]::TEXT[]",
			expr2:    "ARRAY[]::text[]",
			expected: true,
		},
		{
			name:     "BIT literal formats",
			expr1:    "B'1010'",
			expr2:    "'1010'::\"bit\"",
			expected: true,
		},
		{
			name:     "BIT literal formats reverse",
			expr1:    "'0000'::\"bit\"",
			expr2:    "B'0000'",
			expected: true,
		},
		{
			name:     "VARBIT schema qualification",
			expr1:    "varbit(16)",
			expr2:    "public.varbit(16)",
			expected: true,
		},
		{
			name:     "BIT schema qualification",
			expr1:    "bit(32)",
			expr2:    "public.bit(32)",
			expected: true,
		},

		// Literal tests
		{
			name:     "string literals",
			expr1:    "'hello'",
			expr2:    "'hello'",
			expected: true,
		},
		{
			name:     "different string literals",
			expr1:    "'hello'",
			expr2:    "'world'",
			expected: false,
		},
		{
			name:     "numeric literals",
			expr1:    "123",
			expr2:    "123",
			expected: true,
		},
		{
			name:     "boolean literals",
			expr1:    "TRUE",
			expr2:    "true",
			expected: true,
		},

		// Empty and null tests
		{
			name:     "empty expressions",
			expr1:    "",
			expr2:    "",
			expected: true,
		},
		{
			name:     "empty vs non-empty",
			expr1:    "",
			expr2:    "column",
			expected: false,
		},
		{
			name:     "whitespace only expressions",
			expr1:    "   ",
			expr2:    "  ",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := comparer.CompareExpressions(test.expr1, test.expr2)

			if test.error {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != test.expected {
				t.Errorf("Expected %v, got %v for comparing '%s' with '%s'",
					test.expected, result, test.expr1, test.expr2)
			}
		})
	}
}

func TestPostgreSQLExpressionComparer_CompareExpressionLists(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	tests := []struct {
		name     string
		exprs1   []string
		exprs2   []string
		expected bool
	}{
		{
			name:     "identical lists",
			exprs1:   []string{"col1", "col2", "col3"},
			exprs2:   []string{"col1", "col2", "col3"},
			expected: true,
		},
		{
			name:     "case differences",
			exprs1:   []string{"COL1", "Col2", "col3"},
			exprs2:   []string{"col1", "col2", "col3"},
			expected: true,
		},
		{
			name:     "schema prefix differences",
			exprs1:   []string{"public.col1", "col2"},
			exprs2:   []string{"col1", "col2"},
			expected: true,
		},
		{
			name:     "different lengths",
			exprs1:   []string{"col1", "col2"},
			exprs2:   []string{"col1", "col2", "col3"},
			expected: false,
		},
		{
			name:     "different expressions",
			exprs1:   []string{"col1", "col2"},
			exprs2:   []string{"col1", "col3"},
			expected: false,
		},
		{
			name:     "empty lists",
			exprs1:   []string{},
			exprs2:   []string{},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := comparer.CompareExpressionLists(test.exprs1, test.exprs2)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != test.expected {
				t.Errorf("Expected %v, got %v for comparing %v with %v",
					test.expected, result, test.exprs1, test.exprs2)
			}
		})
	}
}

func TestPostgreSQLExpressionComparer_CompareExpressionListsUnordered(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	tests := []struct {
		name     string
		exprs1   []string
		exprs2   []string
		expected bool
	}{
		{
			name:     "same order",
			exprs1:   []string{"col1", "col2", "col3"},
			exprs2:   []string{"col1", "col2", "col3"},
			expected: true,
		},
		{
			name:     "different order",
			exprs1:   []string{"col1", "col2", "col3"},
			exprs2:   []string{"col3", "col1", "col2"},
			expected: true,
		},
		{
			name:     "case and order differences",
			exprs1:   []string{"COL1", "Col2", "col3"},
			exprs2:   []string{"col3", "col1", "col2"},
			expected: true,
		},
		{
			name:     "different expressions",
			exprs1:   []string{"col1", "col2", "col3"},
			exprs2:   []string{"col1", "col2", "col4"},
			expected: false,
		},
		{
			name:     "duplicate handling",
			exprs1:   []string{"col1", "col1", "col2"},
			exprs2:   []string{"col2", "col1", "col1"},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := comparer.CompareExpressionListsUnordered(test.exprs1, test.exprs2)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != test.expected {
				t.Errorf("Expected %v, got %v for comparing %v with %v (unordered)",
					test.expected, result, test.exprs1, test.exprs2)
			}
		})
	}
}

func TestPostgreSQLExpressionComparer_ValidateExpression(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	tests := []struct {
		name        string
		expr        string
		expectError bool
	}{
		{
			name:        "valid simple identifier",
			expr:        "column_name",
			expectError: false,
		},
		{
			name:        "valid qualified identifier",
			expr:        "table.column",
			expectError: false,
		},
		{
			name:        "valid function call",
			expr:        "UPPER(column)",
			expectError: false,
		},
		{
			name:        "valid binary operation",
			expr:        "a = b",
			expectError: false,
		},
		{
			name:        "valid complex expression",
			expr:        "table1.col1 = table2.col2 AND status = 'active'",
			expectError: false,
		},
		{
			name:        "empty expression",
			expr:        "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			expr:        "   ",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := comparer.ValidateExpression(test.expr)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for expression: %s", test.expr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for expression '%s': %v", test.expr, err)
				}
			}
		})
	}
}

func TestPostgreSQLExpressionComparer_GetExpressionComplexity(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	tests := []struct {
		name     string
		expr     string
		minScore int // minimum expected complexity score
	}{
		{
			name:     "simple identifier",
			expr:     "column",
			minScore: 1,
		},
		{
			name:     "qualified identifier",
			expr:     "table.column",
			minScore: 2,
		},
		{
			name:     "binary operation",
			expr:     "a = b",
			minScore: 4, // binary op + 2 identifiers
		},
		{
			name:     "function call",
			expr:     "UPPER(column)",
			minScore: 5, // function + identifier
		},
		{
			name:     "complex expression",
			expr:     "table1.col1 = table2.col2 AND status = 'active'",
			minScore: 10, // multiple operations and identifiers
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score, err := comparer.GetExpressionComplexity(test.expr)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if score < test.minScore {
				t.Errorf("Expected complexity >= %d, got %d for expression: %s",
					test.minScore, score, test.expr)
			}
		})
	}
}

func TestCompareExpressionsSemantically(t *testing.T) {
	tests := []struct {
		name     string
		expr1    string
		expr2    string
		expected bool
	}{
		{
			name:     "basic comparison",
			expr1:    "column_name",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "case insensitive",
			expr1:    "Column_Name",
			expr2:    "column_name",
			expected: true,
		},
		{
			name:     "schema prefix ignored",
			expr1:    "public.table.column",
			expr2:    "table.column",
			expected: true,
		},
		{
			name:     "different expressions",
			expr1:    "col1",
			expr2:    "col2",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CompareExpressionsSemantically(test.expr1, test.expr2)

			if result != test.expected {
				t.Errorf("Expected %v, got %v for comparing '%s' with '%s'",
					test.expected, result, test.expr1, test.expr2)
			}
		})
	}
}

func TestPostgreSQLExpressionComparer_WithConfig(t *testing.T) {
	// Test case-sensitive configuration
	comparer := NewPostgreSQLExpressionComparer().WithConfig(true, true, true, true)

	result, err := comparer.CompareExpressions("Column_Name", "column_name")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// With case sensitivity enabled, these should be different
	if result {
		t.Error("Expected case-sensitive comparison to return false")
	}

	// Test schema prefix preservation
	comparer = NewPostgreSQLExpressionComparer().WithConfig(false, true, true, false)

	result, err = comparer.CompareExpressions("public.table.column", "table.column")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// With schema prefix preservation, these should be different
	if result {
		t.Error("Expected schema-sensitive comparison to return false")
	}
}

func BenchmarkCompareExpressionsSemantically(b *testing.B) {
	expr1 := "public.table1.column1 = table2.column2 AND status = 'active'"
	expr2 := "table1.column1 = table2.column2 AND status = 'active'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompareExpressionsSemantically(expr1, expr2)
	}
}
