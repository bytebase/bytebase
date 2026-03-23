package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowOnDelCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE, &StatementDisallowOnDelCascadeAdvisor{})
}

// StatementDisallowOnDelCascadeAdvisor is the advisor checking for ON DELETE CASCADE.
type StatementDisallowOnDelCascadeAdvisor struct {
}

// Check checks for ON DELETE CASCADE.
func (*StatementDisallowOnDelCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowOnDelCascadeRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowOnDelCascadeRule struct {
	OmniBaseRule
}

func (*statementDisallowOnDelCascadeRule) Name() string {
	return "statement.disallow-on-delete-cascade"
}

func (r *statementDisallowOnDelCascadeRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.checkCreateStmt(n)
	case *ast.AlterTableStmt:
		r.checkAlterTableStmt(n)
	default:
	}
}

func (r *statementDisallowOnDelCascadeRule) checkCreateStmt(create *ast.CreateStmt) {
	_, constraints := omniTableElements(create)
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_FOREIGN && c.FkDelaction == 'c' {
			r.addCascadeAdvice()
			return
		}
	}
	// Also check column-level FK constraints.
	cols, _ := omniTableElements(create)
	for _, col := range cols {
		for _, c := range omniColumnConstraints(col) {
			if c.Contype == ast.CONSTR_FOREIGN && c.FkDelaction == 'c' {
				r.addCascadeAdvice()
				return
			}
		}
	}
}

func (r *statementDisallowOnDelCascadeRule) checkAlterTableStmt(alter *ast.AlterTableStmt) {
	for _, cmd := range omniAlterTableCmds(alter) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_AddConstraint {
			continue
		}
		constraint, ok := cmd.Def.(*ast.Constraint)
		if !ok {
			continue
		}
		if constraint.Contype == ast.CONSTR_FOREIGN && constraint.FkDelaction == 'c' {
			r.addCascadeAdvice()
			return
		}
	}
}

func (r *statementDisallowOnDelCascadeRule) addCascadeAdvice() {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementDisallowCascade.Int32(),
		Title:   r.Title,
		Content: "The CASCADE option is not permitted for ON DELETE clauses",
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
