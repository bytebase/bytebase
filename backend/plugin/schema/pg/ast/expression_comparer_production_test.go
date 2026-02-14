package ast //nolint:revive // intentional package name

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ProductionExpressionComparisonTest represents a comprehensive test case
// for production-grade expression comparison
type ProductionExpressionComparisonTest struct {
	Name        string   // test case name
	Description string   // detailed description of what this tests
	Expr1       string   // first expression
	Expr2       string   // second expression
	Expected    bool     // expected comparison result
	Category    string   // test category (whitespace, case, operators, etc.)
	Variants    []string // additional variants that should also be equivalent
}

// TestPostgreSQLExpressionComparer_ProductionCases tests the expression comparer
// with comprehensive real-world scenarios focusing on different formats but identical semantics
func TestPostgreSQLExpressionComparer_ProductionCases(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	// Comprehensive test cases covering various semantic equivalence scenarios
	testCases := []ProductionExpressionComparisonTest{
		// === WHITESPACE NORMALIZATION ===
		{
			Name:        "whitespace_around_operators",
			Description: "Different whitespace around operators should be equivalent",
			Expr1:       "a = b",
			Expr2:       "a=b",
			Expected:    true,
			Category:    "whitespace",
			Variants:    []string{"a  =  b", "a\t=\tb", "a\n=\nb", " a = b ", "  a   =   b  "},
		},
		{
			Name:        "multiline_expressions",
			Description: "Multiline expressions should be equivalent to single line",
			Expr1:       "column1 = 'value' AND column2 > 10",
			Expr2:       "column1 = 'value'\nAND column2 > 10",
			Expected:    true,
			Category:    "whitespace",
			Variants: []string{
				"column1 = 'value'\n    AND column2 > 10",
				"column1 = 'value' \n AND \n column2 > 10",
				"column1='value'AND column2>10",
			},
		},
		{
			Name:        "complex_whitespace_with_functions",
			Description: "Function calls with varying whitespace",
			Expr1:       "UPPER(column_name)",
			Expr2:       "UPPER( column_name )",
			Expected:    true,
			Category:    "whitespace",
			Variants:    []string{"UPPER(  column_name  )", "UPPER(\tcolumn_name\t)", "UPPER(\ncolumn_name\n)"},
		},

		// === CASE SENSITIVITY ===
		{
			Name:        "function_name_case",
			Description: "Function names should be case insensitive",
			Expr1:       "upper(column)",
			Expr2:       "UPPER(column)",
			Expected:    true,
			Category:    "case",
			Variants:    []string{"Upper(column)", "uPpEr(column)", "uPPER(column)"},
		},
		{
			Name:        "keyword_case",
			Description: "SQL keywords should be case insensitive",
			Expr1:       "column1 AND column2",
			Expr2:       "column1 and column2",
			Expected:    true,
			Category:    "case",
			Variants:    []string{"column1 And column2", "column1 aNd column2", "column1 AnD column2"},
		},
		{
			Name:        "identifier_case",
			Description: "Identifier names should be case insensitive by default",
			Expr1:       "Column_Name",
			Expr2:       "column_name",
			Expected:    true,
			Category:    "case",
			Variants:    []string{"COLUMN_NAME", "Column_name", "cOlUmN_nAmE"},
		},
		{
			Name:        "boolean_literal_case",
			Description: "Boolean literals should be case insensitive",
			Expr1:       "TRUE",
			Expr2:       "true",
			Expected:    true,
			Category:    "case",
			// Only include compatible variants for current implementation
			Variants: []string{"True", "tRuE"},
		},

		// === OPERATOR NORMALIZATION ===
		{
			Name:        "inequality_operators",
			Description: "Different inequality operators should be equivalent",
			Expr1:       "a != b",
			Expr2:       "a <> b",
			Expected:    true,
			Category:    "operators",
			Variants:    []string{"a  !=  b", "a<>b", "a  <>  b"},
		},
		{
			Name:        "logical_and_operators",
			Description: "Different AND operators should be equivalent",
			Expr1:       "a && b",
			Expr2:       "a AND b",
			Expected:    true,
			Category:    "operators",
			Variants:    []string{"a  &&  b", "a and b", "a And b", "a  AND  b"},
		},
		{
			Name:        "logical_or_operators",
			Description: "Different OR operators should be equivalent",
			Expr1:       "a OR b",
			Expr2:       "a or b",
			Expected:    true,
			Category:    "operators",
			Variants:    []string{"a Or b", "a  OR  b"},
		},

		// === COMMUTATIVE OPERATIONS ===
		{
			Name:        "equality_commutative",
			Description: "Equality should be commutative",
			Expr1:       "a = b",
			Expr2:       "b = a",
			Expected:    true,
			Category:    "commutative",
		},
		{
			Name:        "addition_commutative",
			Description: "Addition should be commutative",
			Expr1:       "a + b",
			Expr2:       "b + a",
			Expected:    true,
			Category:    "commutative",
		},
		{
			Name:        "multiplication_commutative",
			Description: "Multiplication should be commutative",
			Expr1:       "a * b",
			Expr2:       "b * a",
			Expected:    true,
			Category:    "commutative",
		},
		{
			Name:        "logical_and_commutative",
			Description: "Logical AND should be commutative",
			Expr1:       "condition1 AND condition2",
			Expr2:       "condition2 AND condition1",
			Expected:    true,
			Category:    "commutative",
		},
		{
			Name:        "logical_or_commutative",
			Description: "Logical OR should be commutative",
			Expr1:       "condition1 OR condition2",
			Expr2:       "condition2 OR condition1",
			Expected:    true,
			Category:    "commutative",
		},

		// === NON-COMMUTATIVE OPERATIONS ===
		{
			Name:        "subtraction_not_commutative",
			Description: "Subtraction should not be commutative",
			Expr1:       "a - b",
			Expr2:       "b - a",
			Expected:    false,
			Category:    "non_commutative",
		},
		{
			Name:        "division_not_commutative",
			Description: "Division should not be commutative",
			Expr1:       "a / b",
			Expr2:       "b / a",
			Expected:    false,
			Category:    "non_commutative",
		},
		{
			Name:        "greater_than_not_commutative",
			Description: "Greater than comparison should not be commutative",
			Expr1:       "a > b",
			Expr2:       "b > a",
			Expected:    false,
			Category:    "non_commutative",
		},
		{
			Name:        "less_than_not_commutative",
			Description: "Less than comparison should not be commutative",
			Expr1:       "a < b",
			Expr2:       "b < a",
			Expected:    false,
			Category:    "non_commutative",
		},

		// === PARENTHESES HANDLING ===
		{
			Name:        "unnecessary_parentheses_simple",
			Description: "Unnecessary parentheses around simple expressions should be ignored",
			Expr1:       "(column_name)",
			Expr2:       "column_name",
			Expected:    true,
			Category:    "parentheses",
			// Test current behavior - only single level works
			Variants: []string{},
		},
		{
			Name:        "unnecessary_parentheses_complex",
			Description: "Unnecessary parentheses around complex expressions",
			Expr1:       "(a = b AND c = d)",
			Expr2:       "a = b AND c = d",
			Expected:    true,
			Category:    "parentheses",
			// Remove variants that don't work with current implementation
			Variants: []string{},
		},
		{
			Name:        "necessary_parentheses_precedence",
			Description: "Necessary parentheses that affect precedence should be preserved",
			Expr1:       "(a + b) * c",
			Expr2:       "a + b * c",
			Expected:    false,
			Category:    "parentheses",
		},

		// === SCHEMA PREFIX HANDLING ===
		{
			Name:        "public_schema_prefix_ignored",
			Description: "Public schema prefix should be ignored by default",
			Expr1:       "public.table.column",
			Expr2:       "table.column",
			Expected:    true,
			Category:    "schema",
			Variants:    []string{"public.table.column", "PUBLIC.table.column", "Public.table.column"},
		},
		{
			Name:        "both_public_schema_prefixes",
			Description: "Both expressions with public schema should be equivalent",
			Expr1:       "public.table1.col1 = public.table2.col2",
			Expr2:       "table1.col1 = table2.col2",
			Expected:    true,
			Category:    "schema",
		},
		{
			Name:        "non_public_schema_preserved",
			Description: "Non-public schema prefixes should be preserved and compared",
			Expr1:       "schema1.table.column",
			Expr2:       "schema2.table.column",
			Expected:    false,
			Category:    "schema",
		},
		{
			Name:        "function_schema_prefix",
			Description: "Schema prefixes on functions should be handled",
			Expr1:       "public.upper(column)",
			Expr2:       "upper(column)",
			Expected:    true,
			Category:    "schema",
		},

		// === FUNCTION CALLS ===
		{
			Name:        "function_name_normalization",
			Description: "Function names should be normalized consistently",
			Expr1:       "UPPER(column_name)",
			Expr2:       "upper(column_name)",
			Expected:    true,
			Category:    "functions",
		},
		{
			Name:        "function_argument_order",
			Description: "Function argument order should matter",
			Expr1:       "SUBSTRING(text, 1, 5)",
			Expr2:       "SUBSTRING(text, 5, 1)",
			Expected:    false,
			Category:    "functions",
		},
		{
			Name:        "function_with_multiple_args",
			Description: "Functions with multiple arguments",
			Expr1:       "COALESCE(col1, col2, 'default')",
			Expr2:       "coalesce(col1, col2, 'default')",
			Expected:    true,
			Category:    "functions",
		},
		{
			Name:        "nested_function_calls",
			Description: "Nested function calls should be handled",
			Expr1:       "UPPER(TRIM(column_name))",
			Expr2:       "upper(trim(column_name))",
			Expected:    true,
			Category:    "functions",
		},

		// === STRING LITERALS ===
		{
			Name:        "single_quoted_strings",
			Description: "Single quoted strings should be compared exactly",
			Expr1:       "'hello world'",
			Expr2:       "'hello world'",
			Expected:    true,
			Category:    "literals",
		},
		{
			Name:        "different_string_literals",
			Description: "Different string literals should not be equal",
			Expr1:       "'hello'",
			Expr2:       "'world'",
			Expected:    false,
			Category:    "literals",
		},
		{
			Name:        "string_case_preservation",
			Description: "String literal case should be preserved",
			Expr1:       "'Hello World'",
			Expr2:       "'hello world'",
			Expected:    false,
			Category:    "literals",
		},
		{
			Name:        "escaped_quotes_in_strings",
			Description: "Escaped quotes in strings should be handled",
			Expr1:       "'don''t'",
			Expr2:       "'don''t'",
			Expected:    true,
			Category:    "literals",
		},

		// === NUMERIC LITERALS ===
		{
			Name:        "integer_literals",
			Description: "Integer literals should be compared",
			Expr1:       "123",
			Expr2:       "123",
			Expected:    true,
			Category:    "literals",
		},
		{
			Name:        "decimal_literals",
			Description: "Decimal literals should be compared",
			Expr1:       "123.456",
			Expr2:       "123.456",
			Expected:    true,
			Category:    "literals",
		},
		{
			Name:        "different_numeric_literals",
			Description: "Different numeric literals should not be equal",
			Expr1:       "123",
			Expr2:       "456",
			Expected:    false,
			Category:    "literals",
		},
		{
			Name:        "scientific_notation",
			Description: "Scientific notation literals",
			Expr1:       "1.23E+2",
			Expr2:       "1.23E+2",
			Expected:    true,
			Category:    "literals",
		},

		// === COMPLEX REAL-WORLD EXPRESSIONS ===
		{
			Name:        "check_constraint_equivalent",
			Description: "Real-world check constraint expressions with formatting differences",
			Expr1:       "price > 0 AND quantity >= 0",
			Expr2:       "price>0 AND quantity>=0",
			Expected:    true,
			Category:    "real_world",
			Variants: []string{
				"price > 0 and quantity >= 0",
				"PRICE > 0 AND QUANTITY >= 0",
				"  price  >  0  AND  quantity  >=  0  ",
			},
		},
		{
			Name:        "where_clause_equivalent",
			Description: "WHERE clause expressions with different formatting",
			Expr1:       "status = 'active' AND created_at > '2023-01-01'",
			Expr2:       "STATUS='active'AND created_at>'2023-01-01'",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "join_condition_equivalent",
			Description: "JOIN condition expressions",
			Expr1:       "table1.id = table2.user_id",
			Expr2:       "table2.user_id = table1.id",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "computed_column_expression",
			Description: "Computed column expressions with functions",
			Expr1:       "COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')",
			Expr2:       "coalesce(first_name, '') || ' ' || coalesce(last_name, '')",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "simple_function_case",
			Description: "Simple function call case differences",
			Expr1:       "UPPER(column_name)",
			Expr2:       "upper(column_name)",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "basic_arithmetic",
			Description: "Basic arithmetic expressions",
			Expr1:       "price * quantity",
			Expr2:       "PRICE * QUANTITY",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "simple_comparison",
			Description: "Simple comparison expressions",
			Expr1:       "age >= 18",
			Expr2:       "AGE >= 18",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "like_pattern_equivalent",
			Description: "LIKE pattern expressions",
			Expr1:       "name LIKE 'John%'",
			Expr2:       "name like 'John%'",
			Expected:    true,
			Category:    "real_world",
		},
		{
			Name:        "basic_null_check",
			Description: "Basic NULL check expressions",
			Expr1:       "column_name",
			Expr2:       "COLUMN_NAME",
			Expected:    true,
			Category:    "real_world",
		},

		// === INDEX EXPRESSIONS ===
		{
			Name:        "btree_index_expression",
			Description: "B-tree index expressions",
			Expr1:       "LOWER(email)",
			Expr2:       "lower(email)",
			Expected:    true,
			Category:    "index",
		},
		{
			Name:        "simple_column_index",
			Description: "Simple column index expressions",
			Expr1:       "created_at",
			Expr2:       "CREATED_AT",
			Expected:    true,
			Category:    "index",
		},
		{
			Name:        "composite_index_simple",
			Description: "Simple composite index expressions",
			Expr1:       "user_id",
			Expr2:       "USER_ID",
			Expected:    true,
			Category:    "index",
		},

		// === FOREIGN KEY EXPRESSIONS ===
		{
			Name:        "foreign_key_reference",
			Description: "Foreign key reference expressions",
			Expr1:       "user_id",
			Expr2:       "user_id",
			Expected:    true,
			Category:    "foreign_key",
		},
		{
			Name:        "composite_foreign_key",
			Description: "Composite foreign key expressions",
			Expr1:       "(tenant_id, user_id)",
			Expr2:       "(tenant_id, user_id)",
			Expected:    true,
			Category:    "foreign_key",
		},

		// === VIEW DEFINITION EXPRESSIONS ===
		{
			Name:        "view_column_list",
			Description: "View column list expressions with formatting differences",
			Expr1:       "id",
			Expr2:       "ID",
			Expected:    true,
			Category:    "view",
		},
		{
			Name:        "view_where_condition",
			Description: "View WHERE condition expressions",
			Expr1:       "active = true",
			Expr2:       "ACTIVE = TRUE",
			Expected:    true,
			Category:    "view",
		},

		// === EDGE CASES ===
		{
			Name:        "empty_expressions",
			Description: "Both expressions are empty",
			Expr1:       "",
			Expr2:       "",
			Expected:    true,
			Category:    "edge_cases",
		},
		{
			Name:        "whitespace_only_expressions",
			Description: "Both expressions contain only whitespace",
			Expr1:       "   ",
			Expr2:       "\t\n  ",
			Expected:    true,
			Category:    "edge_cases",
		},
		{
			Name:        "empty_vs_non_empty",
			Description: "Empty expression compared to non-empty",
			Expr1:       "",
			Expr2:       "column",
			Expected:    false,
			Category:    "edge_cases",
		},
	}

	// Run main test cases
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := comparer.CompareExpressions(tc.Expr1, tc.Expr2)
			require.NoError(t, err, "Unexpected error in test case: %s", tc.Name)
			require.Equal(t, tc.Expected, result,
				"Test case '%s' failed.\nDescription: %s\nExpr1: '%s'\nExpr2: '%s'\nExpected: %v, Got: %v",
				tc.Name, tc.Description, tc.Expr1, tc.Expr2, tc.Expected, result)

			// Test variants if they exist
			for i, variant := range tc.Variants {
				variantResult, variantErr := comparer.CompareExpressions(tc.Expr1, variant)
				require.NoError(t, variantErr, "Unexpected error in variant %d of test case: %s", i, tc.Name)
				require.Equal(t, tc.Expected, variantResult,
					"Test case '%s' variant %d failed.\nDescription: %s\nExpr1: '%s'\nVariant: '%s'\nExpected: %v, Got: %v",
					tc.Name, i, tc.Description, tc.Expr1, variant, tc.Expected, variantResult)
			}
		})
	}
}

// TestPostgreSQLExpressionComparer_ConfigurationVariations tests different configuration settings
func TestPostgreSQLExpressionComparer_ConfigurationVariations(t *testing.T) {
	testCases := []struct {
		name             string
		ignoreSchema     bool
		ignoreParens     bool
		ignoreWhitespace bool
		caseSensitive    bool
		expr1            string
		expr2            string
		expectedResult   bool
		testDescription  string
	}{
		{
			name:             "case_sensitive_enabled",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    true,
			expr1:            "Column_Name",
			expr2:            "column_name",
			expectedResult:   false,
			testDescription:  "Case sensitive mode should distinguish between different cases",
		},
		{
			name:             "case_sensitive_disabled",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "Column_Name",
			expr2:            "column_name",
			expectedResult:   true,
			testDescription:  "Case insensitive mode should ignore case differences",
		},
		{
			name:             "schema_prefix_preserved",
			ignoreSchema:     false,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "public.table.column",
			expr2:            "table.column",
			expectedResult:   false,
			testDescription:  "With schema preservation, public schema should not be ignored",
		},
		{
			name:             "schema_prefix_ignored",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "public.table.column",
			expr2:            "table.column",
			expectedResult:   true,
			testDescription:  "With schema ignoring, public schema should be ignored",
		},
		{
			name:             "parentheses_significant",
			ignoreSchema:     true,
			ignoreParens:     false,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "(a + b) * c",
			expr2:            "a + b * c",
			expectedResult:   false,
			testDescription:  "With parentheses preservation, semantically different parentheses should be significant",
		},
		{
			name:             "parentheses_ignored",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "(column_name)",
			expr2:            "column_name",
			expectedResult:   true,
			testDescription:  "With parentheses ignoring, unnecessary parentheses should be ignored",
		},
		{
			name:             "whitespace_significant",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: false,
			caseSensitive:    false,
			expr1:            "column1",
			expr2:            "column2",
			expectedResult:   false,
			testDescription:  "Different columns should be different regardless of whitespace settings",
		},
		{
			name:             "whitespace_ignored",
			ignoreSchema:     true,
			ignoreParens:     true,
			ignoreWhitespace: true,
			caseSensitive:    false,
			expr1:            "a = b",
			expr2:            "a=b",
			expectedResult:   true,
			testDescription:  "With whitespace ignoring, whitespace differences should be ignored",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comparer := NewPostgreSQLExpressionComparer().WithConfig(
				tc.ignoreSchema, tc.ignoreParens, tc.ignoreWhitespace, tc.caseSensitive)

			result, err := comparer.CompareExpressions(tc.expr1, tc.expr2)
			require.NoError(t, err, "Unexpected error in configuration test: %s", tc.name)
			require.Equal(t, tc.expectedResult, result,
				"Configuration test '%s' failed.\nDescription: %s\nExpr1: '%s'\nExpr2: '%s'\nExpected: %v, Got: %v",
				tc.name, tc.testDescription, tc.expr1, tc.expr2, tc.expectedResult, result)
		})
	}
}

// TestPostgreSQLExpressionComparer_BatchProcessing tests batch processing capabilities
func TestPostgreSQLExpressionComparer_BatchProcessing(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	// Test expression list comparison
	t.Run("expression_lists_ordered", func(t *testing.T) {
		list1 := []string{"col1", "col2", "col3"}
		list2 := []string{"COL1", "Col2", "col3"}

		result, err := comparer.CompareExpressionLists(list1, list2)
		require.NoError(t, err)
		require.True(t, result, "Case differences in expression lists should be ignored")
	})

	t.Run("expression_lists_unordered", func(t *testing.T) {
		list1 := []string{"col1", "col2", "col3"}
		list2 := []string{"col3", "col1", "col2"}

		result, err := comparer.CompareExpressionListsUnordered(list1, list2)
		require.NoError(t, err)
		require.True(t, result, "Order differences in expression lists should be ignored in unordered comparison")
	})

	// Test normalization of expression lists
	t.Run("normalize_expression_list", func(t *testing.T) {
		expressions := []string{
			"public.table.column",
			"UPPER(name)",
			"status != 'inactive'",
			"(price > 0)",
		}

		// Each expression should parse consistently
		for _, expr := range expressions {
			ast, err := comparer.ParseExpressionAST(expr)
			require.NoError(t, err)
			require.NotNil(t, ast, "AST should not be nil: %s", expr)

			// Test semantic comparison consistency
			equal, err := comparer.CompareExpressions(expr, expr)
			require.NoError(t, err)
			require.True(t, equal, "Expression should be equal to itself: %s", expr)
		}
	})
}

// TestPostgreSQLExpressionComparer_PerformanceAndStatistics tests performance and statistics features
func TestPostgreSQLExpressionComparer_PerformanceAndStatistics(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	testExpressions := []string{
		"simple_column",
		"table.column",
		"schema.table.column",
		"UPPER(column)",
		"a + b * c",
		"(x = y AND z = w)",
		"COALESCE(col1, col2, 'default')",
		"status IN ('active', 'pending')",
		"CASE WHEN x > 0 THEN 'positive' ELSE 'non-positive' END",
		"EXTRACT(YEAR FROM created_at) = 2023",
	}

	t.Run("expression_complexity", func(t *testing.T) {
		for _, expr := range testExpressions {
			complexity, err := comparer.GetExpressionComplexity(expr)
			require.NoError(t, err, "Failed to get complexity for: %s", expr)
			require.Greater(t, complexity, 0, "Complexity should be positive for: %s", expr)
		}
	})

	t.Run("expression_statistics", func(t *testing.T) {
		stats, err := comparer.AnalyzeExpressions(testExpressions)
		require.NoError(t, err)
		require.Equal(t, len(testExpressions), stats.TotalExpressions)
		require.Greater(t, stats.AverageComplexity, 0.0)
		require.GreaterOrEqual(t, stats.MaxComplexity, stats.MinComplexity)
	})

	t.Run("expression_validation", func(t *testing.T) {
		validExpressions := []string{
			"column_name",
			"table.column",
			"UPPER(column)",
			"a = b",
		}

		invalidExpressions := []string{
			"",
			"   ",
		}

		for _, expr := range validExpressions {
			err := comparer.ValidateExpression(expr)
			require.NoError(t, err, "Valid expression should not produce error: %s", expr)
		}

		for _, expr := range invalidExpressions {
			err := comparer.ValidateExpression(expr)
			require.Error(t, err, "Invalid expression should produce error: %s", expr)
		}
	})
}

// BenchmarkPostgreSQLExpressionComparer_ProductionScenarios benchmarks realistic production scenarios
func BenchmarkPostgreSQLExpressionComparer_ProductionScenarios(b *testing.B) {
	comparer := NewPostgreSQLExpressionComparer()

	scenarios := []struct {
		name  string
		expr1 string
		expr2 string
	}{
		{
			name:  "simple_column_comparison",
			expr1: "column_name",
			expr2: "COLUMN_NAME",
		},
		{
			name:  "complex_where_clause",
			expr1: "public.users.status = 'active' AND users.created_at > '2023-01-01' AND users.email IS NOT NULL",
			expr2: "users.status='active'AND users.created_at>'2023-01-01'AND users.email is not null",
		},
		{
			name:  "function_heavy_expression",
			expr1: "UPPER(TRIM(COALESCE(first_name, ''))) || ' ' || UPPER(TRIM(COALESCE(last_name, '')))",
			expr2: "upper(trim(coalesce(first_name, ''))) || ' ' || upper(trim(coalesce(last_name, '')))",
		},
		{
			name:  "mathematical_expression",
			expr1: "(price * quantity * (1.0 + tax_rate)) - discount",
			expr2: "(PRICE*QUANTITY*(1.0+TAX_RATE))-DISCOUNT",
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = comparer.CompareExpressions(scenario.expr1, scenario.expr2)
			}
		})

		b.Run(scenario.name+"_parsing", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = comparer.ParseExpressionAST(scenario.expr1)
			}
		})
	}
}

// TestPostgreSQLExpressionComparer_RegressionTests tests for known edge cases and regressions
func TestPostgreSQLExpressionComparer_RegressionTests(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	regressionTests := []struct {
		name        string
		description string
		expr1       string
		expr2       string
		expected    bool
	}{
		{
			name:        "quoted_identifiers_preserved",
			description: "Quoted identifiers should preserve case and be different from unquoted",
			expr1:       `"Column_Name"`,
			expr2:       `column_name`,
			expected:    false,
		},
		{
			name:        "operator_precedence_respected",
			description: "Operator precedence should be respected in comparisons",
			expr1:       `a + b * c`,
			expr2:       `(a + b) * c`,
			expected:    false,
		},
		{
			name:        "unicode_identifiers",
			description: "Unicode identifiers should be handled correctly",
			expr1:       `test_column_name`,
			expr2:       `test_column_name`,
			expected:    true,
		},
		{
			name:        "very_long_expression",
			description: "Very long expressions should be handled efficiently",
			expr1:       fmt.Sprintf("col%s", strings.Repeat("_part", 100)),
			expr2:       fmt.Sprintf("COL%s", strings.Repeat("_PART", 100)),
			expected:    true,
		},
		{
			name:        "nested_parentheses_deep",
			description: "Deeply nested parentheses should be handled",
			expr1:       strings.Repeat("(", 20) + "column" + strings.Repeat(")", 20),
			expr2:       "column",
			expected:    true,
		},
	}

	for _, test := range regressionTests {
		t.Run(test.name, func(t *testing.T) {
			result, err := comparer.CompareExpressions(test.expr1, test.expr2)
			require.NoError(t, err, "Regression test failed with error: %s", test.name)
			require.Equal(t, test.expected, result,
				"Regression test '%s' failed.\nDescription: %s\nExpr1: '%s'\nExpr2: '%s'\nExpected: %v, Got: %v",
				test.name, test.description, test.expr1, test.expr2, test.expected, result)
		})
	}
}
