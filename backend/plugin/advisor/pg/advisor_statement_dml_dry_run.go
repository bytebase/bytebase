package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDMLDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDMLDryRun, &StatementDMLDryRunAdvisor{})
}

// StatementDMLDryRunAdvisor is the advisor checking for DML dry run.
type StatementDMLDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDMLDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDMLDryRunRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		ctx:                      ctx,
		driver:                   checkCtx.Driver,
		usePostgresDatabaseOwner: checkCtx.UsePostgresDatabaseOwner,
		statementsText:           checkCtx.Statements,
	}

	// Only run EXPLAIN queries if we have a database connection
	if rule.driver != nil {
		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)
		return checker.GetAdviceList(), nil
	}

	return nil, nil
}

type statementDMLDryRunRule struct {
	BaseRule
	driver                   *sql.DB
	ctx                      context.Context
	explainCount             int
	setRoles                 []string
	usePostgresDatabaseOwner bool
	statementsText           string
}

// Name returns the rule name.
func (*statementDMLDryRunRule) Name() string {
	return "statement.dml-dry-run"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementDMLDryRunRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Variablesetstmt":
		r.handleVariablesetstmt(ctx.(*parser.VariablesetstmtContext))
	case "Insertstmt":
		r.handleInsertstmt(ctx.(*parser.InsertstmtContext))
	case "Updatestmt":
		r.handleUpdatestmt(ctx.(*parser.UpdatestmtContext))
	case "Deletestmt":
		r.handleDeletestmt(ctx.(*parser.DeletestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementDMLDryRunRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementDMLDryRunRule) handleVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is SET ROLE
	if ctx.SET() != nil && ctx.Set_rest() != nil && ctx.Set_rest().Set_rest_more() != nil {
		setRestMore := ctx.Set_rest().Set_rest_more()
		if setRestMore.ROLE() != nil {
			// Store the SET ROLE statement text
			stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			r.setRoles = append(r.setRoles, stmtText)
		}
	}
}

func (r *statementDMLDryRunRule) handleInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkDMLDryRun(ctx)
}

func (r *statementDMLDryRunRule) handleUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkDMLDryRun(ctx)
}

func (r *statementDMLDryRunRule) handleDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkDMLDryRun(ctx)
}

func (r *statementDMLDryRunRule) checkDMLDryRun(ctx antlr.ParserRuleContext) {
	// Check if we've hit the maximum number of EXPLAIN queries
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}

	r.explainCount++

	// Get the statement text
	stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	normalizedStmt := advisor.NormalizeStatement(stmtText)

	// Run EXPLAIN to perform dry run
	_, err := advisor.Query(r.ctx, advisor.QueryContext{
		UsePostgresDatabaseOwner: r.usePostgresDatabaseOwner,
		PreExecutions:            r.setRoles,
	}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", stmtText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementDMLDryRunFailed.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", normalizedStmt, err.Error()),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
