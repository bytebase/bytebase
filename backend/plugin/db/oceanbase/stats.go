package oceanbase

import (
	"context"
	"database/sql"
	"strings"

	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

// CountAffectedRows returns the planner's estimate of the rows the INSERT, UPDATE, or DELETE
// statement modifies. OceanBase reports it on the plan's root operator, so the statement needs
// no rewrite into a SELECT of the rows it touches, as MySQL's estimate does.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	rows, err := d.db.QueryContext(ctx, "EXPLAIN FORMAT=JSON "+statement)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	plan, err := readExplainJSON(rows)
	if err != nil {
		return 0, err
	}
	return oceanbaseparser.EstimateAffectedRowsFromExplainJSON(plan)
}

// readExplainJSON concatenates the plan rows; OceanBase splits its JSON plan across rows.
func readExplainJSON(rows *sql.Rows) (string, error) {
	var plan strings.Builder
	for rows.Next() {
		var line sql.NullString
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		plan.WriteString(line.String)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return plan.String(), nil
}
