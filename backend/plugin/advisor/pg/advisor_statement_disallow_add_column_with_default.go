package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowAddColumnWithDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT, &StatementDisallowAddColumnWithDefaultAdvisor{})
}

// StatementDisallowAddColumnWithDefaultAdvisor is the advisor checking for to disallow add column with default.
type StatementDisallowAddColumnWithDefaultAdvisor struct {
}

// Check checks for to disallow add column with default.
func (*StatementDisallowAddColumnWithDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowAddColumnWithDefaultRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowAddColumnWithDefaultRule struct {
	OmniBaseRule
}

func (*statementDisallowAddColumnWithDefaultRule) Name() string {
	return "statement.disallow-add-column-with-default"
}

func (r *statementDisallowAddColumnWithDefaultRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range omniAlterTableCmds(alter) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_AddColumn {
			continue
		}
		colDef, ok := cmd.Def.(*ast.ColumnDef)
		if !ok {
			continue
		}
		if colDef.RawDefault != nil {
			r.addAdvice()
			return
		}
		for _, c := range omniColumnConstraints(colDef) {
			if c.Contype == ast.CONSTR_DEFAULT {
				r.addAdvice()
				return
			}
		}
	}
}

func (r *statementDisallowAddColumnWithDefaultRule) addAdvice() {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementAddColumnWithDefault.Int32(),
		Title:   r.Title,
		Content: "Adding column with DEFAULT will locked the whole table and rewriting each rows",
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
