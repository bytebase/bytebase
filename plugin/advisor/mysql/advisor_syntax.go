package mysql

import (
	"github.com/bytebase/bytebase/plugin/advisor"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.MySQL, advisor.MySQLSyntax, &SyntaxAdvisor{})
	advisor.Register(advisor.TiDB, advisor.MySQLSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor for checking syntax.
type SyntaxAdvisor struct {
}

// Check parses the given statement and checks for warnings and errors.
func (adv *SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	p := newParser()

	_, warns, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
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
