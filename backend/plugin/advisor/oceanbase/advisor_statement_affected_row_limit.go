package oceanbase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	checker := &statementAffectedRowLimitChecker{
		level:  level,
		title:  checkCtx.Rule.Type.String(),
		maxRow: int(numberPayload.Number),
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	if checker.driver != nil {
		for _, stmt := range checkCtx.ParsedStatements {
			if stmt.AST == nil {
				continue
			}
			antlrAST, ok := base.GetANTLRAST(stmt.AST)
			if !ok {
				continue
			}
			checker.baseLine = stmt.BaseLine()
			antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
			if checker.explainCount >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return checker.adviceList, nil
}

type statementAffectedRowLimitChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine     int
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	text         string
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

func (checker *statementAffectedRowLimitChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *statementAffectedRowLimitChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.handleStmt(ctx.GetStart().GetLine())
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *statementAffectedRowLimitChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.handleStmt(ctx.GetStart().GetLine())
}

func (checker *statementAffectedRowLimitChecker) handleStmt(lineNumber int) {
	lineNumber += checker.baseLine
	checker.explainCount++
	res, err := advisor.Query(checker.ctx, advisor.QueryContext{}, checker.driver, storepb.Engine_OCEANBASE, fmt.Sprintf("EXPLAIN format=json %s", checker.text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.StatementAffectedRowExceedsLimit.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	} else {
		rowCount, err := getEstimatedRowsFromJSON(res)
		if err != nil {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.Internal.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		} else if rowCount > int64(checker.maxRow) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\" affected %d rows (estimated). The count exceeds %d.", checker.text, rowCount, checker.maxRow),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		}
	}
}
