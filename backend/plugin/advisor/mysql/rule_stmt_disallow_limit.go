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
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE/DELETE statement.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE/DELETE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowLimitOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowLimitOmniRule struct {
	OmniBaseRule
}

func (*disallowLimitOmniRule) Name() string {
	return "StatementDisallowLimitRule"
}

func (r *disallowLimitOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	switch n := node.(type) {
	case *ast.DeleteStmt:
		if n.Limit != nil {
			r.addLimitAdvice(code.DeleteUseLimit, text, n.Loc)
		}
	case *ast.UpdateStmt:
		if n.Limit != nil {
			r.addLimitAdvice(code.UpdateUseLimit, text, n.Loc)
		}
	case *ast.InsertStmt:
		if n.Select != nil {
			r.checkSelectLimit(n.Select, text)
		}
	default:
	}
}

func (r *disallowLimitOmniRule) checkSelectLimit(sel *ast.SelectStmt, text string) {
	if sel == nil {
		return
	}
	if sel.SetOp != ast.SetOpNone {
		// Check set operation top-level limit.
		if sel.Limit != nil {
			r.addLimitAdvice(code.InsertUseLimit, text, sel.Loc)
		}
		return
	}
	if sel.Limit != nil {
		r.addLimitAdvice(code.InsertUseLimit, text, sel.Loc)
	}
}

func (r *disallowLimitOmniRule) addLimitAdvice(c code.Code, text string, _ ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          c.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", text),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
	})
}
