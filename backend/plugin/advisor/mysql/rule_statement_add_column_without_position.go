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
	_ advisor.Advisor = (*StatementAddColumnWithoutPositionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, &StatementAddColumnWithoutPositionAdvisor{})
}

// StatementAddColumnWithoutPositionAdvisor is the advisor checking for no position in ADD COLUMN clause.
type StatementAddColumnWithoutPositionAdvisor struct {
}

// Check checks for no position in ADD COLUMN clause.
func (*StatementAddColumnWithoutPositionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &addColumnWithoutPositionOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type addColumnWithoutPositionOmniRule struct {
	OmniBaseRule
}

func (*addColumnWithoutPositionOmniRule) Name() string {
	return "StatementAddColumnWithoutPositionRule"
}

func (r *addColumnWithoutPositionOmniRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}
	if alter.Table == nil {
		return
	}

	for _, cmd := range alter.Commands {
		if cmd.Type != ast.ATAddColumn {
			continue
		}
		var position string
		if cmd.First {
			position = "FIRST"
		} else if cmd.After != "" {
			position = "AFTER"
		}
		if position != "" {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementAddColumnWithPosition.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("add column with position \"%s\"", position),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
			})
		}
	}
}
