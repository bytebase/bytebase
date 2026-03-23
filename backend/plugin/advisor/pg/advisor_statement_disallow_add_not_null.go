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
	_ advisor.Advisor = (*StatementDisallowAddNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL, &StatementDisallowAddNotNullAdvisor{})
}

// StatementDisallowAddNotNullAdvisor is the advisor checking for to disallow add not null.
type StatementDisallowAddNotNullAdvisor struct {
}

// Check checks for to disallow add not null.
func (*StatementDisallowAddNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowAddNotNullRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowAddNotNullRule struct {
	OmniBaseRule
}

func (*statementDisallowAddNotNullRule) Name() string {
	return "statement_disallow_add_not_null"
}

func (r *statementDisallowAddNotNullRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range omniAlterTableCmds(alter) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_SetNotNull {
			continue
		}
		columnName := cmd.Name
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementAddNotNull.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", columnName),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
