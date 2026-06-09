package oceanbase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/ast"

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
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for insert row limit.
type InsertRowLimitAdvisor struct {
}

// Check checks for insert row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	checker := &insertRowLimitChecker{
		level:  level,
		title:  checkCtx.Rule.Type.String(),
		maxRow: int(numberPayload.Number),
		driver: checkCtx.Driver,
		ctx:    ctx,
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		insert, ok := node.(*ast.InsertStmt)
		if !ok {
			continue
		}
		checker.checkInsert(stmt.Text, stmt.BaseLine(), insert)
		if checker.explainCount >= common.MaximumLintExplainSize {
			break
		}
	}
	return checker.generateAdvice()
}

type insertRowLimitChecker struct {
	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	maxRow       int
	driver       *sql.DB
	ctx          context.Context
	explainCount int
}

func (checker *insertRowLimitChecker) generateAdvice() ([]*storepb.Advice, error) {
	return checker.adviceList, nil
}

func (checker *insertRowLimitChecker) checkInsert(text string, baseLine int, stmt *ast.InsertStmt) {
	line := omniLine(baseLine, text, stmt.Loc)
	if stmt.Select != nil || stmt.TableSource != nil {
		checker.handleInsertQueryExpression(text, line)
	}
	checker.handleNoInsertQueryExpression(text, line, stmt)
}

func (checker *insertRowLimitChecker) handleInsertQueryExpression(text string, line int) {
	if checker.driver == nil {
		return
	}

	checker.explainCount++
	res, err := advisor.Query(checker.ctx, advisor.QueryContext{}, checker.driver, storepb.Engine_OCEANBASE, fmt.Sprintf("EXPLAIN format=json %s", text))
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
		return
	}
	rowCount, err := getEstimatedRowsFromJSON(res)
	if err != nil {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.Internal.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", text, err.Error()),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	} else if rowCount > int64(checker.maxRow) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, rowCount, checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}

func (checker *insertRowLimitChecker) handleNoInsertQueryExpression(text string, line int, stmt *ast.InsertStmt) {
	if len(stmt.Values) > checker.maxRow {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, len(stmt.Values), checker.maxRow),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}
