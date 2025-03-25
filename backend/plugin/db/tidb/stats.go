package tidb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	explainSQL := fmt.Sprintf("EXPLAIN %s", statement)
	rows, err := d.db.QueryContext(ctx, explainSQL)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// mysql> explain delete from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// |  1 | DELETE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | NULL  |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	//
	// mysql> explain insert into td select * from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra           |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// |  1 | INSERT      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL | NULL |     NULL | NULL            |
	// |  1 | SIMPLE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | Using temporary |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	rowsIndex, ok := util.GetColumnIndex(columns, "rows")
	if !ok {
		return 0, nil
	}
	for rows.Next() {
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			var unused any
			scanArgs[i] = &unused
		}
		var rowsColumn sql.NullInt64
		scanArgs[rowsIndex] = &rowsColumn
		if err := rows.Scan(scanArgs...); err != nil {
			return 0, err
		}

		if rowsColumn.Valid {
			return rowsColumn.Int64, nil
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}
