package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
	"go.uber.org/zap"
)

const (
	primaryKeyName = "PRIMARY"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLTableRequirePK, &TableRequirePKAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (adv *TableRequirePKAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequirePKChecker{
		level:   level,
		tables:  make(tablePK),
		catalog: ctx.Catalog,
		logger:  ctx.Logger,
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.generateAdviceList(), nil
}

type tableRequirePKChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	tables     tablePK
	catalog    catalog.Service
	logger     *zap.Logger
}

// Enter implements the ast.Visitor interface
func (v *tableRequirePKChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		v.createTable(node)
	// DROP TABLE
	case *ast.DropTableStmt:
		for _, table := range node.Tables {
			delete(v.tables, table.Name.String())
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		tableName := node.Table.Name.O
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD CONSTRAINT
			case ast.AlterTableAddConstraint:
				if spec.Constraint.Tp == ast.ConstraintPrimaryKey {
					v.tables[tableName] = newColumnSet(convertConstraintToKeySlice(spec.Constraint))
				}
			// DROP PRIMARY KEY
			case ast.AlterTableDropPrimaryKey:
				v.initEmptyTable(tableName)
			// DROP INDEX
			case ast.AlterTableDropIndex:
				if strings.ToUpper(spec.Name) == primaryKeyName {
					v.initEmptyTable(tableName)
				}
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				v.addPKIfExistByCols(tableName, spec.NewColumns)
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				v.changeColumn(tableName, spec.OldColumnName.Name.String(), spec.NewColumns[0].Name.Name.String())
				v.addPKIfExistByCols(tableName, spec.NewColumns[:1])
			// MODIFY COLUMN
			case ast.AlterTableModifyColumn:
				v.addPKIfExistByCols(tableName, spec.NewColumns[:1])
			// DROP COLUMN
			case ast.AlterTableDropColumn:
				v.dropColumn(tableName, spec.OldColumnName.Name.String())
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface
func (v *tableRequirePKChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (v *tableRequirePKChecker) generateAdviceList() []advisor.Advice {
	tableList := v.tables.tableList()
	for _, tableName := range tableList {
		if len(v.tables[tableName]) == 0 {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    common.TableNoPK,
				Title:   "Require PRIMARY KEY",
				Content: fmt.Sprintf("Table `%s` requires PRIMARY KEY", tableName),
			})
		}
	}

	if len(v.adviceList) == 0 {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return v.adviceList
}

func (v *tableRequirePKChecker) initEmptyTable(name string) columnSet {
	v.tables[name] = make(columnSet)
	return v.tables[name]
}

func (v *tableRequirePKChecker) createTable(node *ast.CreateTableStmt) {
	table := node.Table.Name.String()
	v.initEmptyTable(table)
	v.addPKIfExistByCols(table, node.Cols)

	for _, constraint := range node.Constraints {
		if constraint.Tp == ast.ConstraintPrimaryKey {
			v.tables[table] = newColumnSet(convertConstraintToKeySlice(constraint))
		}
	}
}

func (v *tableRequirePKChecker) dropColumn(table string, column string) {
	if _, ok := v.tables[table]; !ok {
		ctx := context.Background()
		pk, err := v.catalog.FindIndex(ctx, &catalog.IndexFind{
			TableName: table,
			IndexName: primaryKeyName,
		})
		if err != nil {
			v.logger.Error(
				"Cannot find primary key in table",
				zap.String("table_name", table),
				zap.Error(err),
			)
			return
		}
		if pk == nil {
			return
		}
		v.tables[table] = newColumnSet(pk.ColumnExpressions)
	}

	delete(v.tables[table], column)
}

func (v *tableRequirePKChecker) changeColumn(table string, oldColumn string, newColumn string) {
	v.dropColumn(table, oldColumn)
	pk := v.tables[table]
	pk[newColumn] = true
}

func (v *tableRequirePKChecker) addPKIfExistByCols(table string, columns []*ast.ColumnDef) {
	for _, column := range columns {
		for _, option := range column.Options {
			if option.Tp == ast.ColumnOptionPrimaryKey {
				v.tables[table] = newColumnSet([]string{column.Name.Name.String()})
				return
			}
		}
	}
}

func convertConstraintToKeySlice(constraint *ast.Constraint) []string {
	var columns []string
	for _, key := range constraint.Keys {
		columns = append(columns, key.Column.Name.String())
	}
	return columns
}
