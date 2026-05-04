package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	primaryKeyName = "PRIMARY"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequirePKChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		tables:           make(tablePK),
		line:             make(map[string]int),
		originalMetadata: checkCtx.OriginalMetadata,
		finalMetadata:    checkCtx.FinalMetadata,
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.generateAdviceList(), nil
}

type tableRequirePKChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	tables           tablePK
	line             map[string]int
	originalMetadata *model.DatabaseMetadata
	finalMetadata    *model.DatabaseMetadata
}

func (c *tableRequirePKChecker) checkStmt(ostmt OmniStmt) {
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		c.handleCreateTable(n, ostmt)
	case *ast.DropTableStmt:
		for _, table := range n.Tables {
			if table != nil {
				delete(c.tables, table.Name)
			}
		}
	case *ast.AlterTableStmt:
		c.handleAlterTable(n, ostmt)
	default:
	}
}

func (c *tableRequirePKChecker) handleCreateTable(n *ast.CreateTableStmt, ostmt OmniStmt) {
	if n.Table == nil {
		return
	}
	if n.Like != nil {
		c.createTableLike(n)
	} else {
		c.createTable(n)
	}
	c.line[n.Table.Name] = ostmt.AbsoluteLine(n.Loc.Start)
}

func (c *tableRequirePKChecker) handleAlterTable(n *ast.AlterTableStmt, ostmt OmniStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint:
			if cmd.Constraint != nil && cmd.Constraint.Type == ast.ConstrPrimaryKey {
				c.tables[tableName] = newColumnSet(constraintPKColumns(cmd.Constraint))
			}
		case ast.ATDropConstraint:
			// omni unifies DROP PRIMARY KEY into ATDropConstraint with
			// Name == "PRIMARY"; see omni/tidb/parser/alter_table.go.
			if strings.ToUpper(cmd.Name) == primaryKeyName {
				c.initEmptyTable(tableName)
				c.line[tableName] = stmtLine
			}
		case ast.ATDropIndex:
			// Some dialects accept DROP INDEX `PRIMARY` as an alias for
			// DROP PRIMARY KEY; preserve pingcap-side behavior by treating
			// it the same way.
			if strings.ToUpper(cmd.Name) == primaryKeyName {
				c.initEmptyTable(tableName)
				c.line[tableName] = stmtLine
			}
		case ast.ATAddColumn:
			c.addPKIfExistByCols(tableName, addColumnTargets(cmd))
		case ast.ATChangeColumn:
			// cmd.Name is OLD; cmd.NewName / cmd.Column.Name is NEW.
			oldName := cmd.Name
			newName := ""
			if cmd.Column != nil {
				newName = cmd.Column.Name
			}
			if newName == "" {
				newName = cmd.NewName
			}
			if c.changeColumn(tableName, oldName, newName) {
				c.line[tableName] = stmtLine
			}
			if cmd.Column != nil {
				c.addPKIfExistByCols(tableName, []*ast.ColumnDef{cmd.Column})
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil {
				c.addPKIfExistByCols(tableName, []*ast.ColumnDef{cmd.Column})
			}
		case ast.ATDropColumn:
			if c.dropColumn(tableName, cmd.Name) {
				c.line[tableName] = stmtLine
			}
		default:
			// Skip other alter table specification types.
		}
	}
}

// addColumnTargets returns the columns produced by an ATAddColumn cmd,
// honoring omni's split between cmd.Columns (multi-column form) and
// cmd.Column (single-column form). Mutually exclusive in practice.
func addColumnTargets(cmd *ast.AlterTableCmd) []*ast.ColumnDef {
	if cmd == nil {
		return nil
	}
	if len(cmd.Columns) > 0 {
		return cmd.Columns
	}
	if cmd.Column != nil {
		return []*ast.ColumnDef{cmd.Column}
	}
	return nil
}

func (c *tableRequirePKChecker) generateAdviceList() []*storepb.Advice {
	tableList := c.tables.tableList()
	for _, tableName := range tableList {
		if len(c.tables[tableName]) == 0 {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.TableNoPK.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Table `%s` requires PRIMARY KEY", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(c.line[tableName]),
			})
		}
	}

	return c.adviceList
}

func (c *tableRequirePKChecker) initEmptyTable(name string) {
	c.tables[name] = make(columnSet)
}

func (c *tableRequirePKChecker) createTable(n *ast.CreateTableStmt) {
	table := n.Table.Name
	c.initEmptyTable(table)
	c.addPKIfExistByCols(table, n.Columns)

	for _, constraint := range n.Constraints {
		if constraint != nil && constraint.Type == ast.ConstrPrimaryKey {
			c.tables[table] = newColumnSet(constraintPKColumns(constraint))
		}
	}
}

func (c *tableRequirePKChecker) createTableLike(n *ast.CreateTableStmt) {
	table := n.Table.Name
	c.initEmptyTable(table)
	if n.Like == nil {
		return
	}
	referTableName := n.Like.Name
	if referTablePK, ok := c.tables[referTableName]; ok {
		var columns []string
		for column := range referTablePK {
			columns = append(columns, column)
		}
		c.tables[table] = newColumnSet(columns)
		return
	}
	schema := c.originalMetadata.GetSchemaMetadata("")
	if schema == nil {
		return
	}
	referTableMetadata := schema.GetTable(referTableName)
	if referTableMetadata == nil {
		return
	}
	primaryKey := referTableMetadata.GetPrimaryKey()
	if primaryKey == nil {
		return
	}
	c.tables[table] = newColumnSet(primaryKey.GetProto().GetExpressions())
}

func (c *tableRequirePKChecker) dropColumn(table string, column string) bool {
	if _, ok := c.tables[table]; !ok {
		schema := c.originalMetadata.GetSchemaMetadata("")
		if schema == nil {
			return false
		}
		t := schema.GetTable(table)
		if t == nil {
			return false
		}
		pk := t.GetIndex(primaryKeyName)
		if pk == nil {
			return false
		}
		c.tables[table] = newColumnSet(pk.GetProto().GetExpressions())
	}

	pk := c.tables[table]
	_, columnInPK := pk[column]
	delete(c.tables[table], column)
	return columnInPK
}

func (c *tableRequirePKChecker) changeColumn(table string, oldColumn string, newColumn string) bool {
	if c.dropColumn(table, oldColumn) {
		pk := c.tables[table]
		pk[newColumn] = true
		return true
	}
	return false
}

func (c *tableRequirePKChecker) addPKIfExistByCols(table string, columns []*ast.ColumnDef) {
	for _, column := range columns {
		if column == nil {
			continue
		}
		for _, constraint := range column.Constraints {
			if constraint != nil && constraint.Type == ast.ColConstrPrimaryKey {
				c.tables[table] = newColumnSet([]string{column.Name})
				return
			}
		}
	}
}

// constraintPKColumns extracts the column names from a PRIMARY KEY
// constraint. omni stores simple column lists in Constraint.Columns;
// expression-based / functional PRIMARY KEYs (rare — TiDB does support
// PRIMARY KEY ((expr)) syntax) populate Constraint.IndexColumns with
// expression nodes that we can't trivially flatten to a column-name set.
// Pingcap parity: this advisor only checks regular PRIMARY KEYs by name,
// so the simple-column path is sufficient.
func constraintPKColumns(constraint *ast.Constraint) []string {
	if constraint == nil {
		return nil
	}
	return constraint.Columns
}
