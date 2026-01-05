package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE, &StatementCheckSetRoleVariable{})
}

type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &StatementCheckSetRoleVariableRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

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
