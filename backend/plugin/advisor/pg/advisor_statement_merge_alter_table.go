package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for no redundant
// ALTER TABLE statements. The judgment lives in omni
// (pg/v1review.StatementMergeAlterTable); this adapter renders its findings
// as advice.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for no redundant ALTER TABLE statements.
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
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.StatementMergeAlterTable(nodes) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.StatementRedundantAlterTable.Int32(),
			Title:   title,
			Content: fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
			// V1 placed the finding on its last statement's last line
			// relative to that statement, without the statement's base line,
			// and ordered findings by that line.
			StartPosition: &storepb.Position{Line: (&BaseRule{StmtText: stmts[f.Statement].Text}).ContentEndLine()},
		})
	}
	slices.SortStableFunc(adviceList, func(a, b *storepb.Advice) int {
		return int(a.StartPosition.Line) - int(b.StartPosition.Line)
	})
	return adviceList, nil
}
