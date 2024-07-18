package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	wildcard = "%"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
	_ ast.Visitor     = (*noLeadingWildcardLikeChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
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
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.StatementLeadingWildcardLike.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" uses leading wildcard LIKE", checker.text),
				StartPosition: &storepb.Position{
					Line: int32(stmt.LastLine()),
				},
			})
		}
	}

	return checker.adviceList, nil
}

type noLeadingWildcardLikeChecker struct {
	adviceList          []*storepb.Advice
	level               storepb.Advice_Status
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
