package mssql

import (
	"context"
	"database/sql"
	"math"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	// MSSQL uses dry-run to get the affected rows.
	// It's dangerous if statements are executed via different connections in the pool.
	// So we have the use a dedicated connection.
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SET SHOWPLAN_ALL ON;"); err != nil {
		return 0, err
	}
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	rowsIndex, ok := util.GetColumnIndex(columns, "EstimateRows")
	if !ok {
		return 0, nil
	}

	for rows.Next() {
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			var unused any
			scanArgs[i] = &unused
		}
		var rowsColumn sql.NullFloat64
		scanArgs[rowsIndex] = &rowsColumn
		if err := rows.Scan(scanArgs...); err != nil {
			return 0, err
		}

		if rowsColumn.Valid {
			return int64(math.Round(rowsColumn.Float64)), nil
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}
