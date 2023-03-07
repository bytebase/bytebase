package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// InstanceChangeHistoryMessage records the change history of an instance.
// it deprecates the old MigrationHistory.
type InstanceChangeHistoryMessage struct {
	InstanceID          int
	DatabaseID          *int
	IssueID             *int
	ReleaseVersion      string
	Sequence            int64
	Source              db.MigrationSource
	Type                db.MigrationType
	Status              db.MigrationStatus
	Version             string
	Description         string
	Statement           string
	Schema              string
	SchemaPrev          string
	ExecutionDurationNs int64
	Payload             string

	// Output only
	ID        int64
	Deleted   bool
	CreatedTs int64
	UpdatedTs int64
	CreatorID int
	UpdaterID int
}

// FindInstanceChangeHistoryMessage is for listing a list of instance change history.
type FindInstanceChangeHistoryMessage struct {
	ID         *int64
	InstanceID int
	DatabaseID *int
	Source     *db.MigrationSource
	Version    *string
	Limit      *int
}

// UpdateInstanceChangeHistoryMessage is for updating an instance change history.
type UpdateInstanceChangeHistoryMessage struct {
	ID int64

	Status              *db.MigrationStatus
	ExecutionDurationNs *int64
	Schema              *string
}

func (*Store) createInstanceChangeHistoryImpl(ctx context.Context, tx *Tx, create *InstanceChangeHistoryMessage, creatorID int) (*InstanceChangeHistoryMessage, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}

	query := `
    INSERT INTO instance_change_history (
      creator_id,
      updater_id,
      instance_id,
      database_id,
      issue_id,
      release_version,
      sequence,
      source,
      type,
      status,
      version,
      description,
      statement,
      "schema",
      schema_prev,
      execution_duration_ns,
      payload
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
    RETURNING id, created_ts
  `

	var id, createdTs int64
	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.InstanceID,
		create.DatabaseID,
		create.IssueID,
		create.ReleaseVersion,
		create.Sequence,
		create.Source,
		create.Type,
		create.Status,
		create.Version,
		create.Description,
		create.Statement,
		create.Schema,
		create.SchemaPrev,
		create.ExecutionDurationNs,
		create.Payload,
	).Scan(&id, &createdTs); err != nil {
		return nil, err
	}

	changeHistory := &InstanceChangeHistoryMessage{
		InstanceID:          create.InstanceID,
		DatabaseID:          create.DatabaseID,
		IssueID:             create.IssueID,
		ReleaseVersion:      create.ReleaseVersion,
		Sequence:            create.Sequence,
		Source:              create.Source,
		Type:                create.Type,
		Status:              create.Status,
		Version:             create.Version,
		Description:         create.Description,
		Statement:           create.Statement,
		Schema:              create.Schema,
		SchemaPrev:          create.SchemaPrev,
		ExecutionDurationNs: create.ExecutionDurationNs,
		Payload:             create.Payload,

		ID:        id,
		CreatorID: creatorID,
		CreatedTs: createdTs,
		UpdaterID: creatorID,
		UpdatedTs: createdTs,
	}

	return changeHistory, nil
}

// ListInstanceChangeHistory finds the instance change history.
func (s *Store) ListInstanceChangeHistory(ctx context.Context, find *FindInstanceChangeHistoryMessage) ([]*InstanceChangeHistoryMessage, error) {
	where, args := []string{"instance_id = $1"}, []interface{}{find.InstanceID}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Version; v != nil {
		where, args = append(where, fmt.Sprintf("version = $%d", len(args)+1)), append(args, *v)
	}

	query := `
  SELECT
    id,
    row_status,
    creator_id,
    created_ts,
    updater_id,
    updated_ts,
    instance_id,
    database_id,
    issue_id,
    release_version,
    sequence,
    source,
    type,
    status,
    version,
    description,
    statement,
    schema,
    schema_prev,
    execution_duration_ns,
    payload
  FROM instance_change_history
  WHERE ` + strings.Join(where, " AND ") + ` ORDER BY instance_id, database_id, sequence DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

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
		var rowStatus string
		if err := rows.Scan(
			&changeHistory.ID,
			&rowStatus,
			&changeHistory.CreatorID,
			&changeHistory.CreatedTs,
			&changeHistory.UpdaterID,
			&changeHistory.UpdatedTs,
			&changeHistory.InstanceID,
			&changeHistory.DatabaseID,
			&changeHistory.IssueID,
			&changeHistory.ReleaseVersion,
			&changeHistory.Sequence,
			&changeHistory.Source,
			&changeHistory.Type,
			&changeHistory.Status,
			&changeHistory.Version,
			&changeHistory.Description,
			&changeHistory.Statement,
			&changeHistory.Schema,
			&changeHistory.SchemaPrev,
			&changeHistory.ExecutionDurationNs,
			&changeHistory.Payload,
		); err != nil {
			return nil, err
		}

		changeHistory.Deleted = convertRowStatusToDeleted(rowStatus)
		list = append(list, &changeHistory)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

// UpdateInstanceChangeHistory updates an instance change history.
// it deprecates the old UpdateHistoryAsDone and UpdateHistoryAsFailed.
func (s *Store) UpdateInstanceChangeHistory(ctx context.Context, update *UpdateInstanceChangeHistoryMessage) error {
	set, args := []string{}, []interface{}{}
	if v := update.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.ExecutionDurationNs; v != nil {
		set, args = append(set, fmt.Sprintf("execution_duration_ns = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.Schema; v != nil {
		set, args = append(set, fmt.Sprintf("schema = $%d", len(args)+1)), append(args, *v)
	}
	query := `
  UPDATE instance_change_history
  SET` + strings.Join(set, ", ") + `
  WHERE` + fmt.Sprintf("id = $%d", len(args)+1)
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

func (*Store) getLargestInstanceChangeHistorySequenceImpl(ctx context.Context, tx *Tx, instanceID int, databaseID *int, baseline bool) (int64, error) {
	query := `
    SELECT
      MAX(sequence)
    FROM instance_change_history
    WHERE instance_id = $1 AND database_id = $1`
	if baseline {
		query += fmt.Sprintf(" AND (type = '%s' OR type = '%s')", db.Baseline, db.Branch)
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, query, instanceID, databaseID).Scan(&sequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return -1, util.FormatErrorWithQuery(err, query)
	}

	return sequence, nil
}

// GetLargestInstanceChangeHistorySequence will get the largest sequence number.
func (s *Store) GetLargestInstanceChangeHistorySequence(ctx context.Context, instanceID int, databaseID *int, baseline bool) (int64, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	sequence, err := s.getLargestInstanceChangeHistorySequenceImpl(ctx, tx, instanceID, databaseID, baseline)
	if err != nil {
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return sequence, nil
}

// GetLargestInstanceChangeHistoryVersionSinceBaseline will get the largest version since last baseline or branch.
func (s *Store) GetLargestInstanceChangeHistoryVersionSinceBaseline(ctx context.Context, instanceID int, databaseID *int) (*string, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sequence, err := s.getLargestInstanceChangeHistorySequenceImpl(ctx, tx, instanceID, databaseID, true /* baseline */)
	if err != nil {
		return nil, err
	}

	query := `
  SELECT
    MAX(version)
  FROM instance_change_history
  WHERE instance_id = $1 AND database_id = $2 AND sequence >= $3`

	var version string
	if err := tx.QueryRowContext(ctx, query, instanceID, databaseID, sequence).Scan(&version); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &version, nil
}

// CreatePendingInstanceChangeHistory creates an instance change history.
// it deprecates the old InsertPendingHistory.
func (s *Store) CreatePendingInstanceChangeHistory(ctx context.Context, sequence int64, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	h, err := s.createInstanceChangeHistoryImpl(ctx, tx, &InstanceChangeHistoryMessage{
		InstanceID:          m.InstanceID,
		DatabaseID:          m.DatabaseID,
		IssueID:             m.IssueIDInt,
		ReleaseVersion:      m.ReleaseVersion,
		Sequence:            sequence,
		Source:              m.Source,
		Type:                m.Type,
		Status:              m.Status,
		Version:             storedVersion,
		Description:         m.Description,
		Statement:           statement,
		Schema:              prevSchema,
		SchemaPrev:          prevSchema,
		ExecutionDurationNs: 0,
		Payload:             m.Payload,
	}, m.CreatorID)
	if err != nil {
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return h.ID, nil
}
