// Package mapper defines the sql extractor for mybatis mapper xml.
package mapper

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/mybatis/mapper/ast"
)

// Parser is the mybatis mapper xml parser.
type Parser struct {
	d           *xml.Decoder
	buf         []rune
	cursor      uint
	currentLine int
	sqlMap      map[string]*ast.SQLNode
}

// NewParser creates a new mybatis mapper xml parser.
func NewParser(stmt string) *Parser {
	reader := strings.NewReader(stmt)
	d := xml.NewDecoder(reader)
	return &Parser{
		d:           d,
		cursor:      0,
		buf:         nil,
		sqlMap:      make(map[string]*ast.SQLNode),
		currentLine: 1,
	}
}

// NewRestoreContext returns the restore context.
func (p *Parser) NewRestoreContext() *ast.RestoreContext {
	return &ast.RestoreContext{
		SQLMap:                           p.sqlMap,
		Variable:                         make(map[string]string),
		SQLLastLineToOriginalLineMapping: make(map[int]int),
		CurrentLastLine:                  1,
	}
}

// Parse parses the mybatis mapper xml statements, building AST without recursion, returns the root node of the AST.
func (p *Parser) Parse() (*ast.RootNode, error) {
	root := &ast.RootNode{}
	// To avoid recursion, we use stack to store the start element and node, and consume the token one by one.
	// The length of start element stack is always equal to the length of node stack - 1, because the root nod
	// is not in the start element stack.
	var startElementStack []*xml.StartElement
	nodeStack := []ast.Node{root}

	for {
		token, err := p.d.Token()
		if err != nil {
			if err == io.EOF {
				if len(startElementStack) == 0 {
					return root, nil
				}
				return nil, errors.Errorf("expected to read the end element of %q, but got EOF", startElementStack[len(startElementStack)-1].Name.Local)
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}
		switch ele := token.(type) {
		case xml.StartElement:
			newNode := p.newNodeByStartElement(&ele)
			if ele.Name.Local == "sql" {
				p.sqlMap[newNode.(*ast.SQLNode).ID] = newNode.(*ast.SQLNode)
			}
			startElementStack = append(startElementStack, &ele)
			nodeStack = append(nodeStack, newNode)

		case xml.EndElement:
			if len(startElementStack) == 0 {
				return nil, errors.Errorf("unexpected end element %q", ele.Name.Local)
			}
			if ele.Name.Local != startElementStack[len(startElementStack)-1].Name.Local {
				return nil, errors.Errorf("expected to read the name of end element is %q, but got %q", startElementStack[len(startElementStack)-1].Name.Local, ele.Name.Local)
			}
			// We will pop the start element stack and node stack at the same time.
			startElementStack = startElementStack[:len(startElementStack)-1]
			popNode := nodeStack[len(nodeStack)-1]
			// To avoid keeping many empty node in AST, we only add the node which is not an empty node to the parent node.
			if _, ok := popNode.(*ast.EmptyNode); !ok {
				nodeStack[len(nodeStack)-2].AddChild(popNode)
			}
			nodeStack = nodeStack[:len(nodeStack)-1]
		case xml.CharData:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
			trimmed := strings.TrimSpace(string(ele))
			if len(trimmed) == 0 {
				continue
			}
			dataNode := ast.NewDataNode([]byte(trimmed))
			if err := dataNode.Scan(); err != nil {
				return nil, errors.Wrapf(err, "cannot parse data node")
			}
			if len(nodeStack) == 0 {
				return nil, errors.Errorf("try to append data node to parent node, but node stack is empty")
			}
			nodeStack[len(nodeStack)-1].AddChild(dataNode)
		case xml.Comment:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
		case xml.Directive:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
		case xml.ProcInst:
			for _, b := range ele.Inst {
				if b == '\n' {
					p.currentLine++
				}
			}
		}
	}
}

// newNodeByStartElement returns the node related to the startElement, for example, returns QueryNode for
// start element which name is "select", "update", "insert", "delete". If the startElement is unacceptable,
// returns an emptyNode instead.
func (p *Parser) newNodeByStartElement(startElement *xml.StartElement) ast.Node {
	switch startElement.Name.Local {
	case "mapper":
		return ast.NewMapperNode(startElement)
	case "select", "update", "insert", "delete":
		return ast.NewQueryNode(startElement, p.currentLine)
	case "if":
		return ast.NewIfNode(startElement)
	case "choose":
		return ast.NewChooseNode(startElement)
	case "when":
		return ast.NewWhenNode(startElement)
	case "otherwise":
		return ast.NewOtherwiseNode(startElement)
	case "where":
		return ast.NewWhereNode(startElement)
	case "set":
		return ast.NewSetNode(startElement)
	case "trim":
		return ast.NewTrimNode(startElement)
	case "foreach":
		return ast.NewForeachNode(startElement)
	case "sql":
		return ast.NewSQLNode(startElement)
	case "include":
		return ast.NewIncludeNode(startElement)
	case "property":
		return ast.NewPropertyNode(startElement)
	}
	return ast.NewEmptyNode()
}
