package oceanbase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLStatementAffectedRowLimit, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE affected row limit.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &statementAffectedRowLimitChecker{
		level:  level,
		title:  string(checkCtx.Rule.Type),
		maxRow: payload.Number,
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	if checker.driver != nil {
		for _, stmt := range stmtList {
			checker.baseLine = stmt.BaseLine
			antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
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
			Code:          advisor.StatementAffectedRowExceedsLimit.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", checker.text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	} else {
		rowCount, err := getRowsFromJSON(res)
		if err != nil {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.Internal.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", checker.text, err.Error()),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		} else if rowCount > int64(checker.maxRow) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.StatementAffectedRowExceedsLimit.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\" affected %d rows (estimated). The count exceeds %d.", checker.text, rowCount, checker.maxRow),
				StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
			})
		}
	}
}

func getRowsFromJSON(res []any) (int64, error) {
	// For EXPLAIN format=json, OceanBase returns JSON data
	// The res struct is []any{columnName, columnTable, rowDataList}
	if len(res) < 3 {
		return 0, errors.Errorf("expected at least 3 elements but got %d", len(res))
	}

	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any for row data but got %T", res[2])
	}
	if len(rowList) == 0 {
		return 0, errors.Errorf("no data returned from EXPLAIN")
	}

	// OceanBase might return JSON data split across multiple elements
	// We need to concatenate them to form a valid JSON string
	var jsonStr string
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any for row but got %T", rowAny)
		}
		for _, cellAny := range row {
			if cell, ok := cellAny.(string); ok {
				jsonStr += cell
			}
		}
	}

	if jsonStr == "" {
		return 0, errors.Errorf("no JSON data found in EXPLAIN result")
	}

	var explainData ExplainJSON
	if err := json.Unmarshal([]byte(jsonStr), &explainData); err != nil {
		return 0, errors.Errorf("failed to parse JSON: %v", err)
	}

	// Return the estimated rows from the JSON response
	return explainData.EstRows, nil
}
