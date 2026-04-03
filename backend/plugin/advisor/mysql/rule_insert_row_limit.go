package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
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
	var advice []*storepb.Advice
	explainCount := 0

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
		line := baseLine + int(mysqlparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line)

		// INSERT ... SELECT: use EXPLAIN to count rows.
		if ins.Select != nil {
			if driver == nil {
				continue
			}
			explainCount++
			res, err := advisor.Query(ctx, advisor.QueryContext{}, driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", text))
			if err != nil {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.InsertTooManyRows.Int32(),
					Title:         checkCtx.Rule.Type.String(),
					Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
					StartPosition: common.ConvertANTLRLineToPosition(line),
				})
			} else {
				rowCount, err := getInsertRows(res)
				if err != nil {
					advice = append(advice, &storepb.Advice{
						Status:        level,
						Code:          code.Internal.Int32(),
						Title:         checkCtx.Rule.Type.String(),
						Content:       fmt.Sprintf("failed to get row count for \"%s\": %s", text, err.Error()),
						StartPosition: common.ConvertANTLRLineToPosition(line),
					})
				} else if rowCount > int64(maxRow) {
					advice = append(advice, &storepb.Advice{
						Status:        level,
						Code:          code.InsertTooManyRows.Int32(),
						Title:         checkCtx.Rule.Type.String(),
						Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, rowCount, maxRow),
						StartPosition: common.ConvertANTLRLineToPosition(line),
					})
				}
			}
			if explainCount >= common.MaximumLintExplainSize {
				break
			}
			continue
		}

		// INSERT ... VALUES: count value rows directly.
		if len(ins.Values) > maxRow {
			advice = append(advice, &storepb.Advice{
				Status:        level,
				Code:          code.InsertTooManyRows.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("\"%s\" inserts %d rows. The count exceeds %d.", text, len(ins.Values), maxRow),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
	}

	return advice, nil
}

func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}

func getInsertRows(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	columns, ok := res[0].([]string)
	if !ok {
		return 0, errors.Errorf("expected []string but got %t", res[0])
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return 0, errors.Errorf("not found any data")
	}

	rowsIndex, err := getColumnIndex(columns, "rows")
	if err != nil {
		return 0, errors.Errorf("failed to find rows column")
	}

	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any but got %t", row)
		}
		switch col := row[rowsIndex].(type) {
		case int:
			return int64(col), nil
		case int32:
			return int64(col), nil
		case int64:
			return col, nil
		case string:
			v, err := strconv.ParseInt(col, 10, 64)
			if err != nil {
				return 0, errors.Errorf("expected int or int64 but got string(%s)", col)
			}
			return v, nil
		default:
			continue
		}
	}

	return 0, nil
}
