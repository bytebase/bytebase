package ast

type ObjectType int

const (
	ObjectTypeUndefined ObjectType = iota
	ObjectTypeTable
	ObjectTypeColumn
)

// CommentStmt is the struct for comment statement.
type CommentStmt struct {
	node

	Type    ObjectType
	Object  Node
	Comment string
}
