package oracle

import (
	"context"
	"database/sql"
)

// ExecuteMigrationWithBeforeCommitTxFunc executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (d *Driver) ExecuteMigrationWithBeforeCommitTxFunc(ctx context.Context, statement string, beforeCommitTxFunc func(tx *sql.Tx) error) (migrationHistoryID string, updatedSchema string, resErr error) {
	if _, err := d.executeWithBeforeCommitTxFunc(ctx, statement, beforeCommitTxFunc); err != nil {
		return "", "", err
	}
	return "", "", nil
}
