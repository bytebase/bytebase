package ast

var (
	_ DMLNode = (*dml)(nil)
)

// DMLNode is the interface for DML.
type DMLNode interface {
	Node

	dmlNode()
}

type dml struct {
	node
}

func (*dml) dmlNode() {}
