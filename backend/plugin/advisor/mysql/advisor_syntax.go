package mysql

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/parser"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLSyntax, &SyntaxAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor for checking syntax.
type SyntaxAdvisor struct {
}

// Check parses the given statement and checks for warnings and errors.
func (*SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	// Due to the limitation of TiDB parser, we should split the multi-statement into single statements, and extract
	// the TiDB unsupported statements, otherwise, the parser will panic or return the error.
	_, supportStmt, err := parser.ExtractTiDBUnsupportStmts(statement)
	if err != nil {
		//nolint:nilerr
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.StatementSyntaxError,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}
	p := newParser()

	_, warns, err := p.Parse(supportStmt, ctx.Charset, ctx.Collation)
	if err != nil {
		//nolint:nilerr
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.StatementSyntaxError,
				Title:   "Syntax error",
				Content: err.Error(),
				Line:    calculateErrorLine(supportStmt, ctx.Charset, ctx.Collation),
			},
		}, nil
	}

	var adviceList []advisor.Advice
	if len(warns) > 0 {
		adviceList = append(adviceList, getWarnWithLine(supportStmt, ctx.Charset, ctx.Collation)...)
	}

	adviceList = append(adviceList, advisor.Advice{
		Status:  advisor.Success,
		Code:    advisor.Ok,
		Title:   "Syntax OK",
		Content: "OK",
	})
	return adviceList, nil
}

func getWarnWithLine(statement string, charset string, collation string) []advisor.Advice {
	list, err := parser.SplitMultiSQL(parser.MySQL, statement)
	if err != nil {
		return []advisor.Advice{{
			Status:  advisor.Error,
			Code:    advisor.Internal,
			Title:   "Split multi-SQL error",
			Content: err.Error(),
			Line:    1,
		}}
	}

	var adviceList []advisor.Advice
	p := newParser()
	for _, stmt := range list {
		if _, warns, _ := p.Parse(stmt.Text, charset, collation); len(warns) > 0 {
			for _, warn := range warns {
				adviceList = append(adviceList, advisor.Advice{
					Status:  advisor.Warn,
					Code:    advisor.StatementSyntaxError,
					Title:   "Syntax Warning",
					Content: warn.Error(),
					Line:    stmt.LastLine,
				})
			}
		}
	}

	return adviceList
}

func calculateErrorLine(statement string, charset string, collation string) int {
	list, err := parser.SplitMultiSQL(parser.MySQL, statement)
	if err != nil {
		//nolint:nilerr
		return 1
	}

	p := newParser()
	for _, stmt := range list {
		if _, _, err := p.Parse(stmt.Text, charset, collation); err != nil {
			return stmt.LastLine
		}
	}

	return 0
}
