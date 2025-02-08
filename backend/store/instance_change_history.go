package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store/model"
)

// InstanceChangeHistoryMessage records the change history of an instance.
// it deprecates the old MigrationHistory.
type InstanceChangeHistoryMessage struct {
	CreatedTs           int64
	UpdatedTs           int64
	Status              db.MigrationStatus
	Version             model.Version
	Statement           string
	ExecutionDurationNs int64

	// Output only
	UID string
}

// FindInstanceChangeHistoryMessage is for listing a list of instance change history.
type FindInstanceChangeHistoryMessage struct {
	Version *model.Version
}

// UpdateInstanceChangeHistoryMessage is for updating an instance change history.
type UpdateInstanceChangeHistoryMessage struct {
	ID string

	Status              *db.MigrationStatus
	ExecutionDurationNs *int64
}

// CreateInstanceChangeHistoryForMigrator creates an instance change history for migrator.
func (s *Store) CreateInstanceChangeHistoryForMigrator(ctx context.Context, create *InstanceChangeHistoryMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := s.createInstanceChangeHistoryImplForMigrator(ctx, tx, create); err != nil {
		return err
	}
	return tx.Commit()
}

// CreatePendingInstanceChangeHistory creates an instance change history.
func (s *Store) CreatePendingInstanceChangeHistoryForMigrator(ctx context.Context, prevSchema string, m *db.MigrationInfo, statement string, sheetID *int, version model.Version) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	instanceChange := &InstanceChangeHistoryMessage{
		Status:              db.Pending,
		Version:             version,
		Statement:           statement,
		ExecutionDurationNs: 0,
	}
	var uid string
	id, err := s.createInstanceChangeHistoryImplForMigrator(ctx, tx, instanceChange)
	if err != nil {
		return "", err
	}
	uid = id

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return uid, nil
}

func (*Store) createInstanceChangeHistoryImplForMigrator(ctx context.Context, tx *Tx, create *InstanceChangeHistoryMessage) (string, error) {
	query := `
		INSERT INTO instance_change_history (
			status,
			version,
			statement,
			execution_duration_ns
		) VALUES ($1, $2, $3, $4)
		RETURNING id`

	storedVersion, err := create.Version.Marshal()
	if err != nil {
		return "", err
	}

	var uid string
	if err := tx.QueryRowContext(ctx, query,
		create.Status,
		storedVersion,
		create.Statement,
		create.ExecutionDurationNs,
	).Scan(&uid); err != nil {
		return "", err
	}

	return uid, nil
}

// UpdateInstanceChangeHistory updates an instance change history.
// it deprecates the old UpdateHistoryAsDone and UpdateHistoryAsFailed.
func (s *Store) UpdateInstanceChangeHistory(ctx context.Context, update *UpdateInstanceChangeHistoryMessage) error {
	set, args := []string{}, []any{}
	if v := update.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.ExecutionDurationNs; v != nil {
		set, args = append(set, fmt.Sprintf("execution_duration_ns = $%d", len(args)+1)), append(args, *v)
	}
	if len(set) == 0 {
		return nil
	}
	query := `
		UPDATE instance_change_history
		SET ` + strings.Join(set, ", ") + `
		WHERE ` + fmt.Sprintf("id = $%d", len(args)+1)
	args = append(args, update.ID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

// ListInstanceChangeHistoryForMigrator finds the instance change history for the migrator.
func (s *Store) ListInstanceChangeHistoryForMigrator(ctx context.Context, find *FindInstanceChangeHistoryMessage) ([]*InstanceChangeHistoryMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Version; v != nil {
		storedVersion, err := find.Version.Marshal()
		if err != nil {
			return nil, err
		}
		where, args = append(where, fmt.Sprintf("version = $%d", len(args)+1)), append(args, storedVersion)
	}

	query := `
		SELECT
			id,
			created_ts,
			updated_ts,
			status,
			version,
			statement,
			execution_duration_ns
		FROM instance_change_history
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY id DESC`

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*InstanceChangeHistoryMessage
	for rows.Next() {
		var changeHistory InstanceChangeHistoryMessage
		var storedVersion string
		if err := rows.Scan(
			&changeHistory.UID,
			&changeHistory.CreatedTs,
			&changeHistory.UpdatedTs,
			&changeHistory.Status,
			&storedVersion,
			&changeHistory.Statement,
			&changeHistory.ExecutionDurationNs,
		); err != nil {
			return nil, err
		}
		version, err := model.NewVersion(storedVersion)
		if err != nil {
			return nil, err
		}
		changeHistory.Version = version

		list = append(list, &changeHistory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}
