package mysql

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLSyntax, &SyntaxAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLSyntax, &SyntaxAdvisor{})
	advisor.Register(db.MariaDB, advisor.MySQLSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor for checking syntax.
type SyntaxAdvisor struct {
}

// Check parses the given statement and checks for warnings and errors.
func (*SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	_, adviceList := parseStatement(statement, ctx.Charset, ctx.Collation)

	if len(adviceList) == 0 {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "Syntax OK",
			Content: "OK",
		})
	}

	return adviceList, nil
}
