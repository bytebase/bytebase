package store

import (
	"context"
	"database/sql"
	"fmt"

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
	query := fmt.Sprintf("SELECT COALESCE(MAX(id), 0) FROM %s WHERE project = $1", table)
	if err := tx.QueryRowContext(ctx, query, projectID).Scan(&maxID); err != nil {
		return 0, errors.Wrapf(err, "failed to get max id for %s", table)
	}
	nextID := maxID + 1
	if nextID < idMinValue {
		nextID = idMinValue
	}
	return nextID, nil
}
