package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}
	requiredColumns := make(columnSet)
	for _, column := range stringArrayPayload.List {
		requiredColumns[column] = true
	}

	rule := &columnRequiredOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		requiredColumns: requiredColumns,
		tables:          make(tableState),
		line:            make(map[string]int),
	}

	adviceList := RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	adviceList = append(adviceList, rule.generateAdviceList()...)
	return adviceList, nil
}

type columnRequiredOmniRule struct {
	OmniBaseRule
	requiredColumns columnSet
	tables          tableState
	line            map[string]int
}

func (*columnRequiredOmniRule) Name() string {
	return "ColumnRequiredRule"
}

func (r *columnRequiredOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.DropTableStmt:
		r.checkDropTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnRequiredOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	r.line[tableName] = int(r.LocToLine(n.Loc)) + r.BaseLine
	r.initEmptyTable(tableName)

	for _, col := range n.Columns {
		r.addColumn(tableName, col.Name)
	}
}

func (r *columnRequiredOmniRule) checkDropTable(n *ast.DropTableStmt) {
	for _, ref := range n.Tables {
		tableName := omniTableName(ref)
		delete(r.tables, tableName)
	}
}

func (r *columnRequiredOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}

	for _, cmd := range n.Commands {
		lineNumber := int(r.LocToLine(cmd.Loc)) + r.BaseLine
		switch cmd.Type {
		case ast.ATAddColumn:
			if cmd.Column != nil {
				r.addColumn(tableName, cmd.Column.Name)
			}
			for _, col := range cmd.Columns {
				r.addColumn(tableName, col.Name)
			}
		case ast.ATDropColumn:
			if r.dropColumn(tableName, cmd.Name) {
				r.line[tableName] = lineNumber
			}
		case ast.ATRenameColumn:
			if r.renameColumn(tableName, cmd.Name, cmd.NewName) {
				r.line[tableName] = lineNumber
			}
		case ast.ATChangeColumn:
			if r.renameColumn(tableName, cmd.Name, cmd.NewName) {
				r.line[tableName] = lineNumber
			}
		default:
		}
	}
}

func (r *columnRequiredOmniRule) generateAdviceList() []*storepb.Advice {
	var adviceList []*storepb.Advice
	tableList := r.tables.tableList()
	for _, tableName := range tableList {
		table := r.tables[tableName]
		var missingColumns []string
		for columnName := range r.requiredColumns {
			if exists, ok := table[columnName]; !ok || !exists {
				missingColumns = append(missingColumns, columnName)
			}
		}

		if len(missingColumns) > 0 {
			slices.Sort(missingColumns)
			adviceList = append(adviceList, &storepb.Advice{
				Status:  r.Level,
				Code:    code.NoRequiredColumn.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Table `%s` requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
				StartPosition: &storepb.Position{
					Line:   int32(r.line[tableName]),
					Column: 0,
				},
			})
		}
	}
	return adviceList
}

func (r *columnRequiredOmniRule) initEmptyTable(tableName string) columnSet {
	r.tables[tableName] = make(columnSet)
	return r.tables[tableName]
}

func (r *columnRequiredOmniRule) addColumn(tableName string, columnName string) {
	if _, ok := r.requiredColumns[columnName]; !ok {
		return
	}

	if table, ok := r.tables[tableName]; !ok {
		r.initFullTable(tableName)
	} else {
		table[columnName] = true
	}
}

func (r *columnRequiredOmniRule) dropColumn(tableName string, columnName string) bool {
	if _, ok := r.requiredColumns[columnName]; !ok {
		return false
	}
	table, ok := r.tables[tableName]
	if !ok {
		table = r.initFullTable(tableName)
	}
	table[columnName] = false
	return true
}

func (r *columnRequiredOmniRule) renameColumn(tableName string, oldColumn string, newColumn string) bool {
	_, oldNeed := r.requiredColumns[oldColumn]
	_, newNeed := r.requiredColumns[newColumn]
	if !oldNeed && !newNeed {
		return false
	}
	table, ok := r.tables[tableName]
	if !ok {
		table = r.initFullTable(tableName)
	}
	if oldNeed {
		table[oldColumn] = false
	}
	if newNeed {
		table[newColumn] = true
	}
	return oldNeed
}

func (r *columnRequiredOmniRule) initFullTable(tableName string) columnSet {
	table := r.initEmptyTable(tableName)
	for column := range r.requiredColumns {
		table[column] = true
	}
	return table
}
