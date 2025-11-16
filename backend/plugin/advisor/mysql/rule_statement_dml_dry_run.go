package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementDMLDryRun, &StatementDmlDryRunAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementDMLDryRun, &StatementDmlDryRunAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementDMLDryRun, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run.
type StatementDmlDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementDmlDryRunRule(ctx, level, string(checkCtx.Rule.Type), checkCtx.Driver)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	if checkCtx.Driver != nil {
		for _, stmt := range stmtList {
			rule.SetBaseLine(stmt.BaseLine)
			checker.SetBaseLine(stmt.BaseLine)
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
			if rule.explainCount >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return checker.GetAdviceList(), nil
}

// StatementDmlDryRunRule checks for DML dry run.
type StatementDmlDryRunRule struct {
	BaseRule
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

// NewStatementDmlDryRunRule creates a new StatementDmlDryRunRule.
func NewStatementDmlDryRunRule(ctx context.Context, level storepb.Advice_Status, title string, driver *sql.DB) *StatementDmlDryRunRule {
	return &StatementDmlDryRunRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		driver: driver,
		ctx:    ctx,
	}
}

// Name returns the rule name.
func (*StatementDmlDryRunRule) Name() string {
	return "StatementDmlDryRunRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDmlDryRunRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeUpdateStatement:
		updateCtx, ok := ctx.(*mysql.UpdateStatementContext)
		if !ok {
			return nil
		}
		if mysqlparser.IsTopMySQLRule(&updateCtx.BaseParserRuleContext) {
			r.handleStmt(updateCtx.GetParser().GetTokenStream().GetTextFromRuleContext(updateCtx), updateCtx.GetStart().GetLine())
		}
	case NodeTypeDeleteStatement:
		deleteCtx, ok := ctx.(*mysql.DeleteStatementContext)
		if !ok {
			return nil
		}
		if mysqlparser.IsTopMySQLRule(&deleteCtx.BaseParserRuleContext) {
			r.handleStmt(deleteCtx.GetParser().GetTokenStream().GetTextFromRuleContext(deleteCtx), deleteCtx.GetStart().GetLine())
		}
	case NodeTypeInsertStatement:
		insertCtx, ok := ctx.(*mysql.InsertStatementContext)
		if !ok {
			return nil
		}
		if mysqlparser.IsTopMySQLRule(&insertCtx.BaseParserRuleContext) {
			r.handleStmt(insertCtx.GetParser().GetTokenStream().GetTextFromRuleContext(insertCtx), insertCtx.GetStart().GetLine())
		}
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementDmlDryRunRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementDmlDryRunRule) handleStmt(text string, lineNumber int) {
	r.explainCount++
	if _, err := advisor.Query(r.ctx, advisor.QueryContext{}, r.driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", text)); err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementDMLDryRunFailed.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
		})
	}
}
