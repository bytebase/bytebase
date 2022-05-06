package mysql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLColumnRequirement, &ColumnRequirementAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLColumnRequirement, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (adv *ColumnRequirementAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := api.UnmarshalRequiredColumnRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	requiredColumns := make(columnSet)
	for _, column := range payload.ColumnList {
		requiredColumns[column] = true
	}
	checker := &columnRequirementChecker{
		level:              level,
		requiredColumnList: strings.Join(payload.ColumnList, ", "),
		requiredColumns:    requiredColumns,
		tables:             make(tableState),
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.generateAdvisorList(), nil
}

type columnSet map[string]bool
type tableState map[string]columnSet

type columnRequirementChecker struct {
	advisorList        []advisor.Advice
	level              advisor.Status
	requiredColumns    columnSet
	requiredColumnList string
	tables             tableState
}

func (v *columnRequirementChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		v.createTable(node)
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			table := node.Table.Name.O
			switch spec.Tp {
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				v.renameColumn(table, spec.OldColumnName.Name.O, spec.NewColumnName.Name.O)
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					v.addColumn(table, column.Name.Name.O)
				}
			// DROP COLUMN
			case ast.AlterTableDropColumn:
				v.dropColumn(table, spec.OldColumnName.Name.O)
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				v.renameColumn(table, spec.OldColumnName.Name.O, spec.NewColumns[0].Name.Name.O)
			}
		}
	}
	return in, false
}

func (v *columnRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (v *columnRequirementChecker) generateAdvisorList() []advisor.Advice {
	for tableName, table := range v.tables {
		for column := range v.requiredColumns {
			if exist, ok := table[column]; !ok || !exist {
				v.advisorList = append(v.advisorList, advisor.Advice{
					Status:  v.level,
					Code:    common.NoRequiredColumn,
					Title:   "Require columns",
					Content: fmt.Sprintf("%q requires columns: %s", tableName, v.requiredColumnList),
				})
				break
			}
		}
	}
	if len(v.advisorList) == 0 {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return v.advisorList
}

// initEmptyTable will initialize a table without any required columns.
func (v *columnRequirementChecker) initEmptyTable(name string) columnSet {
	v.tables[name] = make(columnSet)
	return v.tables[name]
}

// initFullTable will initialize a table with all required columns.
func (v *columnRequirementChecker) initFullTable(name string) columnSet {
	table := v.initEmptyTable(name)
	for column := range v.requiredColumns {
		table[column] = true
	}
	return table
}

func (v *columnRequirementChecker) renameColumn(table string, oldColumn string, newColumn string) {
	_, oldNeed := v.requiredColumns[oldColumn]
	_, newNeed := v.requiredColumns[newColumn]
	if !oldNeed && !newNeed {
		return
	}
	t, ok := v.tables[table]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		t = v.initFullTable(table)
	}
	if oldNeed {
		t[oldColumn] = false
	}
	if newNeed {
		t[newColumn] = true
	}
}

func (v *columnRequirementChecker) dropColumn(table string, column string) {
	if _, ok := v.requiredColumns[column]; !ok {
		return
	}
	t, ok := v.tables[table]
	if !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		t = v.initFullTable(table)
	}
	t[column] = false
}

func (v *columnRequirementChecker) addColumn(table string, column string) {
	if _, ok := v.requiredColumns[column]; !ok {
		return
	}
	if t, ok := v.tables[table]; !ok {
		// We do not retrospectively check.
		// So we assume it contains all required columns.
		v.initFullTable(table)
	} else {
		t[column] = true
	}
}

func (v *columnRequirementChecker) createTable(node *ast.CreateTableStmt) {
	v.initEmptyTable(node.Table.Name.O)
	for _, column := range node.Cols {
		v.addColumn(node.Table.Name.O, column.Name.Name.O)
	}
}
