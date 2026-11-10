package tidb

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	omniast "github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	rows, err := d.db.QueryContext(ctx, fmt.Sprintf("EXPLAIN %s", statement))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	idIndex, ok := util.GetColumnIndex(columns, "id")
	if !ok {
		return 0, errors.Errorf("EXPLAIN returned no id column: %v", columns)
	}
	estRowsIndex, ok := util.GetColumnIndex(columns, "estRows")
	if !ok {
		return 0, errors.Errorf("EXPLAIN returned no estRows column: %v", columns)
	}
	var plan []planRow
	for rows.Next() {
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			var unused any
			scanArgs[i] = &unused
		}
		var row planRow
		scanArgs[idIndex] = &row.id
		scanArgs[estRowsIndex] = &row.estRows
		if err := rows.Scan(scanArgs...); err != nil {
			return 0, err
		}
		plan = append(plan, row)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return getAffectedRows(plan, statement)
}

// getAffectedRows returns the rows statement modifies according to its plan. The plan estimates the
// rows the statement reads, and a multi-table UPDATE or DELETE can change a row of each target table
// for every one of them.
func getAffectedRows(plan []planRow, statement string) (int64, error) {
	estimate, err := getAffectedRowsFromPlan(plan)
	if err != nil {
		return 0, err
	}
	return common.RoundRows(estimate * float64(countDMLTargets(statement))), nil
}

// countDMLTargets returns how many tables an UPDATE or DELETE can change for each row it reads: the
// targets a multi-table DELETE lists, or the tables its assignments qualify, counting an unqualified
// assignment as another joined table, up to the number of joined tables. Other statements change one.
func countDMLTargets(statement string) int {
	list, err := tidbparser.ParseTiDBOmni(statement)
	if err != nil || list == nil || len(list.Items) != 1 {
		return 1
	}
	switch n := list.Items[0].(type) {
	case *omniast.DeleteStmt:
		// Tables holds the delete targets in every form: [t] for the single-table
		// DELETE FROM t, and the listed targets for a multi-table delete.
		return max(len(n.Tables), 1)
	case *omniast.UpdateStmt:
		var tables []string
		unqualified := 0
		for _, assignment := range n.SetList {
			switch {
			case assignment.Column == nil || assignment.Column.Table == "":
				unqualified++
			default:
				table := strings.ToLower(assignment.Column.Table)
				if !slices.Contains(tables, table) {
					tables = append(tables, table)
				}
			}
		}
		return max(min(len(tables)+unqualified, countTableSources(n.Tables)), 1)
	default:
	}
	return 1
}

func countTableSources(exprs []omniast.TableExpr) int {
	total := 0
	for _, expr := range exprs {
		total += countTableExpr(expr)
	}
	return total
}

func countTableExpr(expr omniast.TableExpr) int {
	switch n := expr.(type) {
	case *omniast.JoinClause:
		count := countTableExpr(n.Left)
		if n.Right != nil {
			count += countTableExpr(n.Right)
		}
		return count
	case *omniast.TableRef, *omniast.SubqueryExpr:
		return 1
	default:
		return 0
	}
}

type planRow struct {
	id      string
	estRows string
}

// getAffectedRowsFromPlan returns the estRows of the operator that feeds rows to the
// INSERT, UPDATE, or DELETE at the plan root. The root itself reports N/A, operators
// deeper in the tree estimate rows before the filters, joins, and limits above them
// apply, and a later root plans a CTE or subquery.
func getAffectedRowsFromPlan(plan []planRow) (float64, error) {
	if len(plan) == 0 {
		return 0, errors.New("EXPLAIN returned no plan")
	}
	if operator, _, _ := strings.Cut(plan[0].id, "_"); operator != "Insert" && operator != "Update" && operator != "Delete" {
		return 0, errors.Errorf("EXPLAIN plan root %q is not an INSERT, UPDATE, or DELETE", plan[0].id)
	}
	// EXPLAIN lists operators in pre-order and draws each child of the root with a
	// leading "├─" or "└─", so the next row is the root's first child unless the root
	// has none, as for INSERT ... SET.
	if len(plan) == 1 || (!strings.HasPrefix(plan[1].id, "├─") && !strings.HasPrefix(plan[1].id, "└─")) {
		return 0, nil
	}
	estRows, err := strconv.ParseFloat(plan[1].estRows, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse estRows of %q", plan[1].id)
	}
	return estRows, nil
}
