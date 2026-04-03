package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for merging ALTER TABLE statements.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for merging ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &mergeAlterTableOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		tableMap: make(map[string]tableStatement),
	}

	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	rule.generateAdvice()
	return rule.GetAdviceList(), nil
}

// tableStatement represents information about a table's statements.
type tableStatement struct {
	name     string
	count    int
	lastLine int
}

type mergeAlterTableOmniRule struct {
	OmniBaseRule
	tableMap map[string]tableStatement
}

func (*mergeAlterTableOmniRule) Name() string {
	return "StatementMergeAlterTableRule"
}

func (r *mergeAlterTableOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		r.tableMap[tableName] = tableStatement{
			name:     tableName,
			count:    1,
			lastLine: r.BaseLine + int(r.ContentStartLine()),
		}
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		table, ok := r.tableMap[tableName]
		if !ok {
			table = tableStatement{
				name:  tableName,
				count: 0,
			}
		}
		table.count++
		table.lastLine = r.BaseLine + int(r.ContentStartLine())
		r.tableMap[tableName] = table
	default:
	}
}

func (r *mergeAlterTableOmniRule) generateAdvice() {
	var tableList []tableStatement
	for _, table := range r.tableMap {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(i, j tableStatement) int {
		if i.lastLine < j.lastLine {
			return -1
		}
		if i.lastLine > j.lastLine {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		if table.count > 1 {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementRedundantAlterTable.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("There are %d statements to modify table `%s`", table.count, table.name),
				StartPosition: common.ConvertANTLRLineToPosition(table.lastLine),
			})
		}
	}
}
