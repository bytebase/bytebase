// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"encoding/xml"
	"io"
	"strings"
)

var (
	_ Node = (*QueryNode)(nil)
)

// QueryNodeType is the type of the query node.
type QueryNodeType uint

const (
	// QueryNodeTypeSelect represents a select query node.
	QueryNodeTypeSelect QueryNodeType = iota
	// QueryNodeTypeInsert represents a insert query node.
	QueryNodeTypeInsert
	// QueryNodeTypeUpdate represents a update query node.
	QueryNodeTypeUpdate
	// QueryNodeTypeDelete represents a delete query node.
	QueryNodeTypeDelete
)

// QueryNode represents a query node.
type QueryNode struct {
	// ID is the id of the query node.
	ID string
	// Type is the type of the query node.
	Type QueryNodeType
	// Children is the children of the query node.
	Children []Node
	// Line is the line of the <select><update><delete><insert> tag.
	Line int
}

// RestoreSQL implements Node interface.
func (n *QueryNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	var sb strings.Builder
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, &sb); err != nil {
			return err
		}
	}
	stmt := sb.String()
	trimmed := strings.TrimSpace(stmt)
	if len(trimmed) == 0 {
		return nil
	}
	if _, err := w.Write([]byte(trimmed)); err != nil {
		return err
	}
	if !strings.HasSuffix(trimmed, ";") {
		if _, err := w.Write([]byte(";")); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	ctx.SQLLastLineToOriginalLineMapping[ctx.CurrentLastLine] = n.Line
	ctx.CurrentLastLine++
	return nil
}

// AddChild adds a child to the query node.
func (n *QueryNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

func (*QueryNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L19
	switch child.(type) {
	case *DataNode, *IncludeNode, *TrimNode, *WhereNode, *SetNode, *ForEachNode, *ChooseNode, *SQLNode, *IfNode:
	default:
		return false
	}
	return true
}

// NewQueryNode creates a new query node.
func NewQueryNode(startEle *xml.StartElement, line int) *QueryNode {
	n := &QueryNode{}
	switch startEle.Name.Local {
	case "select":
		n.Type = QueryNodeTypeSelect
	case "insert":
		n.Type = QueryNodeTypeInsert
	case "update":
		n.Type = QueryNodeTypeUpdate
	case "delete":
		n.Type = QueryNodeTypeDelete
	}

	for _, attr := range startEle.Attr {
		if attr.Name.Local == "id" {
			n.ID = attr.Value
		}
	}
	n.Line = line
	return n
}
