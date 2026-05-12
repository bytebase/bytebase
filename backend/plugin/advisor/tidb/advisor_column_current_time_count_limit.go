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

const (
	maxDefaultCurrentTimeColumCount   = 2
	maxOnUpdateCurrentTimeColumnCount = 1
)

var (
	_ advisor.Advisor = (*ColumnCurrentTimeCountLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
}

// ColumnCurrentTimeCountLimitAdvisor is the advisor checking for current time column count limit.
type ColumnCurrentTimeCountLimitAdvisor struct {
}

// Check checks for current time column count limit.
func (*ColumnCurrentTimeCountLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	// Cross-statement accumulator: tracks per-table counts of
	// DEFAULT-CURRENT_TIMESTAMP and ON-UPDATE-CURRENT_TIMESTAMP
	// columns across all statements in the review. Pingcap-typed
	// predecessor used the same single-pass accumulation pattern.
	tableSet := make(map[string]currentTimeTableData)

	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			line := ostmt.AbsoluteLine(n.Loc.Start)
			for _, column := range n.Columns {
				countCurrentTimeColumn(tableSet, tableName, column, line)
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			line := ostmt.AbsoluteLine(n.Loc.Start)
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					for _, column := range addColumnTargets(cmd) {
						countCurrentTimeColumn(tableSet, tableName, column, line)
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column != nil {
						countCurrentTimeColumn(tableSet, tableName, cmd.Column, line)
					}
				default:
				}
			}
		default:
		}
	}

	return generateCurrentTimeAdvice(tableSet, level, title), nil
}

// currentTimeTableData accumulates per-table counts of columns
// declaring DEFAULT CURRENT_TIMESTAMP and/or ON UPDATE CURRENT_TIMESTAMP
// (or their synonyms NOW/LOCALTIME/LOCALTIMESTAMP).
type currentTimeTableData struct {
	tableName                string
	defaultCurrentTimeCount  int
	onUpdateCurrentTimeCount int
	line                     int
}

// countCurrentTimeColumn increments the per-table counts for a single
// column. Only DATETIME/TIMESTAMP columns are counted; the omniIsTimeType
// gate matches pingcap-typed predecessor's TypeDatetime/TypeTimestamp
// switch.
func countCurrentTimeColumn(tableSet map[string]currentTimeTableData, tableName string, column *ast.ColumnDef, line int) {
	if column == nil || !omniIsTimeType(column.TypeName) {
		return
	}
	if omniIsDefaultCurrentTime(column) {
		table, exists := tableSet[tableName]
		if !exists {
			table = currentTimeTableData{tableName: tableName}
		}
		table.defaultCurrentTimeCount++
		table.line = line
		tableSet[tableName] = table
	}
	if omniIsOnUpdateCurrentTime(column) {
		table, exists := tableSet[tableName]
		if !exists {
			table = currentTimeTableData{tableName: tableName}
		}
		table.onUpdateCurrentTimeCount++
		table.line = line
		tableSet[tableName] = table
	}
}

// generateCurrentTimeAdvice emits one advice per (table, category)
// where the count exceeds the limit, sorted by line for deterministic
// output. Mirrors pingcap-typed predecessor's generateAdvice exactly.
func generateCurrentTimeAdvice(tableSet map[string]currentTimeTableData, level storepb.Advice_Status, title string) []*storepb.Advice {
	var tableList []currentTimeTableData
	for _, table := range tableSet {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(a, b currentTimeTableData) int {
		if a.line < b.line {
			return -1
		}
		if a.line > b.line {
			return 1
		}
		return 0
	})

	var adviceList []*storepb.Advice
	for _, table := range tableList {
		if table.defaultCurrentTimeCount > maxDefaultCurrentTimeColumCount {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DefaultCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table `%s` has %d DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.defaultCurrentTimeCount, maxDefaultCurrentTimeColumCount),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
		if table.onUpdateCurrentTimeCount > maxOnUpdateCurrentTimeColumnCount {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.OnUpdateCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table `%s` has %d ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.onUpdateCurrentTimeCount, maxOnUpdateCurrentTimeColumnCount),
				StartPosition: common.ConvertANTLRLineToPosition(table.line),
			})
		}
	}
	return adviceList
}
