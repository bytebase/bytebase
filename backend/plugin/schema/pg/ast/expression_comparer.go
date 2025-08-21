package ast

import (
	"strings"

	"github.com/pkg/errors"
)

// ExpressionComparer provides unified interface for expression semantic comparison
type ExpressionComparer interface {
	// CompareExpressions compares two expressions for semantic equivalence
	CompareExpressions(expr1, expr2 string) (bool, error)

	// NormalizeExpression normalizes expression to canonical form
	NormalizeExpression(expr string) (string, error)

	// ParseExpressionAST parses expression into AST
	ParseExpressionAST(expr string) (ExpressionAST, error)
}

// PostgreSQLExpressionComparer PostgreSQL-specific expression comparer
type PostgreSQLExpressionComparer struct {
	// Configuration options
	IgnoreSchemaPrefix bool // ignore schema prefix (public.table -> table)
	IgnoreParentheses  bool // ignore semantically irrelevant parentheses
	IgnoreWhitespace   bool // ignore whitespace differences
	CaseSensitive      bool // case sensitive comparison

	// Internal components
	parser     *ExpressionParser
	normalizer *ExpressionNormalizer
}

// NewPostgreSQLExpressionComparer creates a new PostgreSQL expression comparer
func NewPostgreSQLExpressionComparer() *PostgreSQLExpressionComparer {
	return &PostgreSQLExpressionComparer{
		IgnoreSchemaPrefix: true,  // default to ignore public schema
		IgnoreParentheses:  true,  // default to ignore irrelevant parentheses
		IgnoreWhitespace:   true,  // default to ignore whitespace
		CaseSensitive:      false, // default case insensitive
		parser:             NewExpressionParser(),
		normalizer:         NewExpressionNormalizer(),
	}
}

// WithConfig configures the comparer with specific options
func (c *PostgreSQLExpressionComparer) WithConfig(ignoreSchema, ignoreParens, ignoreWhitespace, caseSensitive bool) *PostgreSQLExpressionComparer {
	c.IgnoreSchemaPrefix = ignoreSchema
	c.IgnoreParentheses = ignoreParens
	c.IgnoreWhitespace = ignoreWhitespace
	c.CaseSensitive = caseSensitive

	// Create new normalizer with updated configuration
	c.normalizer = &ExpressionNormalizer{
		IgnoreSchemaPrefix: ignoreSchema,
		IgnoreParentheses:  ignoreParens,
		IgnoreWhitespace:   ignoreWhitespace,
		CaseSensitive:      caseSensitive,
	}

	// Create new parser with updated configuration
	c.parser = &ExpressionParser{
		IgnoreSchemaPrefix: ignoreSchema,
		CaseSensitive:      caseSensitive,
	}

	return c
}

// CompareExpressions compares two expressions for semantic equivalence
func (c *PostgreSQLExpressionComparer) CompareExpressions(expr1, expr2 string) (bool, error) {
	// Quick check for identical strings
	if expr1 == expr2 {
		return true, nil
	}

	// Handle empty expressions
	if strings.TrimSpace(expr1) == "" && strings.TrimSpace(expr2) == "" {
		return true, nil
	}
	if strings.TrimSpace(expr1) == "" || strings.TrimSpace(expr2) == "" {
		return false, nil
	}

	// Parse both expressions
	ast1, err1 := c.parser.ParseExpression(expr1)
	if err1 != nil {
		// Fallback to string comparison if parsing fails
		return c.compareExpressionsAsStrings(expr1, expr2), nil // nolint:nilerr
	}

	ast2, err2 := c.parser.ParseExpression(expr2)
	if err2 != nil {
		// Fallback to string comparison if parsing fails
		return c.compareExpressionsAsStrings(expr1, expr2), nil // nolint:nilerr
	}

	if ast1 == nil && ast2 == nil {
		return true, nil
	}
	if ast1 == nil || ast2 == nil {
		return false, nil
	}

	// Normalize both ASTs
	normalized1 := c.normalizer.NormalizeExpression(ast1)
	normalized2 := c.normalizer.NormalizeExpression(ast2)

	// Compare normalized ASTs
	return normalized1.Equals(normalized2), nil
}

// NormalizeExpression normalizes expression to canonical form
func (c *PostgreSQLExpressionComparer) NormalizeExpression(expr string) (string, error) {
	if strings.TrimSpace(expr) == "" {
		return "", nil
	}

	// Parse expression
	ast, err := c.parser.ParseExpression(expr)
	if err != nil {
		return "", err
	}

	if ast == nil {
		return "", nil
	}

	// Normalize AST
	normalized := c.normalizer.NormalizeExpression(ast)

	// Return string representation
	return normalized.String(), nil
}

// ParseExpressionAST parses expression into AST
func (c *PostgreSQLExpressionComparer) ParseExpressionAST(expr string) (ExpressionAST, error) {
	return c.parser.ParseExpression(expr)
}

// compareExpressionsAsStrings fallback string-based comparison
func (c *PostgreSQLExpressionComparer) compareExpressionsAsStrings(expr1, expr2 string) bool {
	// Apply basic normalization
	norm1 := c.normalizeStringExpression(expr1)
	norm2 := c.normalizeStringExpression(expr2)

	return norm1 == norm2
}

// normalizeStringExpression applies basic string normalization
func (c *PostgreSQLExpressionComparer) normalizeStringExpression(expr string) string {
	// Remove extra whitespace
	expr = strings.Join(strings.Fields(expr), " ")

	// Remove schema prefix if configured
	if c.IgnoreSchemaPrefix {
		expr = strings.ReplaceAll(expr, "public.", "")
	}

	// Apply case normalization
	if !c.CaseSensitive {
		expr = strings.ToLower(expr)
	}

	return strings.TrimSpace(expr)
}

// CompareExpressionLists compares two lists of expressions
func (c *PostgreSQLExpressionComparer) CompareExpressionLists(exprs1, exprs2 []string) (bool, error) {
	if len(exprs1) != len(exprs2) {
		return false, nil
	}

	for i, expr1 := range exprs1 {
		equal, err := c.CompareExpressions(expr1, exprs2[i])
		if err != nil {
			return false, err
		}
		if !equal {
			return false, nil
		}
	}

	return true, nil
}

// CompareExpressionListsUnordered compares two lists of expressions ignoring order
func (c *PostgreSQLExpressionComparer) CompareExpressionListsUnordered(exprs1, exprs2 []string) (bool, error) {
	if len(exprs1) != len(exprs2) {
		return false, nil
	}

	// For each expression in first list, find matching expression in second list
	matched := make([]bool, len(exprs2))

	for _, expr1 := range exprs1 {
		found := false
		for j, expr2 := range exprs2 {
			if matched[j] {
				continue // already matched
			}

			equal, err := c.CompareExpressions(expr1, expr2)
			if err != nil {
				return false, err
			}

			if equal {
				matched[j] = true
				found = true
				break
			}
		}

		if !found {
			return false, nil
		}
	}

	return true, nil
}

// NormalizeExpressionList normalizes a list of expressions
func (c *PostgreSQLExpressionComparer) NormalizeExpressionList(exprs []string) ([]string, error) {
	var normalized []string

	for _, expr := range exprs {
		norm, err := c.NormalizeExpression(expr)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, norm)
	}

	return normalized, nil
}

// ValidateExpression validates an expression for correctness
func (c *PostgreSQLExpressionComparer) ValidateExpression(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return errors.New("expression cannot be empty")
	}

	// Parse expression to validate syntax
	ast, err := c.parser.ParseExpression(expr)
	if err != nil {
		return errors.Wrap(err, "failed to parse expression")
	}

	if ast == nil {
		return errors.New("expression parsed to null AST")
	}

	// Use validator visitor to check for semantic errors
	validator := NewExpressionValidatorVisitor()
	validationErrors := validator.Validate(ast)

	if len(validationErrors) > 0 {
		return errors.Errorf("expression validation errors: %v", validationErrors)
	}

	return nil
}

// GetExpressionComplexity returns a complexity score for an expression
func (c *PostgreSQLExpressionComparer) GetExpressionComplexity(expr string) (int, error) {
	ast, err := c.parser.ParseExpression(expr)
	if err != nil {
		return 0, err
	}

	if ast == nil {
		return 0, nil
	}

	return c.calculateComplexity(ast), nil
}

// calculateComplexity calculates complexity score recursively
func (c *PostgreSQLExpressionComparer) calculateComplexity(expr ExpressionAST) int {
	if expr == nil {
		return 0
	}

	complexity := 1 // base complexity for any expression

	// Add complexity based on expression type
	switch e := expr.(type) {
	case *BinaryOpExpr:
		complexity += 2 // binary operations are more complex
		complexity += c.calculateComplexity(e.Left)
		complexity += c.calculateComplexity(e.Right)
	case *UnaryOpExpr:
		complexity++
		complexity += c.calculateComplexity(e.Operand)
	case *FunctionExpr:
		complexity += 3 // functions are complex
		for _, arg := range e.Args {
			complexity += c.calculateComplexity(arg)
		}
	case *ListExpr:
		complexity += len(e.Elements)
		for _, elem := range e.Elements {
			complexity += c.calculateComplexity(elem)
		}
	case *ParenthesesExpr:
		complexity += c.calculateComplexity(e.Inner)
	case *IdentifierExpr:
		// Simple identifier, base complexity already added
		if e.Schema != "" {
			complexity++
		}
		if e.Table != "" {
			complexity++
		}
	case *LiteralExpr:
		// Literal values are simple, base complexity already added
	}

	return complexity
}

// ExpressionStatistics provides statistics about expressions
type ExpressionStatistics struct {
	TotalExpressions  int
	IdentifierCount   int
	LiteralCount      int
	BinaryOpCount     int
	UnaryOpCount      int
	FunctionCount     int
	ListCount         int
	ParenthesesCount  int
	AverageComplexity float64
	MaxComplexity     int
	MinComplexity     int
}

// AnalyzeExpressions analyzes a list of expressions and returns statistics
func (c *PostgreSQLExpressionComparer) AnalyzeExpressions(exprs []string) (*ExpressionStatistics, error) {
	stats := &ExpressionStatistics{
		TotalExpressions: len(exprs),
		MinComplexity:    int(^uint(0) >> 1), // max int
	}

	totalComplexity := 0

	for _, expr := range exprs {
		ast, err := c.parser.ParseExpression(expr)
		if err != nil {
			continue // skip invalid expressions
		}

		if ast == nil {
			continue
		}

		// Count expression types
		c.countExpressionTypes(ast, stats)

		// Calculate complexity
		complexity := c.calculateComplexity(ast)
		totalComplexity += complexity

		if complexity > stats.MaxComplexity {
			stats.MaxComplexity = complexity
		}
		if complexity < stats.MinComplexity {
			stats.MinComplexity = complexity
		}
	}

	if stats.TotalExpressions > 0 {
		stats.AverageComplexity = float64(totalComplexity) / float64(stats.TotalExpressions)
	}

	if stats.MinComplexity == int(^uint(0)>>1) {
		stats.MinComplexity = 0
	}

	return stats, nil
}

// countExpressionTypes recursively counts expression types
func (c *PostgreSQLExpressionComparer) countExpressionTypes(expr ExpressionAST, stats *ExpressionStatistics) {
	if expr == nil {
		return
	}

	switch expr.(type) {
	case *IdentifierExpr:
		stats.IdentifierCount++
	case *LiteralExpr:
		stats.LiteralCount++
	case *BinaryOpExpr:
		stats.BinaryOpCount++
	case *UnaryOpExpr:
		stats.UnaryOpCount++
	case *FunctionExpr:
		stats.FunctionCount++
	case *ListExpr:
		stats.ListCount++
	case *ParenthesesExpr:
		stats.ParenthesesCount++
	}

	// Recursively count children
	for _, child := range expr.Children() {
		c.countExpressionTypes(child, stats)
	}
}

// Global convenience functions

// CompareExpressionsSemantically compares two expressions semantically using default configuration
func CompareExpressionsSemantically(expr1, expr2 string) bool {
	comparer := NewPostgreSQLExpressionComparer()
	result, err := comparer.CompareExpressions(expr1, expr2)
	if err != nil {
		return false
	}
	return result
}

// NormalizeExpressionForComparison normalizes expression using default configuration
func NormalizeExpressionForComparison(expr string) string {
	comparer := NewPostgreSQLExpressionComparer()
	normalized, err := comparer.NormalizeExpression(expr)
	if err != nil {
		return expr // return original if normalization fails
	}
	return normalized
}

// CreatePostgreSQLExpressionComparer creates a PostgreSQL expression comparer with default settings
func CreatePostgreSQLExpressionComparer() ExpressionComparer {
	return NewPostgreSQLExpressionComparer()
}
