package mysql

import (
	"fmt"
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser"
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
	p := newParser()

	root, _, err := p.Parse(statement, charset, collation)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.StatementSyntaxError,
				Title:   advisor.SyntaxErrorTitle,
				Content: err.Error(),
			},
		}
	}

	// sikp the setting line stage
	if len(root) == 0 {
		return root, nil
	}

	// setting line stage
	sqlList, err := parser.SplitMultiSQL(parser.MySQL, statement, true /* filterEmptyStatement */)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.Internal,
				Title:   "Split multi-SQL error",
				Content: err.Error(),
			},
		}
	}
	if len(sqlList) != len(root) {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.Internal,
				Title:   "Split multi-SQL error",
				Content: fmt.Sprintf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(root), len(sqlList), statement),
			},
		}
	}

	for i, node := range root {
		node.SetText(nil, strings.TrimSpace(node.Text()))
		node.SetOriginTextPosition(sqlList[i].LastLine)
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, []advisor.Advice{
					{
						Status:  advisor.Error,
						Code:    advisor.Internal,
						Title:   "Set line error",
						Content: err.Error(),
					},
				}
			}
		}
	}
	return root, nil
}
