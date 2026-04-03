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
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementDisallowUsingFilesortAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_FILESORT, &StatementDisallowUsingFilesortAdvisor{})
}

// StatementDisallowUsingFilesortAdvisor is the advisor checking for using filesort.
type StatementDisallowUsingFilesortAdvisor struct {
}

func (*StatementDisallowUsingFilesortAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	driver := checkCtx.Driver
	title := checkCtx.Rule.Type.String()
	var advice []*storepb.Advice
	explainCount := 0

	if driver != nil {
		for _, stmt := range checkCtx.ParsedStatements {
			if stmt.AST == nil {
				continue
			}
			node, ok := mysqlparser.GetOmniNode(stmt.AST)
			if !ok {
				continue
			}

			// Only handle top-level SELECT statements.
			if _, ok := node.(*ast.SelectStmt); !ok {
				continue
			}

			baseLine := stmt.BaseLine()
			query := strings.TrimRight(strings.TrimSpace(stmt.Text), ";") + ";"
			line := baseLine + int(mysqlparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line)

			explainCount++
			res, err := advisor.Query(ctx, advisor.QueryContext{}, driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", query))
			if err != nil {
				advice = append(advice, &storepb.Advice{
					Status:        level,
					Code:          code.StatementExplainQueryFailed.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Failed to explain query: %s, with error: %s", query, err),
					StartPosition: common.ConvertANTLRLineToPosition(line),
				})
			} else {
				hasUsingFilesort, tables, err := hasUsingFilesortInExtraColumn(res)
				if err != nil {
					advice = append(advice, &storepb.Advice{
						Status:        level,
						Code:          code.Internal.Int32(),
						Title:         title,
						Content:       fmt.Sprintf("Failed to check extra column: %s, with error: %s", query, err),
						StartPosition: common.ConvertANTLRLineToPosition(line),
					})
				} else if hasUsingFilesort {
					advice = append(advice, &storepb.Advice{
						Status:        level,
						Code:          code.StatementHasUsingFilesort.Int32(),
						Title:         title,
						Content:       fmt.Sprintf("Using filesort detected on table(s): %s", tables),
						StartPosition: common.ConvertANTLRLineToPosition(line),
					})
				}
			}

			if explainCount >= common.MaximumLintExplainSize {
				break
			}
		}
	}

	return advice, nil
}

func hasUsingFilesortInExtraColumn(res []any) (bool, string, error) {
	if len(res) != 3 {
		return false, "", errors.Errorf("expected 3 but got %d", len(res))
	}
	columns, ok := res[0].([]string)
	if !ok {
		return false, "", errors.Errorf("expected []string but got %t", res[0])
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return false, "", errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return false, "", errors.Errorf("not found any data")
	}

	extraIndex, err := getColumnIndex(columns, "Extra")
	if err != nil {
		return false, "", errors.Errorf("failed to find rows column")
	}
	tableIndex, err := getColumnIndex(columns, "table")
	if err != nil {
		return false, "", errors.Errorf("failed to find rows column")
	}

	var tables []string
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return false, "", errors.Errorf("expected []any but got %t", row)
		}

		extra, ok := row[extraIndex].(string)
		if !ok {
			return false, "", nil
		}
		if strings.Contains(extra, "Using filesort") {
			tables = append(tables, row[tableIndex].(string))
		}
	}

	if len(tables) == 0 {
		return false, "", nil
	}

	return true, strings.Join(tables, ", "), nil
}
