package pg

import (
	"context"
	"fmt"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// CountAffectedRows returns the planner's estimate of the rows the statement modifies.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	var plan string
	if err := d.db.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", statement)).Scan(&plan); err != nil {
		return 0, err
	}
	return pgparser.GetEstimatedAffectedRowsFromExplainJSON(plan)
}
