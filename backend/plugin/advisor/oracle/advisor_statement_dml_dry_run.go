package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleStatementDMLDryRun, &StatementDmlDryRunAdvisor{})
}

type StatementDmlDryRunAdvisor struct {
}

func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDmlDryRunChecker{
		level:  level,
		title:  string(checkCtx.Rule.Type),
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	if checker.driver != nil {
		antlr.ParseTreeWalkerDefault.Walk(checker, tree)
	}

	return checker.adviceList, nil
}

type statementDmlDryRunChecker struct {
	*parser.BasePlSqlParserListener

	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

func (s *statementDmlDryRunChecker) EnterInsert_statement(ctx *parser.Insert_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		s.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), ctx.GetStart().GetLine())
	}
}

func (s *statementDmlDryRunChecker) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		s.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), ctx.GetStart().GetLine())
	}
}

func (s *statementDmlDryRunChecker) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		s.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), ctx.GetStart().GetLine())
	}
}

func (s *statementDmlDryRunChecker) EnterMerge_statement(ctx *parser.Merge_statementContext) {
	if plsql.IsTopLevelStatement(ctx.GetParent()) {
		s.handleStmt(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx), ctx.GetStart().GetLine())
	}
}

func (s *statementDmlDryRunChecker) handleStmt(text string, lineNumber int) {
	if s.explainCount >= common.MaximumLintExplainSize {
		return
	}
	s.explainCount++
	if _, err := advisor.Query(s.ctx, advisor.QueryContext{}, s.driver, storepb.Engine_ORACLE, fmt.Sprintf("EXPLAIN PLAN FOR %s", text)); err != nil {
		s.adviceList = append(s.adviceList, &storepb.Advice{
			Status:        s.level,
			Code:          advisor.StatementDMLDryRunFailed.Int32(),
			Title:         s.title,
			Content:       fmt.Sprintf("Failed to dry run statement at line %d: %v", lineNumber, err),
			StartPosition: advisor.ConvertANTLRLineToPosition(lineNumber),
		})
	}
}
