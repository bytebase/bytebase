package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMaximumJoinTableCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT, &StatementMaximumJoinTableCountAdvisor{})
}

type StatementMaximumJoinTableCountAdvisor struct {
}

func (*StatementMaximumJoinTableCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &maxJoinTableCountOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		limitMaxValue: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type maxJoinTableCountOmniRule struct {
	OmniBaseRule
	limitMaxValue int
}

func (*maxJoinTableCountOmniRule) Name() string {
	return "StatementMaximumJoinTableCountRule"
}

func (r *maxJoinTableCountOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelect(n, text)
	default:
	}
}

func (r *maxJoinTableCountOmniRule) checkSelect(sel *ast.SelectStmt, text string) {
	if sel == nil {
		return
	}
	if sel.SetOp != ast.SetOpNone {
		r.checkSelect(sel.Left, text)
		r.checkSelect(sel.Right, text)
		return
	}
	count := 0
	var lastJoinLoc ast.Loc
	for _, from := range sel.From {
		r.countJoins(from, &count, &lastJoinLoc)
	}
	if count >= r.limitMaxValue {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementMaximumJoinTableCount.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("\"%s\" exceeds the maximum number of joins %d.", text, r.limitMaxValue),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(lastJoinLoc))),
		})
	}
}

func (r *maxJoinTableCountOmniRule) countJoins(te ast.TableExpr, count *int, lastLoc *ast.Loc) {
	if te == nil {
		return
	}
	switch t := te.(type) {
	case *ast.JoinClause:
		*count++
		*lastLoc = t.Loc
		r.countJoins(t.Left, count, lastLoc)
		r.countJoins(t.Right, count, lastLoc)
	case *ast.SubqueryExpr:
		// Don't count joins in subqueries.
	default:
	}
}
