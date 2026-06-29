// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForSelectRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// WhereRequireForSelectRule is the rule implementation for WHERE clause requirement in SELECT.
type WhereRequireForSelectRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereRequireForSelectRule creates a new WhereRequireForSelectRule.
func NewWhereRequireForSelectRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereRequireForSelectRule {
	return &WhereRequireForSelectRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereRequireForSelectRule) Name() string {
	return "where.require-for-select"
}

// OnStatement checks SELECT statements with FROM clauses in the omni AST.
func (r *WhereRequireForSelectRule) OnStatement(node ast.Node) {
	omniWalk(node, func(n ast.Node) {
		selectStmt, ok := n.(*ast.SelectStmt)
		if !ok {
			return
		}
		if selectStmt.FromClause == nil || len(selectStmt.FromClause.Items) == 0 {
			return
		}
		if len(selectStmt.FromClause.Items) == 1 {
			if table, ok := selectStmt.FromClause.Items[0].(*ast.TableRef); ok && table.Name != nil && strings.EqualFold(table.Name.Name, "DUAL") {
				return
			}
		}
		if selectStmt.WhereClause == nil {
			r.AddAdvice(
				r.level,
				code.StatementNoWhere.Int32(),
				"WHERE clause is required for SELECT statement.",
				common.ConvertANTLRLineToPosition(r.locLine(selectStmt.Loc)),
			)
		}
	})
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.

// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
