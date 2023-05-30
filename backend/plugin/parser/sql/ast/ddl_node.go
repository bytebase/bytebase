package ast

var (
	_ DDLNode = (*ddl)(nil)
)

// DDLNode is the interface for DDL.
type DDLNode interface {
	Node

	ddlNode()
}

type ddl struct {
	node
}

func (*ddl) ddlNode() {}
