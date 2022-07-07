//go:build !release
// +build !release

package pg

import (
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	pgquery "github.com/pganalyze/pg_query_go/v2"
)

var (
	_ parser.Parser = (*PostgreSQLParser)(nil)
)

func init() {
	parser.Register(parser.Postgres, &PostgreSQLParser{})
}

// PostgreSQLParser it the parser for PostgreSQL dialect.
type PostgreSQLParser struct {
}

// Parse implements the parser.Parser interface.
func (p *PostgreSQLParser) Parse(ctx parser.Context, statement string) ([]ast.Node, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, err
	}

	var nodeList []ast.Node

	for _, stmt := range res.Stmts {
		node, err := convert(stmt.Stmt)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, node)
	}
	return nodeList, nil
}
