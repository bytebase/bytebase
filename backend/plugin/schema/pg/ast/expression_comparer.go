package ast

import (
	"strings"

	"github.com/pkg/errors"
)

// ExpressionComparer provides unified interface for expression semantic comparison
type ExpressionComparer interface {
	// CompareExpressions compares two expressions for semantic equivalence
	CompareExpressions(expr1, expr2 string) (bool, error)

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
	result := normalized1.Equals(normalized2)

	// If AST comparison fails, try fallback string comparison for special cases
	if !result {
		fallbackResult := c.compareExpressionsAsStrings(expr1, expr2)
		return fallbackResult, nil
	}

	return result, nil
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

	if norm1 == norm2 {
		return true
	}

	// Handle PostgreSQL implicit type casts
	return c.handleImplicitTypeCasts(norm1, norm2)
}

// normalizeStringExpression applies basic string normalization
func (c *PostgreSQLExpressionComparer) normalizeStringExpression(expr string) string {
	// Remove extra whitespace
	expr = strings.Join(strings.Fields(expr), " ")

	// Remove schema prefix if configured
	if c.IgnoreSchemaPrefix {
		expr = strings.ReplaceAll(expr, "public.", "")
	}

	// Always normalize built-in types regardless of IgnoreSchemaPrefix setting
	// This is necessary because built-in types should never be schema-qualified
	expr = c.normalizeBuiltinTypes(expr)

	// Apply case normalization - but preserve string literal case
	if !c.CaseSensitive {
		expr = c.normalizeCasePreservingStringLiterals(expr)
	}

	return strings.TrimSpace(expr)
}

// handleImplicitTypeCasts handles PostgreSQL implicit type casting scenarios
func (c *PostgreSQLExpressionComparer) handleImplicitTypeCasts(expr1, expr2 string) bool {
	// Handle JSONB implicit casts
	// '{}' should be equivalent to '{}'::jsonb
	if c.isJSONBImplicitCast(expr1, expr2) {
		return true
	}

	// Handle BIT literal format differences
	// B'1010' should be equivalent to '1010'::"bit"
	if c.isBitLiteralEquivalent(expr1, expr2) {
		return true
	}

	// Handle array type implicit casts (already covered by AST comparison)
	// 'ARRAY[]::TEXT[]' vs 'ARRAY[]::text[]'
	if c.isArrayTypeImplicitCast(expr1, expr2) {
		return true
	}

	return false
}

// isJSONBImplicitCast checks if two expressions represent the same JSONB value with/without explicit cast
func (*PostgreSQLExpressionComparer) isJSONBImplicitCast(expr1, expr2 string) bool {
	// Check if one has ::jsonb suffix and the other doesn't, but otherwise identical
	if strings.HasSuffix(expr1, "::jsonb") && !strings.HasSuffix(expr2, "::jsonb") {
		baseExpr1 := strings.TrimSuffix(expr1, "::jsonb")
		baseExpr1 = strings.TrimSpace(baseExpr1)
		return baseExpr1 == expr2
	}
	if strings.HasSuffix(expr2, "::jsonb") && !strings.HasSuffix(expr1, "::jsonb") {
		baseExpr2 := strings.TrimSuffix(expr2, "::jsonb")
		baseExpr2 = strings.TrimSpace(baseExpr2)
		return baseExpr2 == expr1
	}

	// Also handle JSON type casts (json vs jsonb)
	if strings.HasSuffix(expr1, "::json") && !strings.HasSuffix(expr2, "::json") && !strings.HasSuffix(expr2, "::jsonb") {
		baseExpr1 := strings.TrimSuffix(expr1, "::json")
		baseExpr1 = strings.TrimSpace(baseExpr1)
		return baseExpr1 == expr2
	}
	if strings.HasSuffix(expr2, "::json") && !strings.HasSuffix(expr1, "::json") && !strings.HasSuffix(expr1, "::jsonb") {
		baseExpr2 := strings.TrimSuffix(expr2, "::json")
		baseExpr2 = strings.TrimSpace(baseExpr2)
		return baseExpr2 == expr1
	}

	return false
}

// isBitLiteralEquivalent checks if two expressions represent the same BIT value in different formats
func (*PostgreSQLExpressionComparer) isBitLiteralEquivalent(expr1, expr2 string) bool {
	// Check for b'...' vs '...'::"bit" patterns (lowercase b due to AST normalization)
	if strings.HasPrefix(expr1, "b'") && strings.HasSuffix(expr1, "'") &&
		strings.Contains(expr2, ":") && strings.Contains(expr2, "bit") {
		// Extract bit value from b'...' format
		bitValue1 := expr1[2 : len(expr1)-1] // Remove b'...'

		// Extract bit value from '...'::"bit" format
		var bitValue2 string
		if strings.HasPrefix(expr2, "'") && strings.Contains(expr2, "':") {
			// Find the closing quote before the type cast
			endQuotePos := strings.Index(expr2[1:], "'")
			if endQuotePos > 0 {
				bitValue2 = expr2[1 : endQuotePos+1] // Extract between first ' and second '
			}
		}

		return bitValue1 == bitValue2 && bitValue2 != ""
	}

	// Check the reverse case: '...'::"bit" vs b'...'
	if strings.HasPrefix(expr2, "b'") && strings.HasSuffix(expr2, "'") &&
		strings.Contains(expr1, ":") && strings.Contains(expr1, "bit") {
		// Extract bit value from b'...' format
		bitValue2 := expr2[2 : len(expr2)-1] // Remove b'...'

		// Extract bit value from '...'::"bit" format
		var bitValue1 string
		if strings.HasPrefix(expr1, "'") && strings.Contains(expr1, "':") {
			// Find the closing quote before the type cast
			endQuotePos := strings.Index(expr1[1:], "'")
			if endQuotePos > 0 {
				bitValue1 = expr1[1 : endQuotePos+1] // Extract between first ' and second '
			}
		}

		return bitValue1 == bitValue2 && bitValue1 != ""
	}

	return false
}

// isArrayTypeImplicitCast checks if two expressions represent array types with different case
func (c *PostgreSQLExpressionComparer) isArrayTypeImplicitCast(expr1, expr2 string) bool {
	// This is primarily handled by the AST parser, but we can add string-level fallback
	if c.CaseSensitive {
		return false
	}

	// Check for array type patterns like ARRAY[]::TYPE[] vs ARRAY[]::type[]
	if strings.Contains(expr1, "ARRAY[]::") && strings.Contains(expr2, "ARRAY[]::") {
		return strings.EqualFold(expr1, expr2)
	}

	return false
}

// normalizeBuiltinTypes removes public schema qualification from built-in PostgreSQL types
func (*PostgreSQLExpressionComparer) normalizeBuiltinTypes(expr string) string {
	// List of PostgreSQL built-in types that should not be schema-qualified
	builtinTypes := []string{
		"varbit", "bit", "bit varying",
		"varchar", "character varying", "char", "character",
		"text", "numeric", "decimal", "int", "integer", "bigint", "smallint",
		"float", "real", "double precision", "boolean", "bool",
		"date", "time", "timestamp", "timestamptz", "interval",
		"uuid", "json", "jsonb", "bytea", "inet", "cidr",
		"point", "line", "lseg", "box", "path", "polygon", "circle",
		"int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange",
	}

	for _, builtinType := range builtinTypes {
		// Replace public.typename with typename
		publicQualified := "public." + builtinType
		if strings.Contains(expr, publicQualified) {
			expr = strings.ReplaceAll(expr, publicQualified, builtinType)
		}

		// Also handle quoted versions
		quotedPublicQualified := "\"public\"." + builtinType
		if strings.Contains(expr, quotedPublicQualified) {
			expr = strings.ReplaceAll(expr, quotedPublicQualified, builtinType)
		}
	}

	return expr
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

// normalizeCasePreservingStringLiterals normalizes case while preserving string literal content
func (*PostgreSQLExpressionComparer) normalizeCasePreservingStringLiterals(expr string) string {
	// This is a production-grade implementation that preserves:
	// 1. Single-quoted string literal case ('...')
	// 2. Double-quoted identifier case ("...")
	// while normalizing unquoted identifiers, keywords, and function names

	result := make([]rune, 0, len(expr))
	runes := []rune(expr)
	i := 0

	for i < len(runes) {
		char := runes[i]

		// Handle single-quoted string literals
		if char == '\'' {
			// Preserve the entire string literal including quotes
			result = append(result, char) // opening quote
			i++

			// Copy content until closing quote, handling escaped quotes
			for i < len(runes) {
				char = runes[i]
				result = append(result, char)

				if char == '\'' {
					// Check if this is an escaped quote ''
					if i+1 < len(runes) && runes[i+1] == '\'' {
						// Escaped quote, copy both quotes
						i++
						result = append(result, runes[i])
					} else {
						// End of string literal
						break
					}
				}
				i++
			}
		} else if char == '"' {
			// Handle double-quoted identifiers (case-sensitive)
			// Preserve the entire quoted identifier including quotes
			result = append(result, char) // opening quote
			i++

			// Copy content until closing quote, handling escaped quotes
			for i < len(runes) {
				char = runes[i]
				result = append(result, char)

				if char == '"' {
					// Check if this is an escaped quote ""
					if i+1 < len(runes) && runes[i+1] == '"' {
						// Escaped quote, copy both quotes
						i++
						result = append(result, runes[i])
					} else {
						// End of quoted identifier
						break
					}
				}
				i++
			}
		} else {
			// For non-quoted parts, apply case normalization
			result = append(result, char)
		}
		i++
	}

	// Apply case normalization to the parts outside quoted literals/identifiers
	finalResult := string(result)

	// Process both single and double quotes to preserve their content
	// First handle single quotes (string literals)
	singleQuoteParts := strings.Split(finalResult, "'")
	for i := 0; i < len(singleQuoteParts); i++ {
		if i%2 == 0 {
			// Outside single quotes - now check for double quotes
			doubleQuoteParts := strings.Split(singleQuoteParts[i], `"`)
			for j := 0; j < len(doubleQuoteParts); j++ {
				if j%2 == 0 {
					// Outside both single and double quotes - apply case normalization
					doubleQuoteParts[j] = strings.ToLower(doubleQuoteParts[j])
				}
				// Inside double quotes - preserve case
			}
			singleQuoteParts[i] = strings.Join(doubleQuoteParts, `"`)
		}
		// Inside single quotes - preserve case
	}

	return strings.Join(singleQuoteParts, "'")
}

// CreatePostgreSQLExpressionComparer creates a PostgreSQL expression comparer with default settings
func CreatePostgreSQLExpressionComparer() ExpressionComparer {
	return NewPostgreSQLExpressionComparer()
}
