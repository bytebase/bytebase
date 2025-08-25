package ast

import (
	"fmt"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParenthesesMatching tests comprehensive parentheses matching scenarios
func TestParenthesesMatching(t *testing.T) {
	testCases := []struct {
		name               string
		expression         string
		expectParseSuccess bool
		expectParentheses  bool
		expectedStructure  string
		description        string
	}{
		{
			name:               "simple_parentheses",
			expression:         "(a)",
			expectParseSuccess: true,
			expectParentheses:  true,
			expectedStructure:  "Parentheses(Identifier)",
			description:        "Simple parentheses around identifier",
		},
		{
			name:               "nested_parentheses",
			expression:         "((a))",
			expectParseSuccess: true,
			expectParentheses:  true,
			expectedStructure:  "Parentheses(Parentheses(Identifier))",
			description:        "Nested parentheses",
		},
		{
			name:               "function_with_parentheses_arg",
			expression:         "func((a))",
			expectParseSuccess: true,
			expectParentheses:  false, // outer structure should be function
			expectedStructure:  "Function(Parentheses(Identifier))",
			description:        "Function call with parenthesized argument",
		},
		{
			name:               "binary_op_with_parentheses",
			expression:         "(a) + (b)",
			expectParseSuccess: true,
			expectParentheses:  false, // outer structure should be binary op
			expectedStructure:  "BinaryOp(Parentheses(Identifier), Parentheses(Identifier))",
			description:        "Binary operation with parenthesized operands",
		},
		{
			name:               "unmatched_open_paren",
			expression:         "(a",
			expectParseSuccess: true, // should fallback to literal
			expectParentheses:  false,
			expectedStructure:  "Literal",
			description:        "Unmatched opening parenthesis",
		},
		{
			name:               "unmatched_close_paren",
			expression:         "a)",
			expectParseSuccess: true, // should fallback to literal
			expectParentheses:  false,
			expectedStructure:  "Literal",
			description:        "Unmatched closing parenthesis",
		},
		{
			name:               "mismatched_parentheses",
			expression:         "(a))(b",
			expectParseSuccess: true, // should fallback to literal
			expectParentheses:  false,
			expectedStructure:  "Literal",
			description:        "Mismatched parentheses pattern",
		},
		{
			name:               "empty_parentheses",
			expression:         "()",
			expectParseSuccess: true,
			expectParentheses:  true,
			expectedStructure:  "Parentheses(null)",
			description:        "Empty parentheses",
		},
		{
			name:               "complex_nested_expression",
			expression:         "((a + b) * (c - d))",
			expectParseSuccess: true,
			expectParentheses:  true,
			expectedStructure:  "Parentheses(BinaryOp(Parentheses(BinaryOp), Parentheses(BinaryOp)))",
			description:        "Complex nested expression with multiple parentheses levels",
		},
		{
			name:               "partial_wrapping_parentheses",
			expression:         "(a + b) * c",
			expectParseSuccess: true,
			expectParentheses:  false, // outer should be binary op, not parentheses
			expectedStructure:  "BinaryOp(Parentheses(BinaryOp), Identifier)",
			description:        "Parentheses that don't wrap entire expression",
		},
	}

	parser := NewExpressionParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the expression
			ast, err := parser.ParseExpression(tc.expression)

			if tc.expectParseSuccess {
				require.NoError(t, err, "Expected successful parsing for: %s", tc.description)
				require.NotNil(t, ast, "Expected non-nil AST for: %s", tc.description)

				// Check if the top-level structure matches expectation
				if tc.expectParentheses {
					_, isParens := ast.(*ParenthesesExpr)
					assert.True(t, isParens, "Expected parentheses expression for: %s, got: %T", tc.description, ast)
				}

				// Print actual structure for debugging
				t.Logf("Expression: %s", tc.expression)
				t.Logf("Expected: %s", tc.expectedStructure)
				t.Logf("Actual AST: %s", ast.String())
				t.Logf("Actual structure: %s", describeAST(ast))
			} else {
				assert.Error(t, err, "Expected parsing error for: %s", tc.description)
			}
		})
	}
}

// TestParenthesesMatchingHelpers tests the individual helper functions
func TestParenthesesMatchingHelpers(t *testing.T) {
	parser := NewExpressionParser()

	testCases := []struct {
		name            string
		expression      string
		expectedBalance bool
		expectedWrap    bool
		description     string
	}{
		{
			name:            "simple_balanced",
			expression:      "(a)",
			expectedBalance: true,
			expectedWrap:    true,
			description:     "Simple balanced parentheses",
		},
		{
			name:            "nested_balanced",
			expression:      "((a))",
			expectedBalance: true,
			expectedWrap:    true,
			description:     "Nested balanced parentheses",
		},
		{
			name:            "unbalanced_open",
			expression:      "(a",
			expectedBalance: false,
			expectedWrap:    false,
			description:     "Unbalanced - missing close",
		},
		{
			name:            "unbalanced_close",
			expression:      "a)",
			expectedBalance: false,
			expectedWrap:    false,
			description:     "Unbalanced - extra close",
		},
		{
			name:            "partial_wrap",
			expression:      "(a) + b",
			expectedBalance: true,
			expectedWrap:    false,
			description:     "Balanced but doesn't wrap entire expression",
		},
		{
			name:            "empty_parens",
			expression:      "()",
			expectedBalance: true,
			expectedWrap:    true,
			description:     "Empty parentheses",
		},
		{
			name:            "nested_with_extra_content",
			expression:      "((a + b)) + c)",
			expectedBalance: false,
			expectedWrap:    false,
			description:     "Nested parentheses with extra content - should not wrap entire expression",
		},
		{
			name:            "proper_nested_wrapping",
			expression:      "((a + b) + c)",
			expectedBalance: true,
			expectedWrap:    true,
			description:     "Proper nested parentheses that wrap entire expression",
		},
		{
			name:            "edge_case_with_whitespace",
			expression:      "(a + b)   )",
			expectedBalance: false,
			expectedWrap:    false,
			description:     "Edge case with whitespace and extra parenthesis",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tokenizeExpression(tc.expression)
			start := 0
			end := len(tokens)

			// Test isBalancedParentheses
			actualBalance := parser.isBalancedParentheses(tokens, start, end)
			assert.Equal(t, tc.expectedBalance, actualBalance,
				"isBalancedParentheses mismatch for: %s", tc.description)

			// Test parenthesesWrapEntireExpression (only if starts and ends with parentheses)
			if len(tokens) >= 2 && tokens[0].GetText() == "(" && tokens[end-1].GetText() == ")" {
				actualWrap := parser.parenthesesWrapEntireExpression(tokens, start, end)
				assert.Equal(t, tc.expectedWrap, actualWrap,
					"parenthesesWrapEntireExpression mismatch for: %s", tc.description)
			}
		})
	}
}

// TestParenthesesLevelCalculation tests the getParenthesesLevel function
func TestParenthesesLevelCalculation(t *testing.T) {
	parser := NewExpressionParser()

	testCases := []struct {
		expression    string
		pos           int
		expectedLevel int
		description   string
	}{
		{
			expression:    "(a + b)",
			pos:           1, // at 'a'
			expectedLevel: 1,
			description:   "Level 1 inside simple parentheses",
		},
		{
			expression:    "((a) + (b))",
			pos:           2, // at 'a'
			expectedLevel: 2,
			description:   "Level 2 inside nested parentheses",
		},
		{
			expression:    "(a) + (b)",
			pos:           4, // at '+'
			expectedLevel: 0,
			description:   "Level 0 between parentheses groups",
		},
		{
			expression:    "func((a + b), c)",
			pos:           4, // at '+'
			expectedLevel: 2,
			description:   "Level 2 inside function call with nested parentheses",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tokens := tokenizeExpression(tc.expression)
			require.True(t, tc.pos < len(tokens), "Position out of bounds")

			actualLevel := parser.getParenthesesLevel(tokens, 0, tc.pos)
			assert.Equal(t, tc.expectedLevel, actualLevel,
				"Parentheses level mismatch at position %d in expression: %s", tc.pos, tc.expression)
		})
	}
}

// TestEdgeCasesAndErrorHandling tests edge cases and error conditions
func TestEdgeCasesAndErrorHandling(t *testing.T) {
	parser := NewExpressionParser()

	edgeCases := []struct {
		name        string
		expression  string
		expectError bool
		description string
	}{
		{
			name:        "deeply_nested",
			expression:  "(((((a)))))",
			expectError: false,
			description: "Deeply nested parentheses",
		},
		{
			name:        "alternating_parens",
			expression:  "(a)(b)(c)",
			expectError: false,
			description: "Multiple separate parentheses groups",
		},
		{
			name:        "complex_unbalanced",
			expression:  "((a + (b * c)) + d))",
			expectError: false, // should parse as literal
			description: "Complex expression with extra closing paren",
		},
		{
			name:        "mixed_brackets",
			expression:  "(a + [b])",
			expectError: false,
			description: "Mixed parentheses and brackets",
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parser.ParseExpression(tc.expression)

			if tc.expectError {
				assert.Error(t, err, "Expected error for: %s", tc.description)
			} else {
				assert.NoError(t, err, "Unexpected error for: %s", tc.description)
				assert.NotNil(t, ast, "Expected non-nil AST for: %s", tc.description)
				t.Logf("Expression: %s -> %s", tc.expression, ast.String())
			}
		})
	}
}

// Helper functions

// tokenizeExpression converts an expression string to tokens for testing
func tokenizeExpression(expr string) []antlr.Token {
	inputStream := antlr.NewInputStream(expr)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	// Filter out whitespace and EOF tokens
	var filteredTokens []antlr.Token
	for _, token := range tokens {
		if token.GetText() != "<EOF>" && token.GetChannel() == antlr.TokenDefaultChannel {
			filteredTokens = append(filteredTokens, token)
		}
	}
	return filteredTokens
}

// describeAST provides a structural description of an AST for testing
func describeAST(expr ExpressionAST) string {
	if expr == nil {
		return "null"
	}

	switch e := expr.(type) {
	case *ParenthesesExpr:
		return fmt.Sprintf("Parentheses(%s)", describeAST(e.Inner))
	case *FunctionExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, describeAST(arg))
		}
		if len(args) == 0 {
			return fmt.Sprintf("Function(%s)", e.Name)
		}
		return fmt.Sprintf("Function(%s, %v)", e.Name, args)
	case *BinaryOpExpr:
		return fmt.Sprintf("BinaryOp(%s, %s)", describeAST(e.Left), describeAST(e.Right))
	case *UnaryOpExpr:
		return fmt.Sprintf("UnaryOp(%s, %s)", e.Operator, describeAST(e.Operand))
	case *IdentifierExpr:
		return "Identifier"
	case *LiteralExpr:
		return "Literal"
	case *ListExpr:
		return fmt.Sprintf("List(%d)", len(e.Elements))
	default:
		return fmt.Sprintf("Unknown(%T)", expr)
	}
}

// TestGitHubCommentIssues tests specific issues mentioned in GitHub PR comments
func TestGitHubCommentIssues(t *testing.T) {
	comparer := NewPostgreSQLExpressionComparer()

	testCases := []struct {
		name        string
		expr1       string
		expr2       string
		shouldEqual bool
		description string
	}{
		{
			name:        "AND_OR_precedence_parentheses_significant",
			expr1:       "(A AND B) OR C",
			expr2:       "A AND (B OR C)",
			shouldEqual: false,
			description: "Parentheses around AND/OR should be significant for precedence: (A AND B) OR C ≠ A AND (B OR C)",
		},
		{
			name:        "AND_OR_same_precedence_equivalent",
			expr1:       "(A AND B) AND C",
			expr2:       "A AND B AND C",
			shouldEqual: true,
			description: "Parentheses around same precedence operations should not be significant",
		},
		{
			name:        "nested_parentheses_with_extra_content",
			expr1:       "((a + b)) + c)",
			expr2:       "(a + b) + c",
			shouldEqual: false,
			description: "Expression with unmatched parentheses should not equal properly formed expression",
		},
		{
			name:        "OR_operator_equivalence",
			expr1:       "A || B",
			expr2:       "A OR B",
			shouldEqual: true,
			description: "|| and OR operators should be equivalent",
		},
		{
			name:        "complex_precedence_case",
			expr1:       "A AND (B OR C) AND D",
			expr2:       "(A AND (B OR C)) AND D",
			shouldEqual: true,
			description: "Parentheses that don't change precedence should be ignored",
		},
		{
			name:        "test_failure_case",
			expr1:       "A + B",
			expr2:       "C + D",
			shouldEqual: false,
			description: "Completely different expressions should not be equal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := comparer.CompareExpressions(tc.expr1, tc.expr2)
			require.NoError(t, err, "Comparison should not fail for: %s", tc.description)

			if tc.shouldEqual {
				assert.True(t, result, "Expected expressions to be equal: %s vs %s (%s)",
					tc.expr1, tc.expr2, tc.description)
			} else {
				assert.False(t, result, "Expected expressions to be different: %s vs %s (%s)",
					tc.expr1, tc.expr2, tc.description)
			}

			// Also test in reverse order to ensure symmetry
			reverseResult, err := comparer.CompareExpressions(tc.expr2, tc.expr1)
			require.NoError(t, err, "Reverse comparison should not fail")
			assert.Equal(t, result, reverseResult, "Comparison should be symmetric")
		})
	}
}
