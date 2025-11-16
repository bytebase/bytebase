package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleStatementDMLDryRun, &StatementDmlDryRunAdvisor{})
}

type StatementDmlDryRunAdvisor struct {
}

func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsql.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewStatementDmlDryRunRule(ctx, level, string(checkCtx.Rule.Type), checkCtx.Driver)
	checker := NewGenericChecker([]Rule{rule})

	if checkCtx.Driver != nil {
		for _, stmtNode := range stmtList {
			rule.SetBaseLine(stmtNode.BaseLine)
			checker.SetBaseLine(stmtNode.BaseLine)
			antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
		}
	}

	return checker.GetAdviceList()
}

// StatementDmlDryRunRule is the rule implementation for DML dry run checks.
type StatementDmlDryRunRule struct {
	BaseRule

	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

// NewStatementDmlDryRunRule creates a new StatementDmlDryRunRule.
func NewStatementDmlDryRunRule(ctx context.Context, level storepb.Advice_Status, title string, driver *sql.DB) *StatementDmlDryRunRule {
	return &StatementDmlDryRunRule{
		BaseRule: NewBaseRule(level, title, 0),
		driver:   driver,
		ctx:      ctx,
	}
}

// Name returns the rule name.
func (*StatementDmlDryRunRule) Name() string {
	return "statement.dml-dry-run"
}

// OnEnter is called when the parser enters a rule context.
func (r *StatementDmlDryRunRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Insert_statement":
		r.handleInsertStatement(ctx.(*parser.Insert_statementContext))
	case "Update_statement":
		r.handleUpdateStatement(ctx.(*parser.Update_statementContext))
	case "Delete_statement":
		r.handleDeleteStatement(ctx.(*parser.Delete_statementContext))
	case "Merge_statement":
		r.handleMergeStatement(ctx.(*parser.Merge_statementContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*StatementDmlDryRunRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementDmlDryRunRule) handleInsertStatement(ctx *parser.Insert_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		r.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), r.baseLine+ctx.GetStart().GetLine())
	}
}

func (r *StatementDmlDryRunRule) handleUpdateStatement(ctx *parser.Update_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		r.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), r.baseLine+ctx.GetStart().GetLine())
	}
}

func (r *StatementDmlDryRunRule) handleDeleteStatement(ctx *parser.Delete_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		r.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), r.baseLine+ctx.GetStart().GetLine())
	}
}

func (r *StatementDmlDryRunRule) handleMergeStatement(ctx *parser.Merge_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		r.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), r.baseLine+ctx.GetStart().GetLine())
	}
}

func (r *StatementDmlDryRunRule) handleStmt(text string, lineNumber int) {
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}
	r.explainCount++
	if _, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_ORACLE, fmt.Sprintf("EXPLAIN PLAN FOR %s", text)); err != nil {
		r.AddAdvice(
			r.level,
			code.StatementDMLDryRunFailed.Int32(),
			fmt.Sprintf("Failed to dry run statement at line %d: %v", lineNumber, err),
			common.ConvertANTLRLineToPosition(lineNumber),
		)
	}
}
