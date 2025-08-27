package ast

import (
	"fmt"
	"slices"
	"strings"
)

// ExpressionType expression type enumeration
type ExpressionType int

const (
	ExprTypeIdentifier  ExpressionType = iota // identifier (column names, table names, etc.)
	ExprTypeLiteral                           // literal values (numbers, strings, etc.)
	ExprTypeBinaryOp                          // binary operators (+, -, =, <>, etc.)
	ExprTypeUnaryOp                           // unary operators (NOT, -, etc.)
	ExprTypeFunction                          // function calls
	ExprTypeColumn                            // column references (may include table prefix)
	ExprTypeParameter                         // parameter placeholders
	ExprTypeList                              // expression lists (IN clauses, etc.)
	ExprTypeParentheses                       // parentheses expressions
)

// ExpressionVisitor visitor pattern interface for expression AST
type ExpressionVisitor interface {
	VisitIdentifier(expr *IdentifierExpr) any
	VisitLiteral(expr *LiteralExpr) any
	VisitBinaryOp(expr *BinaryOpExpr) any
	VisitUnaryOp(expr *UnaryOpExpr) any
	VisitFunction(expr *FunctionExpr) any
	VisitList(expr *ListExpr) any
	VisitParentheses(expr *ParenthesesExpr) any
}

// ExpressionAST abstract syntax tree node interface for expressions
type ExpressionAST interface {
	Type() ExpressionType
	String() string                       // normalized string representation
	Equals(other ExpressionAST) bool      // semantic equivalence comparison
	Children() []ExpressionAST            // child nodes
	Accept(visitor ExpressionVisitor) any // visitor pattern support
}

// IdentifierExpr identifier expression
type IdentifierExpr struct {
	Schema string // optional schema name
	Table  string // optional table name
	Name   string // identifier name
}

func (*IdentifierExpr) Type() ExpressionType {
	return ExprTypeIdentifier
}

func (e *IdentifierExpr) String() string {
	var parts []string
	if e.Schema != "" {
		parts = append(parts, e.Schema)
	}
	if e.Table != "" {
		parts = append(parts, e.Table)
	}
	parts = append(parts, e.Name)
	return strings.Join(parts, ".")
}

func (e *IdentifierExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeIdentifier {
		return false
	}
	o, ok := other.(*IdentifierExpr)
	if !ok {
		return false
	}

	// Compare using normalized string representation to respect normalization settings
	return e.String() == o.String()
}

func (*IdentifierExpr) Children() []ExpressionAST {
	return nil
}

func (e *IdentifierExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitIdentifier(e)
}

// LiteralExpr literal expression
type LiteralExpr struct {
	Value     string // literal value
	ValueType string // type (string, number, boolean, null)
}

func (*LiteralExpr) Type() ExpressionType {
	return ExprTypeLiteral
}

func (e *LiteralExpr) String() string {
	return e.Value
}

func (e *LiteralExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeLiteral {
		return false
	}
	o, ok := other.(*LiteralExpr)
	if !ok {
		return false
	}
	return e.Value == o.Value && e.ValueType == o.ValueType
}

func (*LiteralExpr) Children() []ExpressionAST {
	return nil
}

func (e *LiteralExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitLiteral(e)
}

// BinaryOpExpr binary operation expression
type BinaryOpExpr struct {
	Left     ExpressionAST
	Operator string // normalized operator (=, <>, +, -, AND, OR, etc.)
	Right    ExpressionAST
}

func (*BinaryOpExpr) Type() ExpressionType {
	return ExprTypeBinaryOp
}

func (e *BinaryOpExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.Left.String(), e.Operator, e.Right.String())
}

func (e *BinaryOpExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeBinaryOp {
		return false
	}
	o, ok := other.(*BinaryOpExpr)
	if !ok {
		return false
	}

	// normalize operators
	op1 := normalizeOperator(e.Operator)
	op2 := normalizeOperator(o.Operator)

	if op1 != op2 {
		return false
	}

	// for commutative operators, check both orders
	if isCommutativeOperator(op1) {
		return (e.Left.Equals(o.Left) && e.Right.Equals(o.Right)) ||
			(e.Left.Equals(o.Right) && e.Right.Equals(o.Left))
	}

	return e.Left.Equals(o.Left) && e.Right.Equals(o.Right)
}

func (e *BinaryOpExpr) Children() []ExpressionAST {
	return []ExpressionAST{e.Left, e.Right}
}

func (e *BinaryOpExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitBinaryOp(e)
}

// UnaryOpExpr unary operation expression
type UnaryOpExpr struct {
	Operator string        // operator (NOT, -, +, etc.)
	Operand  ExpressionAST // operand
}

func (*UnaryOpExpr) Type() ExpressionType {
	return ExprTypeUnaryOp
}

func (e *UnaryOpExpr) String() string {
	return fmt.Sprintf("%s %s", e.Operator, e.Operand.String())
}

func (e *UnaryOpExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeUnaryOp {
		return false
	}
	o, ok := other.(*UnaryOpExpr)
	if !ok {
		return false
	}
	return normalizeOperator(e.Operator) == normalizeOperator(o.Operator) &&
		e.Operand.Equals(o.Operand)
}

func (e *UnaryOpExpr) Children() []ExpressionAST {
	return []ExpressionAST{e.Operand}
}

func (e *UnaryOpExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitUnaryOp(e)
}

// FunctionExpr function call expression
type FunctionExpr struct {
	Schema string          // optional schema name
	Name   string          // function name
	Args   []ExpressionAST // argument list
}

func (*FunctionExpr) Type() ExpressionType {
	return ExprTypeFunction
}

func (e *FunctionExpr) String() string {
	var name string
	if e.Schema != "" {
		name = fmt.Sprintf("%s.%s", e.Schema, e.Name)
	} else {
		name = e.Name
	}

	var args []string
	for _, arg := range e.Args {
		args = append(args, arg.String())
	}

	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}

func (e *FunctionExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeFunction {
		return false
	}
	o, ok := other.(*FunctionExpr)
	if !ok {
		return false
	}

	// Compare using string representation which handles normalization
	return e.String() == o.String()
}

func (e *FunctionExpr) Children() []ExpressionAST {
	return e.Args
}

func (e *FunctionExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitFunction(e)
}

// ListExpr expression list (for IN clauses, etc.)
type ListExpr struct {
	Elements []ExpressionAST
}

func (*ListExpr) Type() ExpressionType {
	return ExprTypeList
}

func (e *ListExpr) String() string {
	var elements []string
	for _, elem := range e.Elements {
		elements = append(elements, elem.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(elements, ", "))
}

func (e *ListExpr) Equals(other ExpressionAST) bool {
	if other.Type() != ExprTypeList {
		return false
	}
	o, ok := other.(*ListExpr)
	if !ok {
		return false
	}

	if len(e.Elements) != len(o.Elements) {
		return false
	}

	// for lists, order usually matters, but could consider unordered comparison
	for i, elem := range e.Elements {
		if !elem.Equals(o.Elements[i]) {
			return false
		}
	}

	return true
}

func (e *ListExpr) Children() []ExpressionAST {
	return e.Elements
}

func (e *ListExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitList(e)
}

// ParenthesesExpr parentheses expression
type ParenthesesExpr struct {
	Inner ExpressionAST
}

func (*ParenthesesExpr) Type() ExpressionType {
	return ExprTypeParentheses
}

func (e *ParenthesesExpr) String() string {
	if e.Inner == nil {
		return "()"
	}
	return fmt.Sprintf("(%s)", e.Inner.String())
}

func (e *ParenthesesExpr) Equals(other ExpressionAST) bool {
	// Handle nil inner expressions
	if e.Inner == nil {
		if other.Type() == ExprTypeParentheses {
			if otherParen, ok := other.(*ParenthesesExpr); ok {
				return otherParen.Inner == nil
			}
		}
		return false
	}

	// If the other expression is also parentheses, compare inner expressions
	if other.Type() == ExprTypeParentheses {
		if otherParen, ok := other.(*ParenthesesExpr); ok {
			if otherParen.Inner == nil {
				return false
			}
			return e.Inner.Equals(otherParen.Inner)
		}
	}

	// Otherwise, parentheses are usually ignored in semantic comparison
	// Compare inner expression directly with the other expression
	return e.Inner.Equals(other)
}

func (e *ParenthesesExpr) Children() []ExpressionAST {
	if e.Inner == nil {
		return nil
	}
	return []ExpressionAST{e.Inner}
}

func (e *ParenthesesExpr) Accept(visitor ExpressionVisitor) any {
	return visitor.VisitParentheses(e)
}

// Helper functions

// normalizeOperator normalizes operators
func normalizeOperator(op string) string {
	switch strings.ToUpper(op) {
	case "!=":
		return "<>"
	case "&&":
		return "AND"
	case "||":
		// In PostgreSQL, || is primarily string concatenation, but in boolean contexts
		// it can be used as OR. For expression comparison, we treat || as OR since
		// this is the most common semantic usage in filter expressions.
		return "OR"
	case "IS NULL", "ISNULL":
		return "IS NULL"
	case "IS NOT NULL", "ISNOTNULL":
		return "IS NOT NULL"
	case "::":
		// PostgreSQL type cast operator
		return "::"
	default:
		return strings.ToUpper(op)
	}
}

// isCommutativeOperator checks if operator satisfies commutative property
func isCommutativeOperator(op string) bool {
	commutativeOps := map[string]bool{
		"=":   true,
		"<>":  true,
		"!=":  true,
		"+":   true,
		"*":   true,
		"AND": true,
		"OR":  true,
	}
	return commutativeOps[strings.ToUpper(op)]
}

// SortExpressions sorts expression list for unordered comparison
func SortExpressions(exprs []ExpressionAST) []ExpressionAST {
	sorted := make([]ExpressionAST, len(exprs))
	copy(sorted, exprs)

	slices.SortFunc(sorted, func(a, b ExpressionAST) int {
		if a.String() < b.String() {
			return -1
		}
		if a.String() > b.String() {
			return 1
		}
		return 0
	})

	return sorted
}
