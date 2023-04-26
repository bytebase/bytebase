package mybatis

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/mybatis/ast"
)

func Parse(stmt string) (ast.Node, error) {
	reader := strings.NewReader(stmt)
	d := xml.NewDecoder(reader)

	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}
		if startEle, ok := token.(xml.StartElement); ok {
			switch startEle.Name.Local {
			case "mapper":
				return parseMapper(d, &startEle)
			}
		}
	}
}

// parseMapper assumes that <mapper> start element has been consumed, and will consume all tokens until </mapper> end element.
func parseMapper(d *xml.Decoder, mapperStartElement *xml.StartElement) (*ast.MapperNode, error) {
	mapperNode := &ast.MapperNode{}
	for _, attr := range mapperStartElement.Attr {
		if attr.Name.Local == "namespace" {
			mapperNode.Namespace = attr.Value
		}
	}

	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("expected read </mapper> end element, but got EOF")
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}
		if startEle, ok := token.(xml.StartElement); ok {
			switch startEle.Name.Local {
			case "select", "update", "insert", "delete":
				queryNode, err := parseQuery(d, &startEle)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse query node in mapper node %v", mapperNode)
				}
				mapperNode.QueryNodes = append(mapperNode.QueryNodes, queryNode)
			}
		}
		if endEle, ok := token.(xml.EndElement); ok && endEle.Name.Local == mapperStartElement.Name.Local {
			return mapperNode, nil
		}
	}
}

// parseQuery assumes that <select>, <update>, <insert>, <delete> start element has been consumed, and will consume all tokens until </select>, </update>, </insert>, </delete> end element.
func parseQuery(d *xml.Decoder, mapperStartElement *xml.StartElement) (*ast.QueryNode, error) {
	queryNode := &ast.QueryNode{}
	for _, attr := range mapperStartElement.Attr {
		if attr.Name.Local == "id" {
			queryNode.Id = attr.Value
		}
	}

	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("expected read </select>, </update>, </insert>, </delete> end element, but got EOF")
			}
			return nil, errors.Wrapf(err, "failed to get token from xml decoder")
		}
		if charDataEle, ok := token.(xml.CharData); ok {
			// Trim the leading and trailing space.
			trimCharData := strings.TrimSpace(string(charDataEle))
			textNode := &ast.TextNode{
				Text: trimCharData,
			}
			queryNode.Children = append(queryNode.Children, textNode)
		}
		if endEle, ok := token.(xml.EndElement); ok && endEle.Name.Local == mapperStartElement.Name.Local {
			return queryNode, nil
		}
	}
}
