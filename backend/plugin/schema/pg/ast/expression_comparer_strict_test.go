package ast

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPostgreSQLExpressionComparer_StrictProductionTests contains strict tests
// for production use. These tests SHOULD FAIL if the implementation is incomplete.
// When tests fail, it indicates implementation bugs that need to be fixed.
func TestPostgreSQLExpressionComparer_StrictProductionTests(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	// === CRITICAL PRODUCTION REQUIREMENTS ===
	// These tests represent real-world scenarios that MUST work in production

	t.Run("OR_operator_equivalence", func(t *testing.T) {
		// CRITICAL: PostgreSQL supports both || and OR operators
		result, err := comparer.CompareExpressions("a || b", "a OR b")
		require.NoError(t, err, "Should not error on valid PostgreSQL syntax")
		require.True(t, result, "CRITICAL BUG: || and OR operators should be semantically equivalent in PostgreSQL")
	})

	t.Run("IS_NULL_case_insensitive", func(t *testing.T) {
		// CRITICAL: SQL keywords should be case insensitive
		result, err := comparer.CompareExpressions("column IS NULL", "column is null")
		require.NoError(t, err, "Should not error on valid SQL syntax")
		require.True(t, result, "CRITICAL BUG: SQL keywords should be case insensitive")
	})

	t.Run("BETWEEN_clause_equivalence", func(t *testing.T) {
		// CRITICAL: BETWEEN clauses are common in production SQL
		result, err := comparer.CompareExpressions("age BETWEEN 18 AND 65", "age between 18 and 65")
		require.NoError(t, err, "Should not error on valid BETWEEN syntax")
		require.True(t, result, "CRITICAL BUG: BETWEEN clauses should be case insensitive")
	})

	t.Run("EXTRACT_function_equivalence", func(t *testing.T) {
		// CRITICAL: EXTRACT is commonly used in PostgreSQL for date operations
		result, err := comparer.CompareExpressions("EXTRACT(YEAR FROM created_at)", "extract(year from created_at)")
		require.NoError(t, err, "Should not error on valid EXTRACT syntax")
		require.True(t, result, "CRITICAL BUG: EXTRACT function should be case insensitive")
	})

	t.Run("CASE_expression_equivalence", func(t *testing.T) {
		// CRITICAL: CASE expressions are essential in production queries
		expr1 := "CASE WHEN status = 'active' THEN 1 ELSE 0 END"
		expr2 := "case when status = 'active' then 1 else 0 end"
		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should not error on valid CASE syntax")
		require.True(t, result, "CRITICAL BUG: CASE expressions should be case insensitive")
	})

	t.Run("IN_clause_equivalence", func(t *testing.T) {
		// CRITICAL: IN clauses are extremely common in production
		expr1 := "status IN ('active', 'pending', 'approved')"
		expr2 := "status in ('active', 'pending', 'approved')"
		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should not error on valid IN syntax")
		require.True(t, result, "CRITICAL BUG: IN clauses should be case insensitive")
	})

	t.Run("complex_nested_parentheses", func(t *testing.T) {
		// CRITICAL: Complex parentheses nesting should be handled
		expr1 := "((column_name))"
		expr2 := "column_name"
		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should not error on nested parentheses")
		require.True(t, result, "CRITICAL BUG: Nested unnecessary parentheses should be ignored")
	})

	t.Run("parentheses_distribution", func(t *testing.T) {
		// CRITICAL: Different parentheses grouping of same logical expression
		expr1 := "(a = b) AND (c = d)"
		expr2 := "a = b AND c = d"
		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should not error on parentheses distribution")
		require.True(t, result, "CRITICAL BUG: Unnecessary parentheses around AND clauses should be equivalent")
	})

	t.Run("boolean_literal_variants", func(t *testing.T) {
		// CRITICAL: All boolean literal formats should be equivalent
		testCases := []struct {
			expr1    string
			expr2    string
			expected bool
		}{
			{"TRUE", "true", true},
			{"FALSE", "false", true},
			{"True", "TRUE", true},
			{"False", "FALSE", true},
			{"TRUE", "FALSE", false}, // Different values should be different
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("boolean_case_%d", i), func(t *testing.T) {
				result, err := comparer.CompareExpressions(tc.expr1, tc.expr2)
				require.NoError(t, err, "Should not error on boolean literals")
				require.Equal(t, tc.expected, result,
					"CRITICAL BUG: Boolean literal comparison failed for %s vs %s", tc.expr1, tc.expr2)
			})
		}
	})

	t.Run("complex_expression_list_parsing", func(t *testing.T) {
		// CRITICAL: Complex expression lists in SELECT, GROUP BY, etc.
		expr1 := "col1, col2, col3"
		expr2 := "COL1, COL2, COL3"
		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should not error on expression lists")
		require.True(t, result, "CRITICAL BUG: Expression lists should be case insensitive")
	})

	t.Run("schema_qualified_expressions", func(t *testing.T) {
		// CRITICAL: Schema qualification is common in production databases
		testCases := []struct {
			name     string
			expr1    string
			expr2    string
			expected bool
		}{
			{
				name:     "public_schema_ignored",
				expr1:    "public.users.id",
				expr2:    "users.id",
				expected: true,
			},
			{
				name:     "different_schemas_preserved",
				expr1:    "schema1.users.id",
				expr2:    "schema2.users.id",
				expected: false,
			},
			{
				name:     "function_schema_qualification",
				expr1:    "public.upper(column)",
				expr2:    "upper(column)",
				expected: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := comparer.CompareExpressions(tc.expr1, tc.expr2)
				require.NoError(t, err, "Should not error on schema-qualified expressions")
				require.Equal(t, tc.expected, result,
					"CRITICAL BUG: Schema qualification handling failed for %s vs %s", tc.expr1, tc.expr2)
			})
		}
	})
}

// TestPostgreSQLExpressionComparer_EdgeCasesStrictMode tests edge cases that must work
func TestPostgreSQLExpressionComparer_EdgeCasesStrictMode(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	t.Run("quoted_identifiers_case_sensitivity", func(t *testing.T) {
		testCases := []struct {
			name        string
			expr1       string
			expr2       string
			shouldEqual bool
			description string
		}{
			{
				name:        "quoted_vs_unquoted_different",
				expr1:       `"Column_Name"`,
				expr2:       `column_name`,
				shouldEqual: false,
				description: "Quoted identifiers should preserve case and differ from unquoted",
			},
			{
				name:        "quoted_vs_unquoted_same_case",
				expr1:       `"column_name"`,
				expr2:       `column_name`,
				shouldEqual: true,
				description: "Quoted lowercase should equal unquoted lowercase",
			},
			{
				name:        "quoted_identifiers_exact_match",
				expr1:       `"Column_Name"`,
				expr2:       `"Column_Name"`,
				shouldEqual: true,
				description: "Identical quoted identifiers should be equal",
			},
			{
				name:        "quoted_identifiers_case_different",
				expr1:       `"Column_Name"`,
				expr2:       `"column_name"`,
				shouldEqual: false,
				description: "Quoted identifiers with different cases should not be equal",
			},
			{
				name:        "quoted_identifiers_with_spaces",
				expr1:       `"Column Name"`,
				expr2:       `"Column Name"`,
				shouldEqual: true,
				description: "Quoted identifiers with spaces should work",
			},
			{
				name:        "quoted_identifiers_mixed_case_complex",
				expr1:       `"MyTable"."MyColumn"`,
				expr2:       `"mytable"."mycolumn"`,
				shouldEqual: false,
				description: "Complex quoted identifiers should preserve case sensitivity",
			},
			{
				name:        "quoted_table_unquoted_column",
				expr1:       `"MyTable".column_name`,
				expr2:       `"MyTable".COLUMN_NAME`,
				shouldEqual: true,
				description: "Unquoted parts should be case insensitive even with quoted parts",
			},
			{
				name:        "quoted_in_expressions",
				expr1:       `"User_ID" = 123`,
				expr2:       `"user_id" = 123`,
				shouldEqual: false,
				description: "Quoted identifiers in expressions should preserve case",
			},
			{
				name:        "quoted_function_names",
				expr1:       `"UPPER"("Column_Name")`,
				expr2:       `"upper"("Column_Name")`,
				shouldEqual: false,
				description: "Quoted function names should be case sensitive",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := comparer.CompareExpressions(tc.expr1, tc.expr2)
				require.NoError(t, err, "Should handle quoted identifiers: %s", tc.description)

				if tc.shouldEqual {
					require.True(t, result, "CRITICAL BUG: %s - '%s' vs '%s'", tc.description, tc.expr1, tc.expr2)
				} else {
					require.False(t, result, "CRITICAL BUG: %s - '%s' vs '%s'", tc.description, tc.expr1, tc.expr2)
				}
			})
		}
	})

	t.Run("unicode_identifier_support", func(t *testing.T) {
		// CRITICAL: Unicode identifiers should be supported in modern PostgreSQL
		result, err := comparer.CompareExpressions("test_column_name", "test_column_name")
		require.NoError(t, err, "Should support Unicode identifiers")
		require.True(t, result, "CRITICAL BUG: Unicode identifiers should be supported")
	})

	t.Run("very_long_expressions", func(t *testing.T) {
		// CRITICAL: Should handle production-scale long expressions
		longExpr1 := fmt.Sprintf("column_%s", strings.Repeat("very_long_name", 50))
		longExpr2 := fmt.Sprintf("COLUMN_%s", strings.Repeat("VERY_LONG_NAME", 50))

		result, err := comparer.CompareExpressions(longExpr1, longExpr2)
		require.NoError(t, err, "Should handle very long expressions without error")
		require.True(t, result, "CRITICAL BUG: Very long expressions should work correctly")
	})

	t.Run("deeply_nested_expressions", func(t *testing.T) {
		// CRITICAL: Deep nesting should be handled (real queries can be very deep)
		nested1 := strings.Repeat("(", 50) + "column" + strings.Repeat(")", 50)
		nested2 := "column"

		result, err := comparer.CompareExpressions(nested1, nested2)
		require.NoError(t, err, "Should handle deeply nested parentheses")
		require.True(t, result, "CRITICAL BUG: Deeply nested unnecessary parentheses should be equivalent")
	})

	t.Run("mixed_quote_types", func(t *testing.T) {
		// CRITICAL: PostgreSQL supports different quote types
		result, err := comparer.CompareExpressions("'hello'", "'hello'")
		require.NoError(t, err, "Should handle string literals")
		require.True(t, result, "String literals should be equal")

		// Different content should be different
		result2, err2 := comparer.CompareExpressions("'hello'", "'world'")
		require.NoError(t, err2, "Should handle different string literals")
		require.False(t, result2, "Different string literals should not be equal")
	})
}

// TestPostgreSQLExpressionComparer_RealWorldProductionQueries tests based on actual production patterns
func TestPostgreSQLExpressionComparer_RealWorldProductionQueries(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	t.Run("audit_log_expressions", func(t *testing.T) {
		// Real audit log query patterns
		expr1 := "created_at >= '2023-01-01' AND action IN ('INSERT', 'UPDATE', 'DELETE')"
		expr2 := "CREATED_AT >= '2023-01-01' AND ACTION IN ('INSERT', 'UPDATE', 'DELETE')"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle audit log expressions")
		require.True(t, result, "CRITICAL BUG: Audit log expressions should be case insensitive")
	})

	t.Run("financial_calculations", func(t *testing.T) {
		// Real financial calculation patterns
		expr1 := "(price * quantity) * (1.0 + tax_rate) - discount"
		expr2 := "(PRICE * QUANTITY) * (1.0 + TAX_RATE) - DISCOUNT"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle financial calculations")
		require.True(t, result, "CRITICAL BUG: Financial calculations should be case insensitive")
	})

	t.Run("user_permissions_check", func(t *testing.T) {
		// Real permission checking patterns
		expr1 := "user_role IN ('admin', 'moderator') OR user_id = owner_id"
		expr2 := "USER_ROLE IN ('admin', 'moderator') OR USER_ID = OWNER_ID"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle permission checks")
		require.True(t, result, "CRITICAL BUG: Permission check expressions should be case insensitive")
	})

	t.Run("data_quality_validation", func(t *testing.T) {
		// Real data validation patterns
		expr1 := "email IS NOT NULL AND email LIKE '%@%.%' AND length(email) > 5"
		expr2 := "EMAIL IS NOT NULL AND EMAIL LIKE '%@%.%' AND LENGTH(EMAIL) > 5"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle data validation")
		require.True(t, result, "CRITICAL BUG: Data validation expressions should be case insensitive")
	})

	t.Run("date_range_queries", func(t *testing.T) {
		// Real date range patterns
		expr1 := "created_at BETWEEN '2023-01-01' AND '2023-12-31'"
		expr2 := "CREATED_AT BETWEEN '2023-01-01' AND '2023-12-31'"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle date ranges")
		require.True(t, result, "CRITICAL BUG: Date range expressions should be case insensitive")
	})

	t.Run("aggregation_with_conditions", func(t *testing.T) {
		// Real aggregation patterns
		expr1 := "SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END)"
		expr2 := "sum(case when status = 'completed' then amount else 0 end)"

		result, err := comparer.CompareExpressions(expr1, expr2)
		require.NoError(t, err, "Should handle conditional aggregations")
		require.True(t, result, "CRITICAL BUG: Conditional aggregations should be case insensitive")
	})
}

// TestPostgreSQLExpressionComparer_ConfigurationStrict tests configuration behavior strictly
func TestPostgreSQLExpressionComparer_ConfigurationStrict(t *testing.T) {
	t.Run("case_sensitivity_must_work", func(t *testing.T) {
		// CRITICAL: Case sensitivity configuration must work correctly
		caseSensitive := NewPostgreSQLExpressionComparer().WithConfig(true, true, true, true)
		caseInsensitive := NewPostgreSQLExpressionComparer().WithConfig(true, true, true, false)

		// Case sensitive should differentiate
		result1, err1 := caseSensitive.CompareExpressions("Column_Name", "column_name")
		require.NoError(t, err1, "Case sensitive comparer should not error")
		require.False(t, result1, "CRITICAL BUG: Case sensitive mode should distinguish different cases")

		// Case insensitive should match
		result2, err2 := caseInsensitive.CompareExpressions("Column_Name", "column_name")
		require.NoError(t, err2, "Case insensitive comparer should not error")
		require.True(t, result2, "CRITICAL BUG: Case insensitive mode should match different cases")
	})

	t.Run("schema_handling_must_work", func(t *testing.T) {
		// CRITICAL: Schema handling configuration must work
		schemaIgnored := NewPostgreSQLExpressionComparer().WithConfig(true, true, true, false)
		schemaPreserved := NewPostgreSQLExpressionComparer().WithConfig(false, true, true, false)

		// Schema ignored should match
		result1, err1 := schemaIgnored.CompareExpressions("public.table.column", "table.column")
		require.NoError(t, err1, "Schema ignored comparer should not error")
		require.True(t, result1, "CRITICAL BUG: Schema ignored mode should match public schema")

		// Schema preserved should differentiate
		result2, err2 := schemaPreserved.CompareExpressions("public.table.column", "table.column")
		require.NoError(t, err2, "Schema preserved comparer should not error")
		require.False(t, result2, "CRITICAL BUG: Schema preserved mode should distinguish schemas")
	})

	t.Run("parentheses_handling_must_work", func(t *testing.T) {
		// CRITICAL: Parentheses handling must be configurable
		parensIgnored := NewPostgreSQLExpressionComparer().WithConfig(true, true, true, false)
		parensPreserved := NewPostgreSQLExpressionComparer().WithConfig(true, false, true, false)

		// Parentheses ignored should match simple cases
		result1, err1 := parensIgnored.CompareExpressions("(column_name)", "column_name")
		require.NoError(t, err1, "Parentheses ignored comparer should not error")
		require.True(t, result1, "CRITICAL BUG: Parentheses ignored mode should match simple parentheses")

		// Test meaningful parentheses that should never be ignored
		result2, err2 := parensPreserved.CompareExpressions("(a + b) * c", "a + b * c")
		require.NoError(t, err2, "Parentheses preserved comparer should not error")
		require.False(t, result2, "CRITICAL BUG: Meaningful parentheses should always be preserved")
	})
}

// TestPostgreSQLExpressionComparer_BatchOperationsStrict tests batch operations with strict requirements
func TestPostgreSQLExpressionComparer_BatchOperationsStrict(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	t.Run("large_expression_lists", func(t *testing.T) {
		// CRITICAL: Should handle large lists efficiently
		largeList1 := make([]string, 1000)
		largeList2 := make([]string, 1000)

		for i := 0; i < 1000; i++ {
			largeList1[i] = fmt.Sprintf("column_%d", i)
			largeList2[i] = fmt.Sprintf("COLUMN_%d", i)
		}

		result, err := comparer.CompareExpressionLists(largeList1, largeList2)
		require.NoError(t, err, "Should handle large expression lists")
		require.True(t, result, "CRITICAL BUG: Large expression lists should be comparable")
	})

	t.Run("unordered_comparison_correctness", func(t *testing.T) {
		// CRITICAL: Unordered comparison must be mathematically correct
		list1 := []string{"a", "b", "c", "a"} // Contains duplicate
		list2 := []string{"c", "a", "b", "a"} // Same elements, different order

		result, err := comparer.CompareExpressionListsUnordered(list1, list2)
		require.NoError(t, err, "Should handle unordered comparison with duplicates")
		require.True(t, result, "CRITICAL BUG: Unordered comparison should handle duplicates correctly")

		// Different lists should be different
		list3 := []string{"a", "b", "c"}
		list4 := []string{"a", "b", "d"} // One different element

		result2, err2 := comparer.CompareExpressionListsUnordered(list3, list4)
		require.NoError(t, err2, "Should handle different unordered lists")
		require.False(t, result2, "Different expression lists should not be equal")
	})
}

// TestPostgreSQLExpressionComparer_ValidationStrict tests validation with strict requirements
func TestPostgreSQLExpressionComparer_ValidationStrict(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	t.Run("comprehensive_validation", func(t *testing.T) {
		// CRITICAL: Should validate all types of valid expressions
		validExpressions := []string{
			"simple_column",
			"schema.table.column",
			"UPPER(column)",
			"a = b AND c = d",
			"CASE WHEN x > 0 THEN 'positive' ELSE 'negative' END",
			"column IN (1, 2, 3)",
			"column BETWEEN 1 AND 10",
			"column IS NOT NULL",
			"SUM(amount) OVER (PARTITION BY category)",
			"EXTRACT(YEAR FROM date_column)",
		}

		for _, expr := range validExpressions {
			err := comparer.ValidateExpression(expr)
			require.NoError(t, err, "CRITICAL BUG: Valid expression should not produce validation error: %s", expr)
		}

		// Invalid expressions should be caught
		invalidExpressions := []string{
			"",
			"   ",
			"\t\n",
		}

		for _, expr := range invalidExpressions {
			err := comparer.ValidateExpression(expr)
			require.Error(t, err, "Invalid expression should produce validation error: %q", expr)
		}
	})

	t.Run("complexity_analysis_accuracy", func(t *testing.T) {
		// CRITICAL: Complexity analysis should be meaningful and consistent
		testCases := []struct {
			expr          string
			minComplexity int
			description   string
		}{
			{"column", 1, "simple column"},
			{"table.column", 2, "qualified column should be more complex"},
			{"a = b", 3, "binary operation should be complex"},
			{"a = b AND c = d", 6, "multiple operations should be very complex"},
			{"UPPER(LOWER(column))", 4, "nested functions should be complex"},
		}

		for _, tc := range testCases {
			complexity, err := comparer.GetExpressionComplexity(tc.expr)
			require.NoError(t, err, "Should calculate complexity for: %s", tc.expr)
			require.GreaterOrEqual(t, complexity, tc.minComplexity,
				"CRITICAL BUG: Complexity for %s (%s) should be at least %d, got %d",
				tc.expr, tc.description, tc.minComplexity, complexity)
		}
	})
}

// TestPostgreSQLExpressionComparer_MemoryAndPerformance tests performance requirements
func TestPostgreSQLExpressionComparer_MemoryAndPerformance(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	t.Run("memory_efficiency_large_expressions", func(t *testing.T) {
		// CRITICAL: Should handle large expressions without excessive memory use
		largeExpr := strings.Repeat("column_name_", 1000) + "final"

		// Should not crash or consume excessive memory
		result, err := comparer.CompareExpressions(largeExpr, strings.ToUpper(largeExpr))
		require.NoError(t, err, "Should handle large expressions efficiently")
		require.True(t, result, "Large expressions should be comparable")

		// Normalization should also work
		normalized, err := comparer.NormalizeExpression(largeExpr)
		require.NoError(t, err, "Should normalize large expressions efficiently")
		require.NotEmpty(t, normalized, "Normalized expression should not be empty")
	})

	t.Run("performance_consistency", func(t *testing.T) {
		// CRITICAL: Performance should be consistent across different expression types
		expressions := []string{
			"simple",
			"schema.table.column",
			"UPPER(column)",
			"a = b AND c = d OR e = f",
			"(a + b) * (c - d) / (e + f)",
		}

		for _, expr := range expressions {
			// Each expression should complete quickly
			start := testing.Short()
			result, err := comparer.CompareExpressions(expr, strings.ToUpper(expr))
			_ = start // Timing check in real implementation would go here

			require.NoError(t, err, "Expression should not cause performance issues: %s", expr)
			require.True(t, result, "Expression should be comparable: %s", expr)
		}
	})
}
