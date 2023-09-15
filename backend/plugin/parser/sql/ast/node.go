package ast

import (
	pgquery "github.com/pganalyze/pg_query_go/v4"
)

var (
	_ Node = (*node)(nil)
)

// Node is the interface for node.
type Node interface {
	Text() string
	SetText(text string)
	LastLine() int
	SetLastLine(line int)
}

// node is the base struct for all Node.
type node struct {
	text     string
	lastline int
	*pgquery.ParseResult
}

// Text implements the Node interface.
func (n *node) Text() string {
	return n.text
}

// SetText implements the Node interface.
func (n *node) SetText(text string) {
	n.text = text
}

func (n *node) LastLine() int {
	return n.lastline
}

func (n *node) SetLastLine(line int) {
	n.lastline = line
}
