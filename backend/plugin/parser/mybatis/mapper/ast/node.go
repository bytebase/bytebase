// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"io"
	"sort"
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
	// SQLLastLineToOriginalLineMapping is the map of the last line number from SQL statement to the line number of the original SQL statement in Mybatis mapper xml.
	SQLLastLineToOriginalLineMapping map[int]int
	// CurrentLastLine is used for internal calculation.
	CurrentLastLine int

	// RestoreDataNodePlaceholder is the placeholder for restoring data node, it may be different in different engine.
	// For example, in MySQL, it is "?", in PostgreSQL, it is "$1".
	RestoreDataNodePlaceholder string
}

// WithRestoreDataNodePlaceholder set the placeholder for restoring data node.
// DataNodePlaceholder is the placeholder for restoring data node, it may be different in different engine.
// For example:
// MyBatis: #{param}.
// MySQL: ?.
// PostgreSQL: $1.
func (r *RestoreContext) WithRestoreDataNodePlaceholder(placeholder string) *RestoreContext {
	r.RestoreDataNodePlaceholder = placeholder
	return r
}

var (
	_ Node = (*RootNode)(nil)
	_ Node = (*EmptyNode)(nil)
)

// RootNode represents the root node of the AST.
type RootNode struct {
	Children []Node
}

// MybatisSQLLineMapping represents the line mapping of the SQL statement in Mybatis mapper xml.
type MybatisSQLLineMapping struct {
	SQLLastLine     int
	OriginalEleLine int
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

// RestoreSQLWithLineMapping restores the SQL statement and returns the sorted(SQL last line ascending) line mapping of the SQL statement in Mybatis mapper xml.
func (n *RootNode) RestoreSQLWithLineMapping(ctx *RestoreContext, w io.Writer) ([]*MybatisSQLLineMapping, error) {
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return nil, err
		}
	}
	// building line mapping
	var lineMapping []*MybatisSQLLineMapping
	for sqlLastLine, originalEleLine := range ctx.SQLLastLineToOriginalLineMapping {
		lineMapping = append(lineMapping, &MybatisSQLLineMapping{
			SQLLastLine:     sqlLastLine,
			OriginalEleLine: originalEleLine,
		})
	}
	sort.Slice(lineMapping, func(i, j int) bool {
		return lineMapping[i].SQLLastLine < lineMapping[j].SQLLastLine
	})
	return lineMapping, nil
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
