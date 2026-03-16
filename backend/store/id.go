package store

import (
	"context"
	"database/sql"

	"github.com/bytebase/bytebase/backend/common/qb"
	"github.com/pkg/errors"
)

// idMinValue is the minimum ID value for per-project auto-increment.
// Matches the old ALTER SEQUENCE ... RESTART WITH 101 convention,
// reserving IDs below 101 for seed/test data.
const idMinValue int64 = 101

// nextProjectID returns the next per-project auto-increment ID for the given table.
// Must be called within a transaction. Locks the project row to serialize concurrent inserts.
// Returns at least idMinValue (101) for new projects.
func nextProjectID(ctx context.Context, tx *sql.Tx, table, projectID string) (int64, error) {
	if _, err := tx.ExecContext(ctx,
		"SELECT 1 FROM project WHERE resource_id = $1 FOR UPDATE", projectID); err != nil {
		return 0, errors.Wrapf(err, "failed to lock project %s", projectID)
	}
	var maxID int64

	q := qb.Q()
	q.Space(`
		SELECT COALESCE(MAX(id), 0) FROM ? WHERE project = ?
	`, qb.Q().Space(table), projectID)
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&maxID); err != nil {
		return 0, errors.Wrapf(err, "failed to get max id for %s", table)
	}
	nextID := maxID + 1
	if nextID < idMinValue {
		nextID = idMinValue
	}
	return nextID, nil
}
