package tidb

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

const (
	wildcard string = "%"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
	_ ast.Visitor     = (*noLeadingWildcardLikeChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleStatementNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noLeadingWildcardLikeChecker{level: level}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.leadingWildcardLike = false
		(stmtNode).Accept(checker)

		if checker.leadingWildcardLike {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.StatementLeadingWildcardLike.Int32(),
				Title:         string(checkCtx.Rule.Type),
				Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", checker.text),
				StartPosition: common.ConvertANTLRLineToPosition(stmtNode.OriginTextPosition()),
			})
		}
	}

	return checker.adviceList, nil
}

type noLeadingWildcardLikeChecker struct {
	adviceList          []*storepb.Advice
	level               storepb.Advice_Status
	text                string
	leadingWildcardLike bool
}

// Enter implements the ast.Visitor interface.
func (v *noLeadingWildcardLikeChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.PatternLikeOrIlikeExpr); !v.leadingWildcardLike && ok {
		pattern, err := restoreNode(node.Pattern, format.RestoreStringWithoutCharset)
		if err != nil {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:  v.level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for no leading wildcard LIKE rule",
				Content: fmt.Sprintf("\"%s\" meet internal error %q", v.text, err.Error()),
			})
		}
		if len(pattern) > 0 && pattern[:1] == wildcard {
			v.leadingWildcardLike = true
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*noLeadingWildcardLikeChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
