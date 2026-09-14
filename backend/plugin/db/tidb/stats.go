package tidb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
	return getAffectedRowsFromPlan(plan)
}

type planRow struct {
	id      string
	estRows string
}

// getAffectedRowsFromPlan returns the estRows of the operator that feeds rows to the
// INSERT, UPDATE, or DELETE at the plan root. The root itself reports N/A, operators
// deeper in the tree estimate rows before the filters, joins, and limits above them
// apply, and a later root plans a CTE or subquery.
func getAffectedRowsFromPlan(plan []planRow) (int64, error) {
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
	return common.RoundRows(estRows), nil
}
