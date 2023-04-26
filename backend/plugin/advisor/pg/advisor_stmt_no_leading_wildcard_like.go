package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

const (
	wildcard = "%"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
	_ ast.Visitor     = (*noLeadingWildcardLikeChecker)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noLeadingWildcardLikeChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.leadingWildcardLike = false
		ast.Walk(checker, stmt)

		if checker.leadingWildcardLike {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementLeadingWildcardLike,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" uses leading wildcard LIKE", checker.text),
				Line:    stmt.LastLine(),
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type noLeadingWildcardLikeChecker struct {
	adviceList          []advisor.Advice
	level               advisor.Status
	title               string
	text                string
	leadingWildcardLike bool
}

// Visit implements the ast.Visitor interface.
func (checker *noLeadingWildcardLikeChecker) Visit(node ast.Node) ast.Visitor {
	if n, ok := node.(*ast.PatternLikeDef); !checker.leadingWildcardLike && ok {
		if pattern, ok := n.Pattern.(*ast.StringDef); ok && len(pattern.Value) > 0 && pattern.Value[:1] == wildcard {
			checker.leadingWildcardLike = true
		}
	}
	return checker
}
