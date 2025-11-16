package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementCheckSetRoleVariable, &StatementCheckSetRoleVariable{})
}

type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &StatementCheckSetRoleVariableRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	if !rule.hasSetRole {
		rule.AddAdvice(&storepb.Advice{
			Status:        level,
			Code:          code.StatementCheckSetRoleVariable.Int32(),
			Title:         rule.title,
			Content:       "No SET ROLE statement found.",
			StartPosition: nil,
		})
	}

	return checker.GetAdviceList(), nil
}

type StatementCheckSetRoleVariableRule struct {
	BaseRule

	hasSetRole      bool
	foundNonSetStmt bool
}

// Name returns the rule name.
func (*StatementCheckSetRoleVariableRule) Name() string {
	return "statement.check-set-role-variable"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementCheckSetRoleVariableRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Variablesetstmt":
		r.handleVariablesetstmt(ctx.(*parser.VariablesetstmtContext))
	default:
		r.handleEveryRule(ctx)
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementCheckSetRoleVariableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleVariablesetstmt handles SET statements
func (r *StatementCheckSetRoleVariableRule) handleVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// If we already found a non-SET statement, skip this
	if r.foundNonSetStmt {
		return
	}

	// Check if this is a SET ROLE statement
	setRest := ctx.Set_rest()
	if setRest != nil {
		setRestMore := setRest.Set_rest_more()
		if setRestMore != nil && setRestMore.ROLE() != nil {
			r.hasSetRole = true
		}
	}
}

// handleEveryRule is called for every rule entry to detect non-SET statements
func (r *StatementCheckSetRoleVariableRule) handleEveryRule(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// If we already found a non-SET statement, no need to continue checking
	if r.foundNonSetStmt {
		return
	}

	// Check if this is a non-SET statement at the top level
	// We only care about statements that are not VariablesetstmtContext
	if _, isSetStmt := ctx.(*parser.VariablesetstmtContext); !isSetStmt {
		// Check if this is a statement node (not structural nodes like Stmt, Root, etc.)
		switch ctx.(type) {
		case *parser.RootContext, *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext:
			// These are structural nodes, not actual statements
			return
		default:
			// This is a non-SET statement
			r.foundNonSetStmt = true
		}
	}
}
