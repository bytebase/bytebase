package mysql

import (
	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

// Wrapper for parser.New().
func newParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func parseStatement(statement string, charset string, collation string) ([]ast.StmtNode, []advisor.Advice) {
	list, err := parser.SplitMySQL(statement)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Warn,
				Code:    advisor.Internal,
				Title:   "Syntax error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	p := newParser()
	var returnNodes []ast.StmtNode
	var adviceList []advisor.Advice
	for _, item := range list {
		nodes, _, err := p.Parse(item.Text, charset, collation)
		if err != nil {
			// TiDB parser doesn't fully support MySQL syntax, so we need to use MySQL parser to parse the statement.
			// But MySQL parser has some performance issue, so we only use it to parse the statement after TiDB parser failed.
			if _, err := parser.ParseMySQL(item.Text); err != nil {
				if syntaxErr, ok := err.(*parser.SyntaxError); ok {
					return nil, []advisor.Advice{
						{
							Status:  advisor.Error,
							Code:    advisor.StatementSyntaxError,
							Title:   advisor.SyntaxErrorTitle,
							Content: syntaxErr.Message,
							Line:    syntaxErr.Line,
						},
					}
				}
				return nil, []advisor.Advice{
					{
						Status:  advisor.Warn,
						Code:    advisor.Internal,
						Title:   "Parse error",
						Content: err.Error(),
						Line:    1,
					},
				}
			}
			// If MySQL parser can parse the statement, but TiDB parser can't, we just ignore the statement.
			continue
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		node.SetText(nil, item.Text)
		node.SetOriginTextPosition(item.LastLine)
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, append(adviceList, advisor.Advice{
					Status:  advisor.Error,
					Code:    advisor.Internal,
					Title:   "Set line error",
					Content: err.Error(),
					Line:    item.LastLine,
				})
			}
		}
		returnNodes = append(returnNodes, node)
	}

	return returnNodes, adviceList
}
