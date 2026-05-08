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
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key type allowlist.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowlist := make(map[string]bool)
	for _, tp := range stringArrayPayload.List {
		allowlist[strings.ToLower(tp)] = true
	}
	checker := &indexPrimaryKeyTypeAllowlistChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		allowlist:        allowlist,
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableNewOmniColumn),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type indexPrimaryKeyTypeAllowlistChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	allowlist        map[string]bool
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableNewOmniColumn
}

func (c *indexPrimaryKeyTypeAllowlistChecker) checkStmt(ostmt OmniStmt) {
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
			case ast.ATAddConstraint, ast.ATAddIndex:
				// Empirically tidb omni emits only ATAddConstraint for
				// `ALTER TABLE ... ADD …` forms (bare and named); ATAddIndex
				// is reserved (not emitted). Dual arm preserved for sibling
				// parity with the naming-convention advisors and mysql analog.
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
			Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", pd.column, pd.table, pd.columnType),
			StartPosition: common.ConvertANTLRLineToPosition(pd.line),
		})
	}
}

func (c *indexPrimaryKeyTypeAllowlistChecker) addNewColumn(tableName string, line int, colDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	for _, constraint := range colDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrPrimaryKey {
			tp := omniDataTypeNameCompact(colDef.TypeName)
			if _, exists := c.allowlist[tp]; !exists {
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

func (c *indexPrimaryKeyTypeAllowlistChecker) changeColumn(tableName, oldColumnName string, line int, newColumnDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	c.tablesNewColumns.delete(tableName, oldColumnName)
	for _, constraint := range newColumnDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrPrimaryKey {
			tp := omniDataTypeNameCompact(newColumnDef.TypeName)
			if _, exists := c.allowlist[tp]; !exists {
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

func (c *indexPrimaryKeyTypeAllowlistChecker) addConstraint(tableName string, line int, constraint *ast.Constraint) []pkData {
	var pkDataList []pkData
	if constraint.Type != ast.ConstrPrimaryKey {
		return pkDataList
	}
	for _, columnName := range constraint.Columns {
		columnType, err := c.getPKColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		if _, exists := c.allowlist[columnType]; !exists {
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

// getPKColumnType returns the rendered type for tableName.columnName,
// preferring the column we observed in this check pass and falling back
// to the catalog. Always returns lowercase to match advice fixture rendering
// and the case-insensitive allowlist comparison.
func (c *indexPrimaryKeyTypeAllowlistChecker) getPKColumnType(tableName, columnName string) (string, error) {
	if colDef, ok := c.tablesNewColumns.get(tableName, columnName); ok {
		return omniDataTypeNameCompact(colDef.TypeName), nil
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
	return strings.ToLower(column.GetProto().Type), nil
}
