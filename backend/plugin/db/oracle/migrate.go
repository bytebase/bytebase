package oracle

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// ExecuteMigrationWithBeforeCommitTxFunc executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (d *Driver) ExecuteMigrationWithBeforeCommitTxFunc(ctx context.Context, m *db.MigrationInfo, statement string, beforeCommitTxFunc func(tx *sql.Tx) error) (migrationHistoryID string, updatedSchema string, resErr error) {
	if m.CreateDatabase {
		return "", "", errors.New("creating databases is not supported")
	}
	if _, err := d.executeWithBeforeCommitTxFunc(ctx, statement, m.CreateDatabase, beforeCommitTxFunc); err != nil {
		return "", "", util.FormatError(err)
	}
	return "", "", nil
}
