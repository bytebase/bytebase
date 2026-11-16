package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for merging ALTER TABLE statements.
// The judgment lives in omni (mariadb/v1review.StatementMergeAlterTable); this adapter
// renders its findings as advice.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for merging ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	var adviceList []*storepb.Advice
	for _, f := range mariadbv1review.StatementMergeAlterTable(nodes) {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementRedundantAlterTable.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}
