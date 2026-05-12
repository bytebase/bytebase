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
)

var (
	_ advisor.Advisor = (*ColumRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumRequireDefaultAdvisor{})
}

// ColumRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columRequireDefaultChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type columRequireDefaultChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// columnData is shared with advisor_column_auto_increment_must_integer.go
// (same package) — same {table, column, line} shape across the column
// attribute advisors.

func (c *columRequireDefaultChecker) checkStmt(ostmt OmniStmt) {
	var cols []columnData
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		// Table-level PK columns are exempt. Column-level PK is handled
		// inside omniNeedDefault via the column's own Constraints.
		// Names are lowercased on both sides — omni preserves user case,
		// but SQL identifier matching is case-insensitive for unquoted
		// names (and the pingcap-typed predecessor used `.L` normalization).
		pkColumns := tablePKColumnsFromConstraints(n.Constraints)
		for _, column := range n.Columns {
			if column == nil {
				continue
			}
			if pkColumns[strings.ToLower(column.Name)] {
				continue
			}
			if !omniHasDefaultValue(column) && omniNeedDefault(column) {
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
					if !omniHasDefaultValue(column) && omniNeedDefault(column) {
						cols = append(cols, columnData{table: tableName, column: column.Name, line: stmtLine})
					}
				}
			case ast.ATChangeColumn, ast.ATModifyColumn:
				if cmd.Column == nil {
					continue
				}
				column := cmd.Column
				if !omniHasDefaultValue(column) && omniNeedDefault(column) {
					cols = append(cols, columnData{table: tableName, column: column.Name, line: stmtLine})
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
			Code:          code.NoDefault.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Column `%s`.`%s` doesn't have DEFAULT.", col.table, col.column),
			StartPosition: common.ConvertANTLRLineToPosition(col.line),
		})
	}
}

// tablePKColumnsFromConstraints returns the set of column names that appear
// in any table-level PRIMARY KEY constraint, lowercased for case-insensitive
// SQL identifier matching. Mirrors pingcap-typed getPkColumnsFromConstraints
// which used `.L` (lowercase) form for the same purpose. Callers must
// lowercase the lookup key before consulting the map (omni preserves the
// user's literal case on both column.Name and constraint.Columns, so two
// references to the same column at different casings won't match without
// normalization). Column-level PK is handled separately by omniNeedDefault
// via the column's own Constraints.
func tablePKColumnsFromConstraints(constraints []*ast.Constraint) map[string]bool {
	pkColumns := make(map[string]bool)
	for _, constraint := range constraints {
		if constraint == nil || constraint.Type != ast.ConstrPrimaryKey {
			continue
		}
		for _, columnName := range constraint.Columns {
			pkColumns[strings.ToLower(columnName)] = true
		}
	}
	return pkColumns
}

// omniHasDefaultValue reports whether the column declares a DEFAULT clause.
// Mirrors pingcap-typed hasDefault. Omni surfaces DEFAULT directly on the
// column rather than via the Options/Constraints list, so this is a
// one-line nil check.
func omniHasDefaultValue(col *ast.ColumnDef) bool {
	return col != nil && col.DefaultValue != nil
}
