// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"io"
)

// Node is the interface implemented by all AST node types.
type Node interface {
	// RestoreSQL restores the node to the original SQL statement.
	RestoreSQL(w io.Writer) error
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

// NewMapperNode creates a new mapper node.
func NewMapperNode(namespace string) *MapperNode {
	return &MapperNode{
		Namespace: namespace,
	}
}

// AddChild adds a child to the mapper node.
func (n *MapperNode) AddChild(child *QueryNode) {
	n.QueryNodes = append(n.QueryNodes, child)
}
