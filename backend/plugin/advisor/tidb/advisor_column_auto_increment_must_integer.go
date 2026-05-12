package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
}

// ColumnAutoIncrementMustIntegerAdvisor is the advisor checking for auto-increment column type.
type ColumnAutoIncrementMustIntegerAdvisor struct {
}

// Check checks for auto-increment column type.
func (*ColumnAutoIncrementMustIntegerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnAutoIncrementMustIntegerChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type columnAutoIncrementMustIntegerChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// columnData is the {table, column, line} carrier shared with
// advisor_column_auto_increment_must_unsigned.go and
// advisor_column_require_default.go — same package, same shape.
type columnData struct {
	table  string
	column string
	line   int
}

func (c *columnAutoIncrementMustIntegerChecker) checkStmt(ostmt OmniStmt) {
	var cols []columnData
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
			if !autoIncrementColumnIsIntegerOmni(column) {
				cols = append(cols, columnData{
					table:  tableName,
					column: column.Name,
					line:   ostmt.AbsoluteLine(column.Loc.Start),
				})
			}
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
					if column == nil {
						continue
					}
					if !autoIncrementColumnIsIntegerOmni(column) {
						cols = append(cols, columnData{table: tableName, column: column.Name, line: stmtLine})
					}
				}
			case ast.ATChangeColumn, ast.ATModifyColumn:
				if cmd.Column == nil {
					continue
				}
				if !autoIncrementColumnIsIntegerOmni(cmd.Column) {
					cols = append(cols, columnData{table: tableName, column: cmd.Column.Name, line: stmtLine})
				}
			default:
			}
		}
	default:
		return
	}

	for _, col := range cols {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.AutoIncrementColumnNotInteger.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", col.table, col.column),
			StartPosition: common.ConvertANTLRLineToPosition(col.line),
		})
	}
}

// autoIncrementColumnIsIntegerOmni returns true unless the column is
// AUTO_INCREMENT with a non-integer type. Mirrors pingcap-typed
// autoIncrementColumnIsInteger semantics: the rule fires only on
// auto-increment columns; everything else passes through.
func autoIncrementColumnIsIntegerOmni(col *ast.ColumnDef) bool {
	if col == nil {
		return true
	}
	if !col.AutoIncrement {
		return true
	}
	return omniIsIntegerType(col.TypeName)
}
