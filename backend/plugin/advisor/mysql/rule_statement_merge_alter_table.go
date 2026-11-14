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
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for merging ALTER
// TABLE statements. The judgment lives in omni
// (mysql/v1review.StatementMergeAlterTable); this adapter renders its findings
// as advice.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for merging ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	nodes, stmts := omniStatements(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for _, f := range mysqlv1review.StatementMergeAlterTable(nodes) {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementRedundantAlterTable.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}

// omniStatements returns the omni nodes of the statements that have one,
// with the parsed statement each came from at the same index.
func omniStatements(parsed []base.ParsedStatement) ([]ast.Node, []base.ParsedStatement) {
	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range parsed {
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
	return nodes, stmts
}
