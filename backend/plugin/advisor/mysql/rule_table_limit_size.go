package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_LIMIT_SIZE, &MaximumTableSizeAdvisor{})
}

// MaximumTableSizeAdvisor checks the row count of the tables a change alters,
// truncates or drops. The judgment lives in omni
// (mysql/v1review.TableLimitSize); this adapter passes the synced snapshot
// and renders the findings as advice.
type MaximumTableSizeAdvisor struct {
}

var (
	_ advisor.Advisor = &MaximumTableSizeAdvisor{}
)

// Check checks the row count of the tables the statements change.
func (*MaximumTableSizeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	status, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mysqlv1review.TableLimitSize(nodes, checkCtx.DBSchema, int(numberPayload.Number)) {
		if f.Code != v1review.TableTooLarge {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        status,
			Code:          code.TableExceedLimitSize.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Apply DDL on large table '%s' ( %s rows ) will lock table for a long time", f.Name, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, v1review.Finding{}))),
		})
	}
	return adviceList, nil
}
