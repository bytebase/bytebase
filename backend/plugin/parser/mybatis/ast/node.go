package ast

import (
	"io"
)

// Node is the interface implemented by all AST node types.
type Node interface {
	// RestoreSQL restores the node to the original SQL statement.
	RestoreSQL(w io.Writer) error
}

// TextNode represents a text node.
type TextNode struct {
	Text string
}

// RestoreSQL implements Node interface.
func (n *TextNode) RestoreSQL(w io.Writer) error {
	_, err := w.Write([]byte(n.Text))
	return err
}

// MapperNode represents a mapper node in mybatis mapper xml begin with <mapper>.
type MapperNode struct {
	Namespace  string
	QueryNodes []*QueryNode
}

// RestoreSQL implements Node interface.
func (n *MapperNode) RestoreSQL(w io.Writer) error {
	for _, node := range n.QueryNodes {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// QueryNode represents a query node.
type QueryNode struct {
	// Id is the id of the query node.
	Id string
	// Children is the children of the query node.
	Children []Node
}

// RestoreSQL implements Node interface.
func (n *QueryNode) RestoreSQL(w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	w.Write([]byte(";\n"))
	return nil
}
