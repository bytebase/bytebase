// Package pg implements the parser for PostgreSQL.
package pg

import (
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	pgquery "github.com/pganalyze/pg_query_go/v2"
	"github.com/pkg/errors"
)

var (
	_ parser.Parser = (*PostgreSQLParser)(nil)
)

const (
	operatorLike    string = "~~"
	operatorNotLike string = "!~~"
)

func init() {
	parser.Register(parser.Postgres, &PostgreSQLParser{})
}

// PostgreSQLParser it the parser for PostgreSQL dialect.
type PostgreSQLParser struct {
}

// Parse implements the parser.Parser interface.
func (*PostgreSQLParser) Parse(_ parser.Context, statement string) ([]ast.Node, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, err
	}

	textList, err := parser.SplitMultiSQL(parser.Postgres, statement)
	if err != nil {
		return nil, err
	}
	if len(res.Stmts) != len(textList) {
		return nil, errors.Errorf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(res.Stmts), len(textList), statement)
	}

	var nodeList []ast.Node

	for i, stmt := range res.Stmts {
		node, err := convert(stmt.Stmt, textList[i])
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, node)
	}
	return nodeList, nil
}
