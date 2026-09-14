package pg

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// CountAffectedRows returns the planner's estimate of the rows the statement modifies. A search
// path that base.WithSearchPath put before the statement is set for the EXPLAIN in a transaction
// that is rolled back.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	setup, statement := base.SplitSearchPath(statement)
	queryRow := d.db.QueryRowContext
	if setup != "" {
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, setup); err != nil {
			return 0, err
		}
		queryRow = tx.QueryRowContext
	}
	var plan string
	if err := queryRow(ctx, "EXPLAIN (FORMAT JSON) "+statement).Scan(&plan); err != nil {
		return 0, err
	}
	return pgparser.GetEstimatedAffectedRowsFromExplainJSON(plan)
}
