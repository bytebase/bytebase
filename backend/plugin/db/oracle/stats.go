package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	// Statement includes trailing semicolon, so we need to remove it.
	if _, err := d.db.ExecContext(ctx, fmt.Sprintf("EXPLAIN PLAN FOR %s", statement)); err != nil {
		return 0, err
	}
	rows, err := d.db.QueryContext(ctx, "SELECT * FROM TABLE(DBMS_XPLAN.DISPLAY)")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// EXPLAIN PLAN FOR UPDATE TEST SET id = 121 WHERE id = 242; SELECT * FROM TABLE(DBMS_XPLAN.DISPLAY);
	// | Id | Operation | Name | Rows | Bytes | Cost (%CPU)| Time |
	// ---------------------------------------------------------------------------
	// | 0 | UPDATE STATEMENT | | 1 | 13 | 3 (0)| 00:00:01 |
	// | 1 | UPDATE | TEST | | | | |
	// |* 2 | TABLE ACCESS FULL| TEST | 1 | 13 | 3 (0)| 00:00:01 |
	var rowsIndex int
	foundRowsIndex := false
	for rows.Next() {
		var planRow sql.NullString
		if err := rows.Scan(&planRow); err != nil {
			return 0, err
		}
		if !planRow.Valid {
			continue
		}
		tokens := strings.Split(planRow.String, "|")
		if !foundRowsIndex {
			for i, token := range tokens {
				token = strings.Trim(token, " ")
				if token == "Rows" {
					rowsIndex, foundRowsIndex = i, true
					break
				}
			}
			continue
		}
		if rowsIndex >= len(tokens) {
			continue
		}
		rowsToken := strings.Trim(tokens[rowsIndex], " ")
		if rowsToken == "" {
			continue
		}
		v, err := strconv.ParseInt(rowsToken, 10, 64)
		if err != nil {
			return 0, errors.Errorf("failed to get integer from %q", rowsToken)
		}
		return v, nil
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}
