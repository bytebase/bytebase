// Package snowflake is the advisor for Snowflake database.
package snowflake

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*SyntaxAdvisor)(nil)
)

func init() {
	advisor.Register(db.Snowflake, advisor.SnowflakeSyntax, &SyntaxAdvisor{})
}

// SyntaxAdvisor is the advisor checking for syntax.
type SyntaxAdvisor struct {
}

// Check checks for syntax.
func (*SyntaxAdvisor) Check(_ advisor.Context, statement string) ([]advisor.Advice, error) {
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
