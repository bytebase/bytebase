package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

const (
	maxDefaultCurrentTimeColumCount   = 2
	maxOnUpdateCurrentTimeColumnCount = 1
)

var (
	_ advisor.Advisor = (*ColumnCurrentTimeCountLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
}

// ColumnCurrentTimeCountLimitAdvisor is the advisor checking for current time column count limit.
type ColumnCurrentTimeCountLimitAdvisor struct {
}

// Check checks for current time column count limit.
func (*ColumnCurrentTimeCountLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnCurrentTimeCountLimitOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		tableSet: make(map[string]currentTimeTableData),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	rule.generateAdvice()

	return rule.GetAdviceList(), nil
}

type currentTimeTableData struct {
	tableName                string
	defaultCurrentTimeCount  int
	onUpdateCurrentTimeCount int
	line                     int
}

type columnCurrentTimeCountLimitOmniRule struct {
	OmniBaseRule
	tableSet map[string]currentTimeTableData
}

func (*columnCurrentTimeCountLimitOmniRule) Name() string {
	return "ColumnCurrentTimeCountLimitRule"
}

func (r *columnCurrentTimeCountLimitOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnCurrentTimeCountLimitOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		if col.TypeName == nil || !omniIsTimeType(col.TypeName) {
			continue
		}
		r.checkTimeColumn(tableName, col)
	}
}

func (r *columnCurrentTimeCountLimitOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		for _, col := range omniGetColumnsFromCmd(cmd) {
			if col.TypeName == nil || !omniIsTimeType(col.TypeName) {
				continue
			}
			r.checkTimeColumn(tableName, col)
		}
	}
}

func (r *columnCurrentTimeCountLimitOmniRule) checkTimeColumn(tableName string, col *ast.ColumnDef) {
	if omniIsDefaultCurrentTime(col) {
		table, exists := r.tableSet[tableName]
		if !exists {
			table = currentTimeTableData{tableName: tableName}
		}
		table.defaultCurrentTimeCount++
		table.line = r.BaseLine + int(r.LocToLine(col.Loc))
		r.tableSet[tableName] = table
	}
	if omniIsOnUpdateCurrentTime(col) {
		table, exists := r.tableSet[tableName]
		if !exists {
			table = currentTimeTableData{tableName: tableName}
		}
		table.onUpdateCurrentTimeCount++
		table.line = r.BaseLine + int(r.LocToLine(col.Loc))
		r.tableSet[tableName] = table
	}
}

func (r *columnCurrentTimeCountLimitOmniRule) generateAdvice() {
	var tableList []currentTimeTableData
	for _, table := range r.tableSet {
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
	for _, table := range tableList {
		if table.defaultCurrentTimeCount > maxDefaultCurrentTimeColumCount {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:  r.Level,
				Code:    code.DefaultCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Table `%s` has %d DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.defaultCurrentTimeCount, maxDefaultCurrentTimeColumCount),
				StartPosition: &storepb.Position{
					Line: int32(table.line),
				},
			})
		}
		if table.onUpdateCurrentTimeCount > maxOnUpdateCurrentTimeColumnCount {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:  r.Level,
				Code:    code.OnUpdateCurrentTimeColumnCountExceedsLimit.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Table `%s` has %d ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", table.tableName, table.onUpdateCurrentTimeCount, maxOnUpdateCurrentTimeColumnCount),
				StartPosition: &storepb.Position{
					Line: int32(table.line),
				},
			})
		}
	}
}
