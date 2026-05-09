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
	_ advisor.Advisor = (*ColumnSetDefaultForNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, &ColumnSetDefaultForNotNullAdvisor{})
}

// ColumnSetDefaultForNotNullAdvisor checks for set default value for not null columns.
type ColumnSetDefaultForNotNullAdvisor struct {
}

// Check is Recipe A. Same wrapper-safety rationale as the batch-6
// charset/collation/comment family: EXPLAIN doesn't accept DDL in either
// pingcap or omni grammar, and CREATE/ALTER TABLE can't nest themselves.
//
// Cardinality: per-violating-column (NOT per-statement). Pingcap's
// pattern was build a list during the visit and append one advice per
// list entry after the switch — preserved here.
//
// Closes BYT-9362 (canNull cleanup) by being the last consumer of the
// pingcap-typed canNull helper.
func (*ColumnSetDefaultForNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice

	for _, ostmt := range stmts {
		var rows []columnName
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			pkCols := omniCreateTablePKColumnSet(n)
			for _, col := range n.Columns {
				if col == nil {
					continue
				}
				if pkCols[col.Name] {
					continue
				}
				if !omniColumnIsNotNull(col) || omniColumnHasDefault(col) || !omniNeedDefault(col) {
					continue
				}
				rows = append(rows, columnName{
					tableName:  tableName,
					columnName: col.Name,
					line:       ostmt.AbsoluteLine(col.Loc.Start),
				})
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			stmtLine := ostmt.FirstTokenLine()
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					for _, col := range addColumnTargets(cmd) {
						if col == nil {
							continue
						}
						if !omniColumnIsNotNull(col) || omniColumnHasDefault(col) || !omniNeedDefault(col) {
							continue
						}
						rows = append(rows, columnName{
							tableName:  tableName,
							columnName: col.Name,
							line:       stmtLine,
						})
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					col := cmd.Column
					if col == nil {
						continue
					}
					if !omniColumnIsNotNull(col) || omniColumnHasDefault(col) || !omniNeedDefault(col) {
						continue
					}
					rows = append(rows, columnName{
						tableName:  tableName,
						columnName: col.Name,
						line:       stmtLine,
					})
				default:
				}
			}
		default:
			continue
		}

		for _, r := range rows {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NotNullColumnWithNoDefault.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", r.tableName, r.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(r.line),
			})
		}
	}

	return adviceList, nil
}

// omniCreateTablePKColumnSet returns the set of column names declared as
// PRIMARY KEY via table-level constraints in the CREATE TABLE statement
// (e.g. `CREATE TABLE t (id INT, PRIMARY KEY (id))`). Inline column-level
// PRIMARY KEY (e.g. `CREATE TABLE t (id INT PRIMARY KEY)`) is detected
// by omniNeedDefault directly, not by this set.
func omniCreateTablePKColumnSet(n *ast.CreateTableStmt) map[string]bool {
	pkCols := make(map[string]bool)
	for _, c := range n.Constraints {
		if c == nil || c.Type != ast.ConstrPrimaryKey {
			continue
		}
		for _, name := range c.Columns {
			pkCols[name] = true
		}
	}
	return pkCols
}

// omniColumnIsNotNull reports whether the column's constraint list
// contains a NOT NULL marker (or a PRIMARY KEY marker, which implies NOT
// NULL in MySQL/TiDB semantics). Mirror of pingcap-typed canNull's
// inverse — pingcap returned `true` if the column COULD be null
// (no NotNull or PrimaryKey option); this returns `true` if it CANNOT.
func omniColumnIsNotNull(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	for _, c := range col.Constraints {
		if c == nil {
			continue
		}
		if c.Type == ast.ColConstrNotNull || c.Type == ast.ColConstrPrimaryKey {
			return true
		}
	}
	return false
}

// omniColumnHasDefault reports whether the column has a DEFAULT clause.
// omni stores the default expression in `ColumnDef.DefaultValue` directly
// (not in Constraints[] like pingcap's column.Options[].Tp ==
// ColumnOptionDefaultValue). Direct field check — no fallible op.
func omniColumnHasDefault(col *ast.ColumnDef) bool {
	return col != nil && col.DefaultValue != nil
}
