package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor flags multiple ALTER TABLE statements
// on the same table that could be merged into one.
// The judgment lives in omni (tidb/v1review.StatementMergeAlterTable); this
// adapter renders its findings as advice.
type StatementMergeAlterTableAdvisor struct{}

// Check flags tables modified more than once across the reviewed statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	nodes := make([]ast.Node, 0, len(stmts))
	for _, ostmt := range stmts {
		nodes = append(nodes, ostmt.Node)
	}
	for _, f := range tidbv1review.StatementMergeAlterTable(nodes) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementRedundantAlterTable.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
			StartPosition: base.ConvertANTLRLineToPosition(stmts[f.Statement].FirstTokenLine()),
		})
	}
	return adviceList, nil
}
