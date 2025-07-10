package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	if d.dbType == storepb.Engine_OCEANBASE {
		return countAffectedRowsForOceanBase(ctx, d.db, statement)
	}

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

func countAffectedRowsForOceanBase(ctx context.Context, sqlDB *sql.DB, dml string) (int64, error) {
	explainSQL := fmt.Sprintf("EXPLAIN FORMAT=JSON %s", dml)
	rows, err := sqlDB.QueryContext(ctx, explainSQL)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var planColumn sql.NullString
		if err := rows.Scan(&planColumn); err != nil {
			return 0, err
		}
		if !planColumn.Valid {
			continue
		}
		var planValue map[string]json.RawMessage
		if err := json.Unmarshal([]byte(planColumn.String), &planValue); err != nil {
			return 0, errors.Wrapf(err, "failed to parse query plan from string: %+v", planColumn.String)
		}
		if len(planValue) == 0 {
			continue
		}
		queryPlan := oceanBaseQueryPlan{}
		if err := queryPlan.Unmarshal(planValue); err != nil {
			return 0, errors.Wrapf(err, "failed to parse query plan from map: %+v", planValue)
		}
		if queryPlan.Operator != "" {
			return queryPlan.EstRows, nil
		}
		count := int64(-1)
		for k, v := range planValue {
			if !strings.HasPrefix(k, "CHILD_") {
				continue
			}
			child := oceanBaseQueryPlan{}
			if err := child.Unmarshal(v); err != nil {
				return 0, errors.Wrapf(err, "failed to parse field '%s', value: %+v", k, v)
			}
			if child.Operator != "" && child.EstRows > count {
				count = child.EstRows
			}
		}
		if count >= 0 {
			return count, nil
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}

// oceanBaseQueryPlan represents the query plan of OceanBase.
type oceanBaseQueryPlan struct {
	ID       int    `json:"ID"`
	Operator string `json:"OPERATOR"`
	Name     string `json:"NAME"`
	EstRows  int64  `json:"EST.ROWS"`
	Cost     int    `json:"COST"`
	OutPut   any    `json:"output"`
}

// Unmarshal parses data and stores the result to current oceanBaseQueryPlan.
func (plan *oceanBaseQueryPlan) Unmarshal(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if b != nil {
		return json.Unmarshal(b, &plan)
	}
	return nil
}
