// Package mybatis defines the sql extractor for mybatis mapper xml.
package mybatis

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/mybatis/ast"
)

// Parser is the mybatis mapper xml parser.
type Parser struct {
	d           *xml.Decoder
	buf         []rune
	cursor      uint
	currentLine uint
	mapperNodes []ast.Node
}

// NewParser creates a new mybatis mapper xml parser.
func NewParser(stmt string) *Parser {
	reader := strings.NewReader(stmt)
	d := xml.NewDecoder(reader)
	return &Parser{
		d:      d,
		cursor: 0,
		buf:    nil,
	}
}

// Parse parses the mybatis mapper xml statement and returns the AST node.
func (p *Parser) Parse() ([]ast.Node, error) {
	for {
		token, err := p.d.Token()
		if err != nil {
			if err == io.EOF {
				return p.mapperNodes, nil
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}

		switch ele := token.(type) {
		case xml.StartElement:
			if ele.Name.Local == "mapper" {
				mapperNode, err := p.parseMapper(&ele)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse mapper node")
				}
				p.mapperNodes = append(p.mapperNodes, mapperNode)
			}
		case xml.CharData:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
		}
	}
}

// parseMapper assumes that <mapper> start element has been consumed, and will consume all tokens until </mapper> end element.
func (p *Parser) parseMapper(mapperStartEle *xml.StartElement) (*ast.MapperNode, error) {
	mapperNode := &ast.MapperNode{}
	for _, attr := range mapperStartEle.Attr {
		if attr.Name.Local == "namespace" {
			mapperNode.Namespace = attr.Value
		}
	}

	for {
		token, err := p.d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("expected read </mapper> end element, but got EOF")
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}
		switch ele := token.(type) {
		case xml.StartElement:
			switch ele.Name.Local {
			case "select", "update", "insert", "delete":
				queryNode, err := p.parseQuery(&ele)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse query node in mapper node %v", mapperNode)
				}
				mapperNode.QueryNodes = append(mapperNode.QueryNodes, queryNode)
			}
		case xml.CharData:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
		case xml.EndElement:
			if ele.Name.Local == mapperStartEle.Name.Local {
				return mapperNode, nil
			}
		}
	}
}

// parseQuery assumes that <select>, <update>, <insert>, <delete> start element has been consumed, and will consume all tokens until </select>, </update>, </insert>, </delete> end element.
func (p *Parser) parseQuery(mapperStartElement *xml.StartElement) (*ast.QueryNode, error) {
	queryNode := &ast.QueryNode{}
	for _, attr := range mapperStartElement.Attr {
		if attr.Name.Local == "id" {
			queryNode.ID = attr.Value
		}
	}

	var buf []byte
	for {
		token, err := p.d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("expected read </select>, </update>, </insert>, </delete> end element, but got EOF")
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}

		switch ele := token.(type) {
		case xml.EndElement:
			dataNode := ast.NewDataNode(buf)
			if err := dataNode.Scan(); err != nil {
				return nil, errors.Wrapf(err, "failed to parse data node %v", dataNode)
			}
			queryNode.Children = append(queryNode.Children, dataNode)
			return queryNode, nil
		case xml.CharData:
			for _, b := range ele {
				if b == '\n' {
					p.currentLine++
				}
			}
			trimmed := strings.TrimSpace(string(ele))
			if len(trimmed) > 0 {
				buf = append(buf, trimmed...)
			}
		}
	}
}
