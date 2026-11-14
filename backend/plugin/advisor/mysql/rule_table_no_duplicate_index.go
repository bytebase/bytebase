package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableNoDuplicateIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX, &TableNoDuplicateIndexAdvisor{})
}

// TableNoDuplicateIndexAdvisor is the advisor checking for no duplicate index in table.
// The judgment lives in omni (mysql/v1review.TableNoDuplicateIndex).
type TableNoDuplicateIndexAdvisor struct {
}

// Check checks for no duplicate index in table.
func (*TableNoDuplicateIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// The rule judges the statements together, so a finding's range can lie
	// in an earlier statement than the one that reports it.
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		if node, ok := mysqlparser.GetNode(stmt.AST); ok {
			stmts = append(stmts, stmt)
			nodes = append(nodes, node)
		}
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mysqlv1review.TableNoDuplicateIndex(nodes) {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.DuplicateIndexInTable.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("`%s` has duplicate index `%s`", f.Table, f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}
