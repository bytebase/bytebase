package base

import (
	"strconv"
	"strings"
)

var (
	_ SelectorNode = (*ItemSelector)(nil)
	_ SelectorNode = (*ArraySelector)(nil)
)

type PathAST struct {
	Root SelectorNode
}

type SelectorNode interface {
	toString() string
	GetIdentifier() string
	SetNext(SelectorNode)
	GetNext() SelectorNode
}

type ItemSelector struct {
	Identifier string
	Next       SelectorNode
}

type ArraySelector struct {
	Identifier string
	Index      int
	Next       SelectorNode
}

func NewPathAST(selectorNode SelectorNode) *PathAST {
	return &PathAST{
		Root: selectorNode,
	}
}

func (p *PathAST) String() (string, error) {
	return p.Root.toString(), nil
}

func NewItemSelector(identifier string) *ItemSelector {
	return &ItemSelector{
		Identifier: identifier,
	}
}

func (n *ItemSelector) toString() string {
	sb := new(strings.Builder)
	if _, err := sb.WriteString(n.Identifier); err != nil {
		return ""
	}
	if v := n.Next; v != nil {
		if _, err := sb.WriteString("."); err != nil {
			return ""
		}
		if _, err := sb.WriteString(v.toString()); err != nil {
			return ""
		}
	}

	return sb.String()
}

func (n *ItemSelector) GetNext() SelectorNode {
	return n.Next
}

func (n *ItemSelector) SetNext(next SelectorNode) {
	n.Next = next
}

func (n *ItemSelector) GetIdentifier() string {
	return n.Identifier
}

func NewArraySelector(identifier string, index int) *ArraySelector {
	return &ArraySelector{
		Identifier: identifier,
		Index:      index,
	}
}
func (n *ArraySelector) toString() string {
	sb := new(strings.Builder)
	if _, err := sb.WriteString(n.Identifier); err != nil {
		return ""
	}
	if _, err := sb.WriteString("["); err != nil {
		return ""
	}
	if _, err := sb.WriteString(strconv.Itoa(n.Index)); err != nil {
		return ""
	}
	if _, err := sb.WriteString("]"); err != nil {
		return ""
	}
	if v := n.Next; v != nil {
		if _, err := sb.WriteString("."); err != nil {
			return ""
		}
		if _, err := sb.WriteString(v.toString()); err != nil {
			return ""
		}
	}

	return sb.String()
}

func (n *ArraySelector) GetNext() SelectorNode {
	return n.Next
}

func (n *ArraySelector) SetNext(next SelectorNode) {
	n.Next = next
}

func (n *ArraySelector) GetIdentifier() string {
	return n.Identifier
}
