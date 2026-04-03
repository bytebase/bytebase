package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &noSelectAllOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type noSelectAllOmniRule struct {
	OmniBaseRule
}

func (*noSelectAllOmniRule) Name() string {
	return "NoSelectAllRule"
}

func (r *noSelectAllOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok {
			return true
		}
		// Only check TargetList on leaf SelectStmt (not UNION nodes).
		if sel.SetOp != ast.SetOpNone {
			return true
		}
		for _, target := range sel.TargetList {
			if r.isStarTarget(target) {
				r.AddAdviceAbsolute(&storepb.Advice{
					Status:        r.Level,
					Code:          code.StatementSelectAll.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("\"%s\" uses SELECT all", text),
					StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.findStarLine(target))),
				})
			}
		}
		// Recurse into children (CTEs, subqueries in FROM, etc.) via Inspect.
		return true
	})
}

func (*noSelectAllOmniRule) isStarTarget(expr ast.ExprNode) bool {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return true
	case *ast.ResTarget:
		if _, ok := e.Val.(*ast.StarExpr); ok {
			return true
		}
	default:
	}
	return false
}

func (r *noSelectAllOmniRule) findStarLine(expr ast.ExprNode) int32 {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return r.LocToLine(e.Loc)
	case *ast.ResTarget:
		if star, ok := e.Val.(*ast.StarExpr); ok {
			return r.LocToLine(star.Loc)
		}
		return r.LocToLine(e.Loc)
	default:
		return r.ContentStartLine()
	}
}
