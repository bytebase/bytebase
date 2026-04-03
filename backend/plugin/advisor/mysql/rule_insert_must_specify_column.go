package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
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

	rule := &insertMustSpecifyColumnOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type insertMustSpecifyColumnOmniRule struct {
	OmniBaseRule
}

func (*insertMustSpecifyColumnOmniRule) Name() string {
	return "InsertMustSpecifyColumnRule"
}

func (r *insertMustSpecifyColumnOmniRule) OnStatement(node ast.Node) {
	ins, ok := node.(*ast.InsertStmt)
	if !ok {
		return
	}

	text := r.TrimmedStmtText() + ";"

	// INSERT ... SELECT without columns.
	if ins.Select != nil && len(ins.Columns) == 0 {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.InsertNotSpecifyColumn.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", text),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
		})
		return
	}

	// INSERT ... VALUES without columns.
	if len(ins.Values) > 0 && len(ins.Columns) == 0 {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.InsertNotSpecifyColumn.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", text),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
		})
	}
}
