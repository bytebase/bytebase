// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"encoding/xml"
	"io"
)

var (
	_ Node = (*MapperNode)(nil)
)

// MapperNode represents a mapper node in mybatis mapper xml begin with <mapper>.
type MapperNode struct {
	Namespace string
	Children  []Node
}

// RestoreSQL implements Node interface.
func (n *MapperNode) RestoreSQL(ctx *RestoreContext, w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

// NewMapperNode creates a new mapper node.
func NewMapperNode(startElement *xml.StartElement) *MapperNode {
	m := &MapperNode{}
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "namespace" {
			m.Namespace = attr.Value
		}
	}
	return m
}

// AddChild adds a child to the mapper node.
func (n *MapperNode) AddChild(child Node) {
	if !n.isChildAcceptable(child) {
		return
	}
	n.Children = append(n.Children, child)
}

func (*MapperNode) isChildAcceptable(child Node) bool {
	// https://github.com/mybatis/mybatis-3/blob/master/src/main/resources/org/apache/ibatis/builder/xml/mybatis-3-mapper.dtd#L19
	switch child.(type) {
	case *SQLNode, *QueryNode:
		return true
	default:
		return false
	}
}
