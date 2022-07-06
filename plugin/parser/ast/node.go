package ast

// Node is the interface for node.
type Node interface {
	Text() string
	SetText(text string)
}

// node is the base struct for all Node.
type node struct {
	text string
}

// Text implements the Node interface.
func (n *node) Text() string {
	return n.text
}

// SetText implements the Node interface.
func (n *node) SetText(text string) {
	n.text = text
}
