package mysql

import (
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
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
func (adv *SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	p := newParser()

	_, warns, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    common.DbStatementSyntaxError,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}

	var adviceList []advisor.Advice
	for _, warn := range warns {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Warn,
			Code:    common.DbStatementSyntaxError,
			Title:   "Syntax Warning",
			Content: warn.Error(),
		})
	}

	adviceList = append(adviceList, advisor.Advice{
		Status:  advisor.Success,
		Code:    common.Ok,
		Title:   "Syntax OK",
		Content: "OK"})
	return adviceList, nil
}
