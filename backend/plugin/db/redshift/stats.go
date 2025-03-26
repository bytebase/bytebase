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

	// # EXPLAIN INSERT INTO t SELECT * FROM t;
	// QUERY PLAN
	// -------------------------------------------------------------
	// XN Seq Scan on t  (cost=0.00..0.11 rows=11 width=49)
	//
	// # EXPLAIN SELECT * FROM t;
	// QUERY PLAN
	// ------------------------------------------------------
	// XN Seq Scan on t  (cost=0.00..0.11 rows=11 width=49)
	var rowCount int64
	for rows.Next() {
		var planRow sql.NullString
		if err := rows.Scan(&planRow); err != nil {
			return 0, err
		}

		if !planRow.Valid {
			continue
		}

		matches := rowsRegexp.FindStringSubmatch(planRow.String)
		if len(matches) != 2 {
			continue
		}
		v, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return 0, errors.Errorf("failed to get integer from %q", matches[1])
		}
		rowCount = v
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return rowCount, nil
}
