// Package ast defines the abstract syntax tree of mybatis mapper xml.
package ast

import (
	"encoding/xml"
	"io"
)

var (
	_ Node = (*IfNode)(nil)
	_ Node = (*ChooseNode)(nil)
	_ Node = (*WhenNode)(nil)
	_ Node = (*OtherwiseNode)(nil)
)

// IfNode represents a if node in mybatis mapper xml likes <if test="condition">...</if>.
type IfNode struct {
	Test     string
	Children []Node
}

// NewIfNode creates a new if node.
func NewIfNode(startElement *xml.StartElement) *IfNode {
	node := &IfNode{}
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "test" {
			node.Test = attr.Value
		}
	}
	return node
}

// RestoreSQL implements Node interface, the if condition will be ignored.
func (n *IfNode) RestoreSQL(w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild adds a child to the if node.
func (n *IfNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}

// ChooseNode represents a choose node in mybatis mapper xml likes <choose>...</choose>.
type ChooseNode struct {
	Children []Node
}

// NewChooseNode creates a new choose node.
func NewChooseNode(_ *xml.StartElement) *ChooseNode {
	return &ChooseNode{}
}

// RestoreSQL implements Node interface.
func (n *ChooseNode) RestoreSQL(w io.Writer) error {
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild implements Node interface.
func (n *ChooseNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}

// WhenNode represents a when node in mybatis mapper xml select node likes <select><when test="condition">...</when></select>.
type WhenNode struct {
	Test     string
	Children []Node
}

// NewWhenNode creates a new when node.
func NewWhenNode(startElement *xml.StartElement) *WhenNode {
	node := &WhenNode{}
	for _, attr := range startElement.Attr {
		if attr.Name.Local == "test" {
			node.Test = attr.Value
		}
	}
	return node
}

// RestoreSQL implements Node interface, the when condition will be ignored.
func (n *WhenNode) RestoreSQL(w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild adds a child to the when node.
func (n *WhenNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}

// OtherwiseNode represents a otherwise node in mybatis mapper xml select node likes <select><otherwise>...</otherwise></select>.
type OtherwiseNode struct {
	Children []Node
}

// NewOtherwiseNode creates a new otherwise node.
func NewOtherwiseNode(_ *xml.StartElement) *OtherwiseNode {
	return &OtherwiseNode{}
}

// RestoreSQL implements Node interface.
func (n *OtherwiseNode) RestoreSQL(w io.Writer) error {
	if len(n.Children) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
	}
	for _, node := range n.Children {
		if err := node.RestoreSQL(w); err != nil {
			return err
		}
	}
	return nil
}

// AddChild adds a child to the otherwise node.
func (n *OtherwiseNode) AddChild(child Node) {
	n.Children = append(n.Children, child)
}
