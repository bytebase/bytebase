package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

const (
	wildcard string = "%"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (adv *NoLeadingWildcardLikeAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noLeadingWildcardLikeChecker{level: level}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.leadingWildcardLike = false
		(stmtNode).Accept(checker)

		if checker.leadingWildcardLike {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    common.StatementLeadingWildcardLike,
				Title:   "No leading wildcard LIKE",
				Content: fmt.Sprintf("%q uses leading wildcard LIKE", checker.text),
			})
		}
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil

}

type noLeadingWildcardLikeChecker struct {
	adviceList          []advisor.Advice
	level               advisor.Status
	text                string
	leadingWildcardLike bool
}

// Enter implements the ast.Visitor interface
func (v *noLeadingWildcardLikeChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.PatternLikeExpr); !v.leadingWildcardLike && ok {
		pattern, err := restoreNode(node.Pattern, format.RestoreStringWithoutCharset)
		if err != nil {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    common.Internal,
				Title:   "Internal error for no leading wildcard LIKE rule",
				Content: fmt.Sprintf("%q meet internal error %q", v.text, err.Error()),
			})
		}
		if len(pattern) > 0 && pattern[:1] == wildcard {
			v.leadingWildcardLike = true
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *noLeadingWildcardLikeChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
