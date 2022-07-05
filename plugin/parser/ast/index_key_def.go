package ast

// IndexKeyType is the type for index key.
type IndexKeyType int

const (
	// IndexKeyTypeColumn is the type for the column.
	IndexKeyTypeColumn IndexKeyType = iota
	// IndexKeyTypeExpression is the type for the expression.
	IndexKeyTypeExpression
)

// IndexKeyDef is the struct for index key definition.
// Only support conversion IndexKeyTypeColumn now.
// TODO(rebelice): support conversion IndexKeyTypeExpression.
type IndexKeyDef struct {
	node

	Type IndexKeyType
	Key  string
}
