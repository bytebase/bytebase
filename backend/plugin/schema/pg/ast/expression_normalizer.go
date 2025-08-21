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
	right := v.normalizeChild(expr.Right)
	operator := v.normalizeOperator(expr.Operator)

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

// normalizeIdentifier normalizes identifier names
func (v *normalizationVisitor) normalizeIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	// Handle quoted identifiers - preserve exact case
	if v.isQuotedIdentifier(identifier) {
		return identifier
	}

	// For unquoted identifiers, apply case normalization
	if v.normalizer.CaseSensitive {
		return identifier
	}

	return strings.ToLower(identifier)
}

// normalizeFunctionName normalizes function names
func (*normalizationVisitor) normalizeFunctionName(name string) string {
	// Function names are case-insensitive in PostgreSQL
	return strings.ToLower(name)
}

// normalizeLiteral normalizes literal values
func (*normalizationVisitor) normalizeLiteral(value, valueType string) string {
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
	default:
		return value
	}
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
	// For now, we use a simple heuristic - this could be made more sophisticated

	inner := expr.Inner
	if inner == nil {
		return false
	}

	// Parentheses around binary operations might be significant for precedence
	if _, isBinaryOp := inner.(*BinaryOpExpr); isBinaryOp {
		// Could analyze the context to determine if parentheses change precedence
		// For now, be conservative and keep them
		return true
	}

	// Parentheses around simple identifiers or literals are usually not significant
	switch inner.(type) {
	case *IdentifierExpr, *LiteralExpr:
		return false
	default:
		return true // be conservative
	}
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
