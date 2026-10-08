package redshift

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var rowsRegexp = regexp.MustCompile("rows=([0-9]+)")

// CountAffectedRows returns the planner's estimate of the rows the statement modifies. A search
// path that base.WithSearchPath put before the statement is set for the EXPLAIN in a transaction
// that is rolled back.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	setup, statement := base.SplitSearchPath(statement)
	query := d.db.QueryContext
	if setup != "" {
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, setup); err != nil {
			return 0, err
		}
		query = tx.QueryContext
	}
	rows, err := query(ctx, "EXPLAIN "+statement)
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
