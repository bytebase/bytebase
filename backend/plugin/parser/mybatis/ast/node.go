// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"io"
)

// Node is the interface implemented by all AST node types.
type Node interface {
	// RestoreSQL restores the node to the original SQL statement.
	RestoreSQL(w io.Writer) error
	// AddChild adds a child to the node.
	AddChild(child Node)
}

var (
	_ Node = (*RootNode)(nil)
	_ Node = (*EmptyNode)(nil)
)

// RootNode represents the root node of the AST.
type RootNode struct {
	Children []Node
}

// RestoreSQL implements Node interface.
func (n *RootNode) RestoreSQL(w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild adds a child to the root node.
func (n *RootNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}

// EmptyNode represents an unacceptable nodes in mybatis mapper xml.
type EmptyNode struct{}

// NewEmptyNode returns a new empty node.
func NewEmptyNode() *EmptyNode {
	return &EmptyNode{}
}

// RestoreSQL implements Node interface.
func (*EmptyNode) RestoreSQL(io.Writer) error {
	return nil
}

// AddChild implements Node interface.
func (*EmptyNode) AddChild(Node) {
}
