package oceanbase

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
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementInsertRowLimit, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for insert row limit.
type InsertRowLimitAdvisor struct {
}

// Check checks for insert row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &insertRowLimitChecker{
		level:  level,
		title:  string(checkCtx.Rule.Type),
		maxRow: payload.Number,
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		if checker.explainCount >= common.MaximumLintExplainSize {
			break
		}
	}
	return checker.generateAdvice()
}

type insertRowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	text         string
	line         int
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

func (checker *insertRowLimitChecker) generateAdvice() ([]*storepb.Advice, error) {
	return checker.adviceList, nil
}

func (checker *insertRowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *insertRowLimitChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.line = checker.baseLine + ctx.GetStart().GetLine()
	if ctx.InsertQueryExpression() != nil {
		checker.handleInsertQueryExpression(ctx.InsertQueryExpression())
	}
	checker.handleNoInsertQueryExpression(ctx)
}

func (checker *insertRowLimitChecker) handleInsertQueryExpression(ctx mysql.IInsertQueryExpressionContext) {
	if checker.driver == nil || ctx == nil {
		return
	}

	checker.explainCount++
	res, err := advisor.Query(checker.ctx, advisor.QueryContext{}, checker.driver, storepb.Engine_OCEANBASE, fmt.Sprintf("EXPLAIN format=json %s", checker.text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
		return
	}
	rowCount, err := getEstimatedRowsFromJSON(res)
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.Internal.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
	} else if rowCount > int64(checker.maxRow) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, rowCount, checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
	}
}

func (checker *insertRowLimitChecker) handleNoInsertQueryExpression(ctx mysql.IInsertStatementContext) {
	if ctx.InsertFromConstructor() == nil {
		return
	}
	if ctx.InsertFromConstructor().InsertValues() == nil {
		return
	}
	if ctx.InsertFromConstructor().InsertValues().ValueList() == nil {
		return
	}

	allValues := ctx.InsertFromConstructor().InsertValues().ValueList().AllValues()
	if len(allValues) > checker.maxRow {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", checker.text, len(allValues), checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(checker.line),
		})
	}
}
