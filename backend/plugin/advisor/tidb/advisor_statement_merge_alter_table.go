package tidb

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor flags multiple ALTER TABLE statements
// on the same table that could be merged into one. The pre-omni rule
// accumulated per-table {count, lastLine} across statements, sorted
// tables by lastLine, and emitted one advice per table with count>1.
// Pure aggregator pattern (Recipe A); no sub-walks. Preserves pingcap-
// tidb's "CREATE TABLE counts as 1 modification on that table" framing
// per fixture line 16-30: CREATE on t followed by 1 ALTER on t emits
// the same "2 statements to modify table" advice as 2 ALTERs.
type StatementMergeAlterTableAdvisor struct{}

// Check flags tables modified more than once across the reviewed statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	type tableState struct {
		name     string
		count    int
		lastLine int
	}
	tableMap := make(map[string]*tableState)
	touch := func(name string, line int) {
		entry := tableMap[name]
		if entry == nil {
			entry = &tableState{name: name}
			tableMap[name] = entry
		}
		entry.count++
		entry.lastLine = line
	}

	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table != nil {
				touch(n.Table.Name, ostmt.FirstTokenLine())
			}
		case *ast.AlterTableStmt:
			if n.Table != nil {
				touch(n.Table.Name, ostmt.FirstTokenLine())
			}
		default:
		}
	}

	tableList := make([]*tableState, 0, len(tableMap))
	for _, t := range tableMap {
		tableList = append(tableList, t)
	}
	slices.SortFunc(tableList, func(i, j *tableState) int {
		switch {
		case i.lastLine < j.lastLine:
			return -1
		case i.lastLine > j.lastLine:
			return 1
		default:
			return 0
		}
	})

	var adviceList []*storepb.Advice
	for _, t := range tableList {
		if t.count > 1 {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementRedundantAlterTable.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("There are %d statements to modify table `%s`", t.count, t.name),
				StartPosition: common.ConvertANTLRLineToPosition(t.lastLine),
			})
		}
	}
	return adviceList, nil
}
