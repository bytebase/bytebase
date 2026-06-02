// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForUpdateDeleteRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// WhereRequireForUpdateDeleteRule is the rule implementation for WHERE clause requirement in UPDATE/DELETE.
type WhereRequireForUpdateDeleteRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereRequireForUpdateDeleteRule creates a new WhereRequireForUpdateDeleteRule.
func NewWhereRequireForUpdateDeleteRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereRequireForUpdateDeleteRule {
	return &WhereRequireForUpdateDeleteRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereRequireForUpdateDeleteRule) Name() string {
	return "where.require-for-update-delete"
}

// OnStatement checks top-level UPDATE and DELETE statements in the omni AST.
func (r *WhereRequireForUpdateDeleteRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		if n.WhereClause == nil {
			r.AddAdvice(
				r.level,
				code.StatementNoWhere.Int32(),
				"WHERE clause is required for UPDATE statement.",
				common.ConvertANTLRLineToPosition(r.locLine(n.Loc)),
			)
		}
	case *ast.DeleteStmt:
		if n.WhereClause == nil {
			r.AddAdvice(
				r.level,
				code.StatementNoWhere.Int32(),
				"WHERE clause is required for DELETE statement.",
				common.ConvertANTLRLineToPosition(r.locLine(n.Loc)),
			)
		}
	case *ast.PLSQLBlock:
		r.checkPLSQLBlock(n)
	default:
	}
}

func (r *WhereRequireForUpdateDeleteRule) checkPLSQLBlock(block *ast.PLSQLBlock) {
	omniWalkPLSQLBlockStatements(block, func(stmt ast.StmtNode) bool {
		switch stmt.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			r.OnStatement(stmt)
			return false
		default:
			return true
		}
	})
}

// OnEnter is called when the parser enters a rule context.

// Ignore other node types

// OnExit is called when the parser exits a rule context.
