package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
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

	maxRow := int(numberPayload.Number)
	driver := checkCtx.Driver
	title := checkCtx.Rule.Type.String()
	var advice []*storepb.Advice
	var explains advisor.ExplainBudget

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		ins, ok := node.(*ast.InsertStmt)
		if !ok {
			continue
		}

		baseLine := stmt.BaseLine()
		text := strings.TrimRight(strings.TrimSpace(stmt.Text), ";") + ";"
		position := common.ConvertANTLRLineToPosition(baseLine + int(mysqlparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line))

		// INSERT ... SELECT and INSERT ... TABLE: use EXPLAIN to count rows.
		if ins.Select != nil || ins.TableSource != nil {
			if driver == nil {
				continue
			}
			if !explains.Spend(position) {
				continue
			}

			plan, err := mysqldriver.ExplainJSON(ctx, driver, text)
			if err != nil {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.InsertTooManyRows.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
					StartPosition: position,
				})
				continue
			}
			rowCount, err := mysqldriver.EstimateAffectedRows(ctx, driver, ins, text, plan)
			if err != nil {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.Internal.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", text, err.Error()),
					StartPosition: position,
				})
				continue
			}
			if rowCount > int64(maxRow) {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.InsertTooManyRows.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, rowCount, maxRow),
					StartPosition: position,
				})
			}
			continue
		}

		// INSERT ... VALUES: count value rows directly.
		if len(ins.Values) > maxRow {
			advice = append(advice, &storepb.Advice{
				Status:        level,
				Code:          code.InsertTooManyRows.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, len(ins.Values), maxRow),
				StartPosition: position,
			})
		}
	}

	return explains.AppendSkippedAdvice(advice, title, code.InsertTooManyRows), nil
}

func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}
