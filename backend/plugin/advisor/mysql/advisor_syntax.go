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
			},
		}, nil
	}

	var adviceList []advisor.Advice
	for _, warn := range warns {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Warn,
			Code:    advisor.StatementSyntaxError,
			Title:   "Syntax Warning",
			Content: warn.Error(),
		})
	}

	adviceList = append(adviceList, advisor.Advice{
		Status:  advisor.Success,
		Code:    advisor.Ok,
		Title:   "Syntax OK",
		Content: "OK"})
	return adviceList, nil
}
