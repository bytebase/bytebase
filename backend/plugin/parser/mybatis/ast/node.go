// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"io"
)

// Node is the interface implemented by all AST node types.
type Node interface {
	// RestoreSQL restores the node to the original SQL statement.
	RestoreSQL(ctx *RestoreContext, w io.Writer) error
	// AddChild adds a child to the node.
	AddChild(child Node)
	// isChildAcceptable checks whether the child is acceptable.
	isChildAcceptable(child Node) bool
}

// RestoreContext is the context for restoring SQL statement.
type RestoreContext struct {
	// SQLMap is the map of SQL statement, key is the id of the sql element, value is the node of the sql element.
	// It will be used to restore the <include> element.
	SQLMap map[string]*SQLNode
	// Variable is the map of variable, key is the name of the variable, value is the value of the variable.
	// It will be used to restore the ${} element.
	Variable map[string]string
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
func (n *RootNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild adds a child to the root node.
func (n *RootNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}

// isChildAcceptable implements Node interface.
func (*RootNode) isChildAcceptable(Node) bool {
	return true
}

// EmptyNode represents an unacceptable nodes in mybatis mapper xml.
type EmptyNode struct {
}

// NewEmptyNode returns a new empty node.
func NewEmptyNode() *EmptyNode {
	return &EmptyNode{}
}

// RestoreSQL implements Node interface.
func (*EmptyNode) RestoreSQL(*RestoreContext, io.Writer) error {
	return nil
}

// AddChild implements Node interface.
func (*EmptyNode) AddChild(Node) {
}

// isChildAcceptable implements Node interface.
func (*EmptyNode) isChildAcceptable(Node) bool {
	return false
}
