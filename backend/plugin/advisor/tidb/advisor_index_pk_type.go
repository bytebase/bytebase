package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexPkTypeAdvisor)(nil)
)

const errCannotFindColumnTypeFmt = "cannot find the type of `%s`.`%s`"

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT, &IndexPkTypeAdvisor{})
}

// IndexPkTypeAdvisor is the advisor checking for correct type of PK.
type IndexPkTypeAdvisor struct {
}

// Check checks for correct type of PK.
func (*IndexPkTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &indexPkTypeChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		line:             make(map[string]int),
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableNewOmniColumn),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

// tableNewOmniColumn tracks per-review column definitions by table+name
// for the index-family advisors that need to resolve column types referenced
// later in the same review (e.g. an ALTER TABLE ADD INDEX over a column
// just added). Shared by advisor_index_pk_type, advisor_index_type_no_blob,
// and advisor_index_primary_key_type_allowlist.
type omniColumnNameToColumnDef map[string]*ast.ColumnDef
type tableNewOmniColumn map[string]omniColumnNameToColumnDef

func (t tableNewOmniColumn) set(tableName, columnName string, colDef *ast.ColumnDef) {
	if _, ok := t[tableName]; !ok {
		t[tableName] = make(omniColumnNameToColumnDef)
	}
	t[tableName][columnName] = colDef
}

func (t tableNewOmniColumn) get(tableName, columnName string) (colDef *ast.ColumnDef, ok bool) {
	if _, ok := t[tableName]; !ok {
		return nil, false
	}
	col, ok := t[tableName][columnName]
	return col, ok
}

func (t tableNewOmniColumn) delete(tableName, columnName string) {
	if _, ok := t[tableName]; !ok {
		return
	}
	delete(t[tableName], columnName)
}

type indexPkTypeChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	line             map[string]int
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableNewOmniColumn
}

type pkData struct {
	table      string
	column     string
	columnType string
	line       int
}

func (c *indexPkTypeChecker) checkStmt(ostmt OmniStmt) {
	var pkDataList []pkData
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		for _, column := range n.Columns {
			if column == nil {
				continue
			}
			pds := c.addNewColumn(tableName, ostmt.AbsoluteLine(column.Loc.Start), column)
			pkDataList = append(pkDataList, pds...)
		}
		for _, constraint := range n.Constraints {
			if constraint == nil {
				continue
			}
			pds := c.addConstraint(tableName, ostmt.AbsoluteLine(constraint.Loc.Start), constraint)
			pkDataList = append(pkDataList, pds...)
		}
	case *ast.AlterTableStmt:
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
			case ast.ATAddColumn:
				for _, column := range addColumnTargets(cmd) {
					pds := c.addNewColumn(tableName, stmtLine, column)
					pkDataList = append(pkDataList, pds...)
				}
			case ast.ATAddConstraint:
				if cmd.Constraint != nil {
					pds := c.addConstraint(tableName, stmtLine, cmd.Constraint)
					pkDataList = append(pkDataList, pds...)
				}
			case ast.ATChangeColumn, ast.ATModifyColumn:
				if cmd.Column == nil {
					continue
				}
				newColumnDef := cmd.Column
				oldColumnName := newColumnDef.Name
				if cmd.Type == ast.ATChangeColumn && cmd.Name != "" {
					// CHANGE COLUMN: cmd.Name is the OLD column name.
					oldColumnName = cmd.Name
				}
				pds := c.changeColumn(tableName, oldColumnName, stmtLine, newColumnDef)
				pkDataList = append(pkDataList, pds...)
			default:
			}
		}
	default:
		return
	}

	for _, pd := range pkDataList {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.IndexPKType.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", pd.table, pd.column, pd.columnType),
			StartPosition: common.ConvertANTLRLineToPosition(pd.line),
		})
	}
}

func (c *indexPkTypeChecker) addNewColumn(tableName string, line int, colDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	for _, constraint := range colDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrPrimaryKey {
			tp := formatColumnType(colDef.TypeName)
			if tp != "INT" && tp != "BIGINT" {
				pkDataList = append(pkDataList, pkData{
					table:      tableName,
					column:     colDef.Name,
					columnType: tp,
					line:       line,
				})
			}
		}
	}
	c.tablesNewColumns.set(tableName, colDef.Name, colDef)
	return pkDataList
}

func (c *indexPkTypeChecker) changeColumn(tableName, oldColumnName string, line int, newColumnDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	c.tablesNewColumns.delete(tableName, oldColumnName)
	for _, constraint := range newColumnDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrPrimaryKey {
			tp := formatColumnType(newColumnDef.TypeName)
			if tp != "INT" && tp != "BIGINT" {
				pkDataList = append(pkDataList, pkData{
					table:      tableName,
					column:     newColumnDef.Name,
					columnType: tp,
					line:       line,
				})
			}
		}
	}
	c.tablesNewColumns.set(tableName, newColumnDef.Name, newColumnDef)
	return pkDataList
}

func (c *indexPkTypeChecker) addConstraint(tableName string, line int, constraint *ast.Constraint) []pkData {
	var pkDataList []pkData
	if constraint.Type != ast.ConstrPrimaryKey {
		return pkDataList
	}
	for _, columnName := range constraint.Columns {
		columnType, err := c.getPKColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		if columnType != "INT" && columnType != "BIGINT" {
			pkDataList = append(pkDataList, pkData{
				table:      tableName,
				column:     columnName,
				columnType: columnType,
				line:       line,
			})
		}
	}
	return pkDataList
}

// getPKColumnType returns the rendered type string for tableName.columnName,
// preferring the column we observed in this check pass and falling back to
// the catalog. Returns an empty string and a non-nil error if the column
// cannot be located in either.
func (c *indexPkTypeChecker) getPKColumnType(tableName, columnName string) (string, error) {
	if colDef, ok := c.tablesNewColumns.get(tableName, columnName); ok {
		return formatColumnType(colDef.TypeName), nil
	}
	schema := c.originalMetadata.GetSchemaMetadata("")
	if schema == nil {
		return "", errors.Errorf(errCannotFindColumnTypeFmt, tableName, columnName)
	}
	table := schema.GetTable(tableName)
	if table == nil {
		return "", errors.Errorf(errCannotFindColumnTypeFmt, tableName, columnName)
	}
	column := table.GetColumn(columnName)
	if column == nil {
		return "", errors.Errorf(errCannotFindColumnTypeFmt, tableName, columnName)
	}
	return column.GetProto().Type, nil
}

// formatColumnType renders a column type for advice content:
// "INT"/"BIGINT" uppercase (matched against the rule), everything else
// lowercase Name with optional "(Length[,Scale])". Types with non-trivial
// rendering (ENUM/SET value lists, BIT, DECIMAL implicit-Scale) are not
// special-cased; extend the switch if a fixture exposes a divergence.
func formatColumnType(t *ast.DataType) string {
	if t == nil {
		return ""
	}
	switch strings.ToUpper(t.Name) {
	case "INT":
		return "INT"
	case "BIGINT":
		return "BIGINT"
	}
	lower := strings.ToLower(t.Name)
	switch {
	case t.Length > 0 && t.Scale > 0:
		return fmt.Sprintf("%s(%d,%d)", lower, t.Length, t.Scale)
	case t.Length > 0:
		return fmt.Sprintf("%s(%d)", lower, t.Length)
	default:
		return lower
	}
}
