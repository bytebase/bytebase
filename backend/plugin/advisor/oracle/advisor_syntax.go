// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor checking for syntax.
type SyntaxAdvisor struct {
}

// Check checks for syntax.
func (*SyntaxAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	if _, errAdvice := parseStatement(statement); errAdvice != nil {
		return errAdvice, nil
	}

	return []advisor.Advice{{
		Status:  advisor.Success,
		Code:    advisor.Ok,
		Title:   "Syntax OK",
		Content: "OK",
	}}, nil
}
