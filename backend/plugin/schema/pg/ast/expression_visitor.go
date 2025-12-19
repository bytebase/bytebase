package ast

import (
	"fmt"
	"strings"
)

// BaseExpressionVisitor provides a base implementation of ExpressionVisitor
type BaseExpressionVisitor struct{}

// VisitIdentifier visits identifier expressions
func (*BaseExpressionVisitor) VisitIdentifier(expr *IdentifierExpr) any {
	return expr
}

// VisitLiteral visits literal expressions
func (*BaseExpressionVisitor) VisitLiteral(expr *LiteralExpr) any {
	return expr
}

// VisitBinaryOp visits binary operation expressions
func (*BaseExpressionVisitor) VisitBinaryOp(expr *BinaryOpExpr) any {
	return expr
}

// VisitUnaryOp visits unary operation expressions
func (*BaseExpressionVisitor) VisitUnaryOp(expr *UnaryOpExpr) any {
	return expr
}

// VisitFunction visits function expressions
func (*BaseExpressionVisitor) VisitFunction(expr *FunctionExpr) any {
	return expr
}

// VisitList visits list expressions
func (*BaseExpressionVisitor) VisitList(expr *ListExpr) any {
	return expr
}

// VisitParentheses visits parentheses expressions
func (*BaseExpressionVisitor) VisitParentheses(expr *ParenthesesExpr) any {
	return expr
}

// ExpressionValidatorVisitor validates expressions for correctness
type ExpressionValidatorVisitor struct {
	BaseExpressionVisitor
	errors []string
}

// NewExpressionValidatorVisitor creates a new expression validator visitor
func NewExpressionValidatorVisitor() *ExpressionValidatorVisitor {
	return &ExpressionValidatorVisitor{}
}

// Validate validates an expression and returns any errors found
func (v *ExpressionValidatorVisitor) Validate(expr ExpressionAST) []string {
	v.errors = nil
	v.visitRecursively(expr)
	return v.errors
}

// visitRecursively visits expression and validates it
func (v *ExpressionValidatorVisitor) visitRecursively(expr ExpressionAST) {
	if expr == nil {
		v.errors = append(v.errors, "null expression found")
		return
	}

	// Validate based on expression type
	switch e := expr.(type) {
	case *IdentifierExpr:
		v.validateIdentifier(e)
	case *LiteralExpr:
		v.validateLiteral(e)
	case *BinaryOpExpr:
		v.validateBinaryOp(e)
	case *UnaryOpExpr:
		v.validateUnaryOp(e)
	case *FunctionExpr:
		v.validateFunction(e)
	case *ListExpr:
		v.validateList(e)
	case *ParenthesesExpr:
		v.validateParentheses(e)
	}

	// Recursively validate children
	for _, child := range expr.Children() {
		v.visitRecursively(child)
	}
}

// validateIdentifier validates identifier expressions
func (v *ExpressionValidatorVisitor) validateIdentifier(expr *IdentifierExpr) {
	if expr.Name == "" {
		v.errors = append(v.errors, "identifier name cannot be empty")
	}

	// Could add more validation rules for identifier names
	if strings.ContainsAny(expr.Name, " \t\n\r") && !v.isQuotedIdentifier(expr.Name) {
		v.errors = append(v.errors, fmt.Sprintf("unquoted identifier contains whitespace: %s", expr.Name))
	}
}

// validateLiteral validates literal expressions
func (v *ExpressionValidatorVisitor) validateLiteral(expr *LiteralExpr) {
	if expr.Value == "" && expr.ValueType != "null" {
		v.errors = append(v.errors, "literal value cannot be empty unless it's null")
	}

	// Validate literal format based on type
	switch expr.ValueType {
	case "string":
		if !v.isValidStringLiteral(expr.Value) {
			v.errors = append(v.errors, fmt.Sprintf("invalid string literal: %s", expr.Value))
		}
	case "number":
		if !v.isValidNumericLiteral(expr.Value) {
			v.errors = append(v.errors, fmt.Sprintf("invalid numeric literal: %s", expr.Value))
		}
	case "boolean":
		upper := strings.ToUpper(expr.Value)
		if upper != "TRUE" && upper != "FALSE" {
			v.errors = append(v.errors, fmt.Sprintf("invalid boolean literal: %s", expr.Value))
		}
	case "null":
		if strings.ToUpper(expr.Value) != "NULL" {
			v.errors = append(v.errors, fmt.Sprintf("invalid null literal: %s", expr.Value))
		}
	default:
		// Unknown value type - this is acceptable as the type system may support additional types
	}
}

// validateBinaryOp validates binary operation expressions
func (v *ExpressionValidatorVisitor) validateBinaryOp(expr *BinaryOpExpr) {
	if expr.Left == nil {
		v.errors = append(v.errors, "binary operation missing left operand")
	}
	if expr.Right == nil {
		v.errors = append(v.errors, "binary operation missing right operand")
	}
	if expr.Operator == "" {
		v.errors = append(v.errors, "binary operation missing operator")
	}

	// Validate operator
	if !v.isValidBinaryOperator(expr.Operator) {
		v.errors = append(v.errors, fmt.Sprintf("invalid binary operator: %s", expr.Operator))
	}
}

// validateUnaryOp validates unary operation expressions
func (v *ExpressionValidatorVisitor) validateUnaryOp(expr *UnaryOpExpr) {
	if expr.Operand == nil {
		v.errors = append(v.errors, "unary operation missing operand")
	}
	if expr.Operator == "" {
		v.errors = append(v.errors, "unary operation missing operator")
	}

	// Validate operator
	if !v.isValidUnaryOperator(expr.Operator) {
		v.errors = append(v.errors, fmt.Sprintf("invalid unary operator: %s", expr.Operator))
	}
}

// validateFunction validates function expressions
func (v *ExpressionValidatorVisitor) validateFunction(expr *FunctionExpr) {
	if expr.Name == "" {
		v.errors = append(v.errors, "function name cannot be empty")
	}

	// Could add validation for function name format
	if strings.ContainsAny(expr.Name, " \t\n\r") {
		v.errors = append(v.errors, fmt.Sprintf("function name contains whitespace: %s", expr.Name))
	}
}

// validateList validates list expressions
func (v *ExpressionValidatorVisitor) validateList(expr *ListExpr) {
	if len(expr.Elements) == 0 {
		v.errors = append(v.errors, "list expression cannot be empty")
	}
}

// validateParentheses validates parentheses expressions
func (v *ExpressionValidatorVisitor) validateParentheses(expr *ParenthesesExpr) {
	if expr.Inner == nil {
		v.errors = append(v.errors, "parentheses expression cannot be empty")
	}
}

// Helper validation methods

// isQuotedIdentifier checks if identifier is quoted
func (*ExpressionValidatorVisitor) isQuotedIdentifier(name string) bool {
	return len(name) >= 2 &&
		((name[0] == '"' && name[len(name)-1] == '"') ||
			(name[0] == '`' && name[len(name)-1] == '`'))
}

// isValidStringLiteral validates string literal format
func (*ExpressionValidatorVisitor) isValidStringLiteral(value string) bool {
	if len(value) < 2 {
		return false
	}
	return (value[0] == '\'' && value[len(value)-1] == '\'') ||
		(value[0] == '"' && value[len(value)-1] == '"')
}

// isValidNumericLiteral validates numeric literal format
func (*ExpressionValidatorVisitor) isValidNumericLiteral(value string) bool {
	if len(value) == 0 {
		return false
	}

	hasDecimal := false
	for i, char := range value {
		if i == 0 && (char == '+' || char == '-') {
			continue
		}
		if char == '.' {
			if hasDecimal {
				return false // multiple decimals
			}
			hasDecimal = true
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// isValidBinaryOperator validates binary operators
func (*ExpressionValidatorVisitor) isValidBinaryOperator(operator string) bool {
	validOps := []string{
		"=", "<>", "!=", "<", ">", "<=", ">=",
		"+", "-", "*", "/", "%",
		"AND", "OR", "LIKE", "ILIKE", "IN", "NOT IN",
		"||",                     // string concatenation or logical OR
		"IS NULL", "IS NOT NULL", // NULL checking operators
	}

	upper := strings.ToUpper(operator)
	for _, validOp := range validOps {
		if upper == validOp {
			return true
		}
	}

	return false
}

// isValidUnaryOperator validates unary operators
func (*ExpressionValidatorVisitor) isValidUnaryOperator(operator string) bool {
	validOps := []string{"NOT", "-", "+"}

	upper := strings.ToUpper(operator)
	for _, validOp := range validOps {
		if upper == validOp {
			return true
		}
	}

	return false
}
