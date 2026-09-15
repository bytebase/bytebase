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
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
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

	maxRow := int(numberPayload.Number)
	driver := checkCtx.Driver
	if driver == nil {
		return nil, nil
	}
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
		switch node.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
		default:
			continue
		}

		baseLine := stmt.BaseLine()
		text := strings.TrimRight(strings.TrimSpace(stmt.Text), ";") + ";"
		position := common.ConvertANTLRLineToPosition(baseLine + int(mysqlparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line))

		if !explains.Spend(position) {
			continue
		}

		query := mysqlparser.AffectedRowsQuery(node, stmt.Text)
		plan, err := mysqldriver.ExplainJSON(ctx, driver, query)
		if err != nil {
			advice = append(advice, &storepb.Advice{
				Status:        level,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", text, err.Error()),
				StartPosition: position,
			})
			continue
		}
		rowCount, err := mysqldriver.EstimateAffectedRows(ctx, driver, node, query, plan)
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
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" affected %d rows (estimated). The count exceeds %d.", text, rowCount, maxRow),
				StartPosition: position,
			})
		}
	}

	return explains.AppendSkippedAdvice(advice, title, code.StatementAffectedRowExceedsLimit), nil
}
