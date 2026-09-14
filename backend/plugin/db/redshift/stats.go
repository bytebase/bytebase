package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var rowsRegexp = regexp.MustCompile("rows=([0-9]+)")

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	explainSQL := fmt.Sprintf("EXPLAIN %s", statement)
	rows, err := d.db.QueryContext(ctx, explainSQL)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var plan []string
	for rows.Next() {
		var line sql.NullString
		if err := rows.Scan(&line); err != nil {
			return 0, err
		}
		if line.Valid {
			plan = append(plan, line.String)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return getAffectedRowsFromPlan(plan)
}

// getAffectedRowsFromPlan returns the rows estimate of the top plan node. Redshift
// plans have no DML node, so an INSERT, UPDATE, or DELETE shows only the scans and
// joins that produce its rows, and the nodes below the top estimate their inputs.
func getAffectedRowsFromPlan(plan []string) (int64, error) {
	for _, line := range plan {
		matches := rowsRegexp.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		rows, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to parse rows estimate in %q", line)
		}
		return rows, nil
	}
	return 0, errors.Errorf("EXPLAIN plan has no rows estimate: %q", plan)
}
