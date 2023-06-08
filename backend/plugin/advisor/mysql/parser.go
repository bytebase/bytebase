package mysql

import (
	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"

	mysqlparser "github.com/bytebase/mysql-parser"

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
	tree, tokens, err := parser.ParseMySQL(statement)
	if err != nil {
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
	if tree == nil {
		return nil, nil
	}

	var returnNodes []ast.StmtNode
	var adviceList []advisor.Advice
	p := newParser()
	for _, child := range tree.GetChildren() {
		if child == nil {
			continue
		}

		if query, ok := child.(mysqlparser.IQueryContext); ok {
			text := tokens.GetTextFromRuleContext(query)
			lastLine := query.GetStop().GetLine()
			if nodes, _, err := p.Parse(text, charset, collation); err == nil {
				if len(nodes) != 1 {
					continue
				}
				node := nodes[0]
				node.SetText(nil, text)
				node.SetOriginTextPosition(lastLine)
				if n, ok := node.(*ast.CreateTableStmt); ok {
					if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
						return nil, append(adviceList, advisor.Advice{
							Status:  advisor.Error,
							Code:    advisor.Internal,
							Title:   "Set line error",
							Content: err.Error(),
							Line:    lastLine,
						})
					}
				}
				returnNodes = append(returnNodes, node)
			}
		}
	}

	return returnNodes, adviceList
}
