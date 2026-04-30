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
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &columnNoNullChecker{
		level:         level,
		title:         checkCtx.Rule.Type.String(),
		columnSet:     make(map[string]columnName),
		finalMetadata: checkCtx.FinalMetadata,
	}

	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			checker.checkCreateTable(n, ostmt)
		case *ast.AlterTableStmt:
			checker.checkAlterTable(n, ostmt)
		default:
		}
	}

	return checker.generateAdvice(), nil
}

type columnNoNullChecker struct {
	adviceList    []*storepb.Advice
	level         storepb.Advice_Status
	title         string
	columnSet     map[string]columnName
	finalMetadata *model.DatabaseMetadata
}

func (c *columnNoNullChecker) checkCreateTable(n *ast.CreateTableStmt, ostmt OmniStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, col := range n.Columns {
		if col == nil {
			continue
		}
		cn := columnName{
			tableName:  tableName,
			columnName: col.Name,
			line:       ostmt.AbsoluteLine(col.Loc.Start),
		}
		if _, exists := c.columnSet[cn.name()]; !exists {
			c.columnSet[cn.name()] = cn
		}
	}
}

func (c *columnNoNullChecker) checkAlterTable(n *ast.AlterTableStmt, ostmt OmniStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	// Match prior pingcap behavior: report the line of the ALTER TABLE statement
	// itself, not the line of each affected column inside the statement.
	stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		var columns []string
		switch cmd.Type {
		case ast.ATAddColumn:
			// omni populates either Columns (multi-column ADD COLUMN (...))
			// or Column (single-column ADD COLUMN). Mutually exclusive in
			// practice but not enforced by the type — read defensively.
			if len(cmd.Columns) > 0 {
				for _, col := range cmd.Columns {
					if col != nil {
						columns = append(columns, col.Name)
					}
				}
			} else if cmd.Column != nil {
				columns = append(columns, cmd.Column.Name)
			}
		case ast.ATChangeColumn:
			// Only care about the new column name.
			if cmd.Column != nil {
				columns = append(columns, cmd.Column.Name)
			}
		default:
			// MODIFY COLUMN (ATModifyColumn) is intentionally NOT handled.
			// Pre-existing inheritance from the pingcap-AST version of this
			// advisor — `ALTER TABLE t MODIFY COLUMN x INT NULL` slips past.
			// Not an omni-introduced regression; preserve current behavior.
		}
		for _, name := range columns {
			cn := columnName{
				tableName:  tableName,
				columnName: name,
				line:       stmtLine,
			}
			if _, exists := c.columnSet[cn.name()]; !exists {
				c.columnSet[cn.name()] = cn
			}
		}
	}
}

func (c *columnNoNullChecker) generateAdvice() []*storepb.Advice {
	var columnList []columnName
	for _, column := range c.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(i, j columnName) int {
		if i.line != j.line {
			if i.line < j.line {
				return -1
			}
			return 1
		}
		if i.columnName < j.columnName {
			return -1
		}
		if i.columnName > j.columnName {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		schema := c.finalMetadata.GetSchemaMetadata("")
		if schema == nil {
			continue
		}
		table := schema.GetTable(column.tableName)
		if table == nil {
			continue
		}
		col := table.GetColumn(column.columnName)
		if col != nil && col.GetProto().Nullable {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.ColumnCannotNull.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}

	return c.adviceList
}

type columnName struct {
	tableName  string
	columnName string
	line       int
}

func (c columnName) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}
