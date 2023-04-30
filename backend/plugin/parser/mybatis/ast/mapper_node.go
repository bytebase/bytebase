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
func (n *MapperNode) RestoreSQL(w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
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
	n.Children = append(n.Children, child)
}
