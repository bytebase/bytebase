package pg

import (
	"github.com/bytebase/bytebase/plugin/advisor"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.Postgres, advisor.PostgreSQLSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor for checking syntax.
type SyntaxAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (adv *SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	var res []advisor.Advice
	if _, errAdvice := parseStatement(statement); errAdvice != nil {
		for _, advice := range errAdvice {
			if advice.Code == advisor.StatementSyntaxError {
				res = append(res, advice)
			}
		}
	}

	if len(res) == 0 {
		res = append(res, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "Syntax OK",
			Content: "OK",
		})
	}

	return res, nil
}
