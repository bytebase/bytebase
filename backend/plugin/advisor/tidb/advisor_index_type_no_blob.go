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
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &indexTypeNoBlobChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		originalMetadata: checkCtx.OriginalMetadata,
		tablesNewColumns: make(tableNewOmniColumn),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type indexTypeNoBlobChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	originalMetadata *model.DatabaseMetadata
	tablesNewColumns tableNewOmniColumn
}

func (c *indexTypeNoBlobChecker) checkStmt(ostmt OmniStmt) {
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
	case *ast.CreateIndexStmt:
		if n.Table == nil {
			return
		}
		// Note: pingcap-typed advisor does NOT exclude FULLTEXT/SPATIAL
		// indexes from the BLOB check, even though the mysql omni rule does.
		// Preserve pingcap behavior; aligning to mysql's stricter exclusion
		// is a separate intentional behavior change (cumulative #9 territory).
		tableName := n.Table.Name
		stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
		for _, columnName := range omniIndexColumns(n.Columns) {
			columnType, err := c.getColumnType(tableName, columnName)
			if err != nil {
				continue
			}
			if isBlobType(columnType) {
				pkDataList = append(pkDataList, pkData{
					table:      tableName,
					column:     columnName,
					columnType: columnType,
					line:       stmtLine,
				})
			}
		}
	default:
		return
	}

	for _, pd := range pkDataList {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", pd.table, pd.column, pd.columnType),
			StartPosition: common.ConvertANTLRLineToPosition(pd.line),
		})
	}
}

func (c *indexTypeNoBlobChecker) addNewColumn(tableName string, line int, colDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	// Note: pingcap-typed addNewColumn checks ColumnOptionUniqKey only (NOT
	// ColumnOptionPrimaryKey), so column-level `PRIMARY KEY` on a BLOB inside
	// CREATE TABLE is NOT flagged by the pingcap version. Preserve that
	// behavior. The change-column path checks both (matching the pingcap
	// inconsistency); replicate as-is.
	tp := omniDataTypeNameCompact(colDef.TypeName)
	for _, constraint := range colDef.Constraints {
		if constraint != nil && constraint.Type == ast.ColConstrUnique {
			if isBlobType(tp) {
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

func (c *indexTypeNoBlobChecker) changeColumn(tableName, oldColumnName string, line int, newColumnDef *ast.ColumnDef) []pkData {
	var pkDataList []pkData
	c.tablesNewColumns.delete(tableName, oldColumnName)
	tp := omniDataTypeNameCompact(newColumnDef.TypeName)
	for _, constraint := range newColumnDef.Constraints {
		if constraint != nil && (constraint.Type == ast.ColConstrPrimaryKey || constraint.Type == ast.ColConstrUnique) {
			if isBlobType(tp) {
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

func (c *indexTypeNoBlobChecker) addConstraint(tableName string, line int, constraint *ast.Constraint) []pkData {
	var pkDataList []pkData
	// Mirror pingcap match: PRIMARY KEY + UNIQUE family + INDEX/KEY family.
	// Pingcap had 6 distinct ConstraintTp values (ConstraintPrimaryKey,
	// ConstraintUniqKey, ConstraintKey, ConstraintIndex, ConstraintUniqIndex,
	// ConstraintUniq); omni unifies the unique forms under ConstrUnique and
	// the index/key forms under ConstrIndex (cumulative #2). Pingcap-typed
	// advisor does NOT include ConstraintForeignKey; the mysql omni analog
	// does. Preserve pingcap behavior — FK is a separate behavior change.
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex:
	default:
		return pkDataList
	}
	for _, columnName := range constraint.Columns {
		columnType, err := c.getColumnType(tableName, columnName)
		if err != nil {
			continue
		}
		if isBlobType(columnType) {
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

// getColumnType returns the lowercase rendered type for tableName.columnName,
// preferring the column we observed in this check pass and falling back to
// the catalog.
func (c *indexTypeNoBlobChecker) getColumnType(tableName, columnName string) (string, error) {
	if colDef, ok := c.tablesNewColumns.get(tableName, columnName); ok {
		return omniDataTypeNameCompact(colDef.TypeName), nil
	}
	column := c.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column != nil {
		return strings.ToLower(column.GetProto().Type), nil
	}
	return "", errors.Errorf(errCannotFindColumnTypeFmt, tableName, columnName)
}

func isBlobType(columnType string) bool {
	switch strings.ToLower(columnType) {
	case "blob", "tinyblob", "mediumblob", "longblob":
		return true
	default:
		return false
	}
}
