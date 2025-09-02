package ast

import (
	"slices"
	"strings"
)

// ExpressionNormalizer normalizes expressions for semantic comparison
type ExpressionNormalizer struct {
	// Configuration options
	IgnoreSchemaPrefix bool // ignore public schema prefix
	IgnoreParentheses  bool // ignore semantically irrelevant parentheses
	IgnoreWhitespace   bool // ignore whitespace differences
	CaseSensitive      bool // case sensitive comparison
}

// NewExpressionNormalizer creates a new expression normalizer
func NewExpressionNormalizer() *ExpressionNormalizer {
	return &ExpressionNormalizer{
		IgnoreSchemaPrefix: true,  // default to ignore public schema
		IgnoreParentheses:  true,  // default to ignore irrelevant parentheses
		IgnoreWhitespace:   true,  // default to ignore whitespace
		CaseSensitive:      false, // default case insensitive
	}
}

// NormalizeExpression normalizes an expression AST for comparison
func (n *ExpressionNormalizer) NormalizeExpression(expr ExpressionAST) ExpressionAST {
	if expr == nil {
		return nil
	}

	visitor := &normalizationVisitor{normalizer: n}
	result := expr.Accept(visitor)
	if normalized, ok := result.(ExpressionAST); ok {
		return normalized
	}
	return expr
}

// NormalizeExpressionString normalizes an expression string for comparison
func (n *ExpressionNormalizer) NormalizeExpressionString(exprStr string) (string, error) {
	parser := NewExpressionParser()
	ast, err := parser.ParseExpression(exprStr)
	if err != nil {
		return "", err
	}

	if ast == nil {
		return "", nil
	}

	normalized := n.NormalizeExpression(ast)
	return normalized.String(), nil
}

// normalizationVisitor implements the visitor pattern for normalization
type normalizationVisitor struct {
	normalizer *ExpressionNormalizer
}

// VisitIdentifier normalizes identifier expressions
func (v *normalizationVisitor) VisitIdentifier(expr *IdentifierExpr) any {
	schema := v.normalizeSchema(expr.Schema)
	table := v.normalizeIdentifier(expr.Table)
	name := v.normalizeIdentifier(expr.Name)

	// Special handling for two-part identifiers where first part might be public schema
	if schema == "" && table != "" && strings.ToLower(table) == "public" && v.normalizer.IgnoreSchemaPrefix {
		// This is likely public.column, treat as just column
		table = ""
	}

	normalized := &IdentifierExpr{
		Schema: schema,
		Table:  table,
		Name:   name,
	}
	return normalized
}

// VisitLiteral normalizes literal expressions
func (v *normalizationVisitor) VisitLiteral(expr *LiteralExpr) any {
	normalized := &LiteralExpr{
		Value:     v.normalizeLiteral(expr.Value, expr.ValueType),
		ValueType: expr.ValueType,
	}
	return normalized
}

// VisitBinaryOp normalizes binary operation expressions
func (v *normalizationVisitor) VisitBinaryOp(expr *BinaryOpExpr) any {
	left := v.normalizeChild(expr.Left)
	operator := v.normalizeOperator(expr.Operator)

	// Special handling for type cast operator
	var right ExpressionAST
	if operator == "::" {
		// For type cast, the right operand should be normalized as a type name
		right = v.normalizeChildAsType(expr.Right)

		// Check if this is a redundant type cast that can be removed
		if v.isRedundantTypeCast(left, right) {
			return left // Return the left operand without the cast
		}
	} else {
		right = v.normalizeChild(expr.Right)
	}

	// For commutative operators, ensure consistent ordering
	if isCommutativeOperator(operator) && v.shouldSwapOperands(left, right) {
		left, right = right, left
	}

	return &BinaryOpExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// VisitUnaryOp normalizes unary operation expressions
func (v *normalizationVisitor) VisitUnaryOp(expr *UnaryOpExpr) any {
	return &UnaryOpExpr{
		Operator: v.normalizeOperator(expr.Operator),
		Operand:  v.normalizeChild(expr.Operand),
	}
}

// VisitFunction normalizes function expressions
func (v *normalizationVisitor) VisitFunction(expr *FunctionExpr) any {
	var normalizedArgs []ExpressionAST
	for _, arg := range expr.Args {
		normalizedArgs = append(normalizedArgs, v.normalizeChild(arg))
	}

	return &FunctionExpr{
		Schema: v.normalizeSchema(expr.Schema),
		Name:   v.normalizeFunctionName(expr.Name),
		Args:   normalizedArgs,
	}
}

// VisitList normalizes list expressions
func (v *normalizationVisitor) VisitList(expr *ListExpr) any {
	var normalizedElements []ExpressionAST
	for _, elem := range expr.Elements {
		normalizedElements = append(normalizedElements, v.normalizeChild(elem))
	}

	// For some contexts, list order might not matter, but default to preserving order
	return &ListExpr{
		Elements: normalizedElements,
	}
}

// VisitParentheses normalizes parentheses expressions
func (v *normalizationVisitor) VisitParentheses(expr *ParenthesesExpr) any {
	inner := v.normalizeChild(expr.Inner)

	// If configured to ignore irrelevant parentheses, return inner expression
	if v.normalizer.IgnoreParentheses && !v.isSignificantParentheses(expr) {
		return inner
	}

	return &ParenthesesExpr{Inner: inner}
}

// Helper methods for normalization

// normalizeChild normalizes a child expression
func (v *normalizationVisitor) normalizeChild(expr ExpressionAST) ExpressionAST {
	if expr == nil {
		return nil
	}
	result := expr.Accept(v)
	if normalized, ok := result.(ExpressionAST); ok {
		return normalized
	}
	return expr
}

// normalizeChildAsType normalizes a child expression as a type name
func (v *normalizationVisitor) normalizeChildAsType(expr ExpressionAST) ExpressionAST {
	if expr == nil {
		return nil
	}

	// If it's a literal, treat it as a type name
	if literal, ok := expr.(*LiteralExpr); ok {
		normalizedValue := v.normalizeTypeName(literal.Value)
		return &LiteralExpr{
			Value:     normalizedValue,
			ValueType: literal.ValueType,
		}
	}

	// Otherwise, normalize normally
	return v.normalizeChild(expr)
}

// normalizeSchema normalizes schema names
func (v *normalizationVisitor) normalizeSchema(schema string) string {
	if schema == "" {
		return ""
	}

	// Remove public schema if configured to ignore it
	if v.normalizer.IgnoreSchemaPrefix && strings.ToLower(schema) == "public" {
		return ""
	}

	return v.normalizeIdentifier(schema)
}

// normalizeIdentifier normalizes identifier names according to PostgreSQL rules:
// - Unquoted identifiers are case-insensitive and folded to lowercase
// - Quoted identifiers preserve their original case but without quotes in the IR
func (v *normalizationVisitor) normalizeIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	// Handle quoted identifiers
	if v.isQuotedIdentifier(identifier) {
		// Remove quotes and preserve the original case content
		unquoted := identifier[1 : len(identifier)-1]
		// Handle escaped quotes within the identifier
		unquoted = strings.ReplaceAll(unquoted, `""`, `"`)
		return unquoted
	}

	// For unquoted identifiers, apply PostgreSQL case folding (to lowercase)
	return v.normalizeUnquotedIdentifier(identifier)
}

// normalizeUnquotedIdentifier normalizes unquoted identifiers
func (v *normalizationVisitor) normalizeUnquotedIdentifier(identifier string) string {
	// For unquoted identifiers, apply case normalization
	if v.normalizer.CaseSensitive {
		return identifier
	}

	return strings.ToLower(identifier)
}

// normalizeFunctionName normalizes function names using PostgreSQL standard rules
// Same logic as normalizeIdentifier - function names are identifiers
func (v *normalizationVisitor) normalizeFunctionName(name string) string {
	// Function names follow the same identifier normalization rules
	return v.normalizeIdentifier(name)
}

// normalizeLiteral normalizes literal values
func (v *normalizationVisitor) normalizeLiteral(value, valueType string) string {
	switch strings.ToLower(valueType) {
	case "boolean":
		return strings.ToUpper(value) // TRUE/FALSE
	case "null":
		return "NULL"
	case "string":
		// Normalize string literals - preserve quotes and content
		return value
	case "number":
		// Normalize numeric literals - could handle leading zeros, etc.
		return value
	case "type":
		// Type names in cast expressions should be case-insensitive
		return v.normalizeTypeName(value)
	default:
		return value
	}
}

// normalizeTypeName normalizes type names for case-insensitive comparison
func (v *normalizationVisitor) normalizeTypeName(typeName string) string {
	if v.normalizer.CaseSensitive {
		return typeName
	}
	return strings.ToLower(typeName)
}

// normalizeOperator normalizes operators
func (*normalizationVisitor) normalizeOperator(operator string) string {
	return normalizeOperator(operator) // Use the helper from expression_ast.go
}

// shouldSwapOperands determines if operands should be swapped for consistent ordering
func (*normalizationVisitor) shouldSwapOperands(left, right ExpressionAST) bool {
	// Simple ordering based on string representation
	// This ensures consistent ordering for commutative operators
	leftStr := left.String()
	rightStr := right.String()
	return leftStr > rightStr
}

// isSignificantParentheses determines if parentheses are semantically significant
func (*normalizationVisitor) isSignificantParentheses(expr *ParenthesesExpr) bool {
	// Parentheses are significant if they change operator precedence or grouping

	inner := expr.Inner
	if inner == nil {
		return false
	}

	// Parentheses around simple identifiers or literals are never significant
	// This is the key insight: (name) and name are always semantically equivalent
	switch inner.(type) {
	case *IdentifierExpr, *LiteralExpr:
		return false
	}

	// Parentheses around binary operations might be significant for precedence
	if binaryOp, isBinaryOp := inner.(*BinaryOpExpr); isBinaryOp {
		// For most practical cases in filter expressions, parentheses around
		// high-precedence operators (=, <>, <, >, etc.) are not significant
		// when used with lower-precedence operators (AND, OR)
		op := strings.ToUpper(binaryOp.Operator)

		// High precedence comparison operators usually don't need parentheses
		highPrecedenceOps := map[string]bool{
			"=":           true,
			"<>":          true,
			"!=":          true,
			"<":           true,
			">":           true,
			"<=":          true,
			">=":          true,
			"LIKE":        true,
			"ILIKE":       true,
			"IS NULL":     true,
			"IS NOT NULL": true,
			"::":          true, // Type cast operator
		}

		if highPrecedenceOps[op] {
			return false // These operators have high precedence, parentheses usually not needed
		}

		// For AND/OR operators, parentheses might be significant for grouping
		// But in simple cases like (A AND B) vs A AND B, they're not significant
		if op == "AND" || op == "OR" {
			return false // Allow AND/OR to be ungrouped for simple cases
		}

		// Be conservative for other operators
		return true
	}

	// Be conservative for other expression types (functions, etc.)
	return true
}

// isQuotedIdentifier checks if an identifier is quoted
func (*normalizationVisitor) isQuotedIdentifier(identifier string) bool {
	return len(identifier) >= 2 &&
		((identifier[0] == '"' && identifier[len(identifier)-1] == '"') ||
			(identifier[0] == '`' && identifier[len(identifier)-1] == '`'))
}

// ExpressionListNormalizer provides utilities for normalizing lists of expressions
type ExpressionListNormalizer struct {
	normalizer *ExpressionNormalizer
}

// NewExpressionListNormalizer creates a new expression list normalizer
func NewExpressionListNormalizer() *ExpressionListNormalizer {
	return &ExpressionListNormalizer{
		normalizer: NewExpressionNormalizer(),
	}
}

// NormalizeExpressionList normalizes a list of expression strings
func (ln *ExpressionListNormalizer) NormalizeExpressionList(expressions []string) ([]string, error) {
	var normalized []string

	for _, expr := range expressions {
		normalizedExpr, err := ln.normalizer.NormalizeExpressionString(expr)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, normalizedExpr)
	}

	return normalized, nil
}

// NormalizeAndSortExpressionList normalizes and sorts a list of expressions for unordered comparison
func (ln *ExpressionListNormalizer) NormalizeAndSortExpressionList(expressions []string) ([]string, error) {
	normalized, err := ln.NormalizeExpressionList(expressions)
	if err != nil {
		return nil, err
	}

	// Sort the normalized expressions
	// This is useful for comparing lists where order doesn't matter
	return SortStringList(normalized), nil
}

// SortStringList sorts a list of strings
func SortStringList(strs []string) []string {
	sorted := make([]string, len(strs))
	copy(sorted, strs)

	// Use Go's slices.Sort function (more efficient than sort.Strings)
	slices.Sort(sorted)

	return sorted
}

// isRedundantTypeCast checks if a type cast can be safely removed
// This handles cases where PostgreSQL implicit casting makes explicit casts unnecessary
func (v *normalizationVisitor) isRedundantTypeCast(leftExpr, typeExpr ExpressionAST) bool {
	// Check if left is a string literal and right is a JSON/JSONB type
	if literalExpr, ok := leftExpr.(*LiteralExpr); ok {
		if typeIdent, ok := typeExpr.(*IdentifierExpr); ok {
			typeName := strings.ToLower(typeIdent.Name)

			// For JSON-like literals cast to jsonb/json, the cast is redundant
			// because PostgreSQL will implicitly cast string literals to JSON types
			if (typeName == "jsonb" || typeName == "json") &&
				literalExpr.ValueType == "string" &&
				v.isJSONLiteral(literalExpr.Value) {
				return true
			}
		}
	}

	return false
}

// isJSONLiteral checks if a string value looks like JSON
func (*normalizationVisitor) isJSONLiteral(value string) bool {
	// Remove quotes if present
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	// Check for simple JSON patterns
	trimmed := strings.TrimSpace(value)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) ||
		trimmed == "null" ||
		trimmed == "true" ||
		trimmed == "false" ||
		(len(trimmed) > 0 && (trimmed[0] == '"' ||
			(trimmed[0] >= '0' && trimmed[0] <= '9') ||
			trimmed[0] == '-'))
}
