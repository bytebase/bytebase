package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &insertMustSpecifyColumnRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type insertMustSpecifyColumnRule struct {
	OmniBaseRule
}

func (*insertMustSpecifyColumnRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN)
}

func (r *insertMustSpecifyColumnRule) OnStatement(node ast.Node) {
	ins, ok := node.(*ast.InsertStmt)
	if !ok {
		return
	}

	if ins.Cols == nil || len(ins.Cols.Items) == 0 {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.InsertNotSpecifyColumn.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", r.TrimmedStmtText()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
