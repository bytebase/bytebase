package pg

import (
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor for checking syntax.
type SyntaxAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (adv *SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	if _, errAdvice := parseStatement(statement); errAdvice != nil {
		return errAdvice, nil
	}

	return []advisor.Advice{
		{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "Syntax OK",
			Content: "OK",
		},
	}, nil
}
