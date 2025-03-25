package cockroachdb

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/pkg/errors"
)

var rowsRegexp = regexp.MustCompile("rows=([0-9]+)")

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	explainSQL := fmt.Sprintf("EXPLAIN %s", statement)
	var rowCount int64
	if err := crdb.Execute(func() error {
		rows, err := d.db.QueryContext(ctx, explainSQL)
		if err != nil {
			return err
		}
		defer rows.Close()

		// test-bb=# EXPLAIN INSERT INTO t SELECT * FROM t;
		// QUERY PLAN
		// -------------------------------------------------------------
		//  Insert on t  (cost=0.00..1.03 rows=0 width=0)
		//    ->  Seq Scan on t t_1  (cost=0.00..1.03 rows=3 width=520)
		// (2 rows)
		//
		// d1=# explain select * from h1;
		// QUERY PLAN
		// ------------------------------------------------------
		//  Seq Scan on h1  (cost=0.00..35.50 rows=2550 width=4)
		// (1 row)
		for rows.Next() {
			var planRow sql.NullString
			if err := rows.Scan(&planRow); err != nil {
				return err
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
				return errors.Errorf("failed to get integer from %q", matches[1])
			}
			rowCount = v
		}
		err = rows.Err()
		return err
	}); err != nil {
		return 0, err
	}

	return rowCount, nil
}
