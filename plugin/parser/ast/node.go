package ast

var (
	_ Node = (*node)(nil)
)

// Node is the interface for node.
type Node interface {
	Text() string
	SetText(text string)
	Line() int
	SetLine(line int)
}

// node is the base struct for all Node.
type node struct {
	text string
	line int
}

// Text implements the Node interface.
func (n *node) Text() string {
	return n.text
}

// SetText implements the Node interface.
func (n *node) SetText(text string) {
	n.text = text
}

func (n *node) Line() int {
	return n.line
}

func (n *node) SetLine(line int) {
	n.line = line
}
