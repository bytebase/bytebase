package mysql

import (
	"fmt"
	"strings"

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
	list, err := parser.SplitMultiSQLAndNormalize(parser.MySQL, statement)
	if err != nil {
		return nil, []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.Internal,
				Title:   "Split multi-SQL error",
				Content: err.Error(),
				Line:    1,
			},
		}
	}

	var result []ast.StmtNode
	var adviceList []advisor.Advice
	p := newParser()
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		if parser.IsTiDBUnsupportDDLStmt(sql.Text) {
			// skip the TiDB parser unsupported DDL statement
			continue
		}

		nodes, warns, err := p.Parse(sql.Text, charset, collation)
		if err != nil {
			return nil, []advisor.Advice{
				{
					Status:  advisor.Error,
					Code:    advisor.StatementSyntaxError,
					Title:   advisor.SyntaxErrorTitle,
					Content: err.Error(),
					Line:    sql.LastLine,
				},
			}
		}
		for _, warn := range warns {
			adviceList = append(adviceList, advisor.Advice{
				Status:  advisor.Warn,
				Code:    advisor.StatementSyntaxError,
				Title:   "Syntax Warning",
				Content: warn.Error(),
				Line:    sql.LastLine,
			})
		}
		if len(nodes) == 0 {
			continue
		}
		if len(nodes) > 1 {
			return nil, append(adviceList, advisor.Advice{
				Status:  advisor.Error,
				Code:    advisor.Internal,
				Title:   "Split multi-SQL error",
				Content: fmt.Sprintf("get more than one sql after split: %s", sql.Text),
				Line:    sql.LastLine,
			})
		}
		node := nodes[0]
		node.SetText(nil, strings.TrimSpace(node.Text()))
		node.SetOriginTextPosition(sql.LastLine)
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, append(adviceList, advisor.Advice{
					Status:  advisor.Error,
					Code:    advisor.Internal,
					Title:   "Set line error",
					Content: err.Error(),
					Line:    sql.LastLine,
				})
			}
		}
		result = append(result, node)
	}

	return result, adviceList
}
