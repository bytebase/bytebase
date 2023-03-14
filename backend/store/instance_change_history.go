package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// InstanceChangeHistoryMessage records the change history of an instance.
// it deprecates the old MigrationHistory.
type InstanceChangeHistoryMessage struct {
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64
	// nil means bytebase meta instance.
	InstanceID          *int
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
	ID      int64
	Deleted bool
}

// FindInstanceChangeHistoryMessage is for listing a list of instance change history.
type FindInstanceChangeHistoryMessage struct {
	ID         *int64
	InstanceID *int
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

// CreateInstanceChangeHistory creates instance change history in batch.
func (s *Store) CreateInstanceChangeHistory(ctx context.Context, creates ...*InstanceChangeHistoryMessage) ([]*InstanceChangeHistoryMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := s.createInstanceChangeHistoryImpl(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func (*Store) createInstanceChangeHistoryImpl(ctx context.Context, tx *Tx, creates ...*InstanceChangeHistoryMessage) ([]*InstanceChangeHistoryMessage, error) {
	if len(creates) == 0 {
		return nil, nil
	}
	var query strings.Builder
	var values []interface{}
	var queryValues []string

	_, _ = query.WriteString(`
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
		payload,
		created_ts,
		updated_ts
	) VALUES `)

	count := 1
	for _, create := range creates {
		if create.Payload == "" {
			create.Payload = "{}"
		}
		values = append(values,
			create.CreatorID,
			create.CreatorID,
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
		)
		const countToPayload = 17
		var valueStr []string
		for i := 0; i < countToPayload; i++ {
			valueStr = append(valueStr, fmt.Sprintf("$%d", count))
			count++
		}
		if create.CreatedTs == 0 {
			valueStr = append(valueStr, "DEFAULT")
		} else {
			valueStr = append(valueStr, fmt.Sprintf("$%d", count))
			values = append(values, create.CreatedTs)
			count++
		}
		if create.UpdatedTs == 0 {
			valueStr = append(valueStr, "DEFAULT")
		} else {
			valueStr = append(valueStr, fmt.Sprintf("$%d", count))
			values = append(values, create.UpdatedTs)
			count++
		}
		queryValues = append(queryValues, fmt.Sprintf("(%s)", strings.Join(valueStr, " , ")))
	}

	_, _ = query.WriteString(strings.Join(queryValues, ", "))
	_, _ = query.WriteString(` RETURNING id, created_ts`)

	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*InstanceChangeHistoryMessage

	i := 0
	for rows.Next() {
		var id, createdTs int64
		if err := rows.Scan(&id, &createdTs); err != nil {
			return nil, err
		}

		create := creates[i]
		list = append(list, &InstanceChangeHistoryMessage{
			CreatorID:           create.CreatorID,
			UpdaterID:           create.CreatorID,
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
			CreatedTs: createdTs,
			UpdatedTs: createdTs,
		})
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func convertInstanceChangeHistoryToMigrationHistory(change *InstanceChangeHistoryMessage) (*db.MigrationHistory, error) {
	var issueID string
	if v := change.IssueID; v != nil {
		issueID = strconv.Itoa(*v)
	}

	useSemanticVersion, version, semanticVersionSuffix, err := util.FromStoredVersion(change.Version)
	if err != nil {
		return nil, err
	}

	return &db.MigrationHistory{
		ID:                    strconv.FormatInt(change.ID, 10),
		Creator:               "",
		CreatedTs:             change.CreatedTs,
		Updater:               "",
		UpdatedTs:             change.UpdatedTs,
		ReleaseVersion:        change.ReleaseVersion,
		Namespace:             "",
		Sequence:              int(change.Sequence),
		Source:                change.Source,
		Type:                  change.Type,
		Status:                change.Status,
		Version:               version,
		Description:           change.Description,
		Statement:             change.Statement,
		Schema:                change.Schema,
		SchemaPrev:            change.SchemaPrev,
		ExecutionDurationNs:   change.ExecutionDurationNs,
		IssueID:               issueID,
		Payload:               change.Payload,
		UseSemanticVersion:    useSemanticVersion,
		SemanticVersionSuffix: semanticVersionSuffix,
	}, nil
}

// FindInstanceChangeHistoryList finds a list of instance change history and returns as a list of migration history.
func (s *Store) FindInstanceChangeHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	findMessage := &FindInstanceChangeHistoryMessage{
		InstanceID: find.InstanceID,
		DatabaseID: find.DatabaseID,
		Source:     find.Source,
		Version:    find.Version,
		Limit:      find.Limit,
	}
	if v := find.ID; v != nil {
		id, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, err
		}
		findMessage.ID = &id
	}

	list, err := s.ListInstanceChangeHistory(ctx, findMessage)
	if err != nil {
		return nil, err
	}
	var migrationHistoryList []*db.MigrationHistory
	for _, change := range list {
		migrationHistory, err := convertInstanceChangeHistoryToMigrationHistory(change)
		if err != nil {
			return nil, err
		}
		if change.DatabaseID != nil {
			database, err := s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: change.DatabaseID})
			if err != nil {
				return nil, err
			}
			migrationHistory.Namespace = database.DatabaseName
		}
		creator, err := s.GetPrincipalByID(ctx, change.CreatorID)
		if err != nil {
			return nil, err
		}
		migrationHistory.Creator = creator.Name
		updater, err := s.GetPrincipalByID(ctx, change.UpdaterID)
		if err != nil {
			return nil, err
		}
		migrationHistory.Updater = updater.Name
		migrationHistoryList = append(migrationHistoryList, migrationHistory)
	}

	return migrationHistoryList, nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

// UpdateInstanceChangeHistoryAsDone updates a change history to done.
func (s *Store) UpdateInstanceChangeHistoryAsDone(ctx context.Context, migrationDurationNs int64, updatedSchema string, insertedID string) error {
	status := db.Done
	id, err := strconv.ParseInt(insertedID, 10, 64)
	if err != nil {
		return err
	}
	update := &UpdateInstanceChangeHistoryMessage{
		ID:                  id,
		ExecutionDurationNs: &migrationDurationNs,
		Status:              &status,
		Schema:              &updatedSchema,
	}
	return s.UpdateInstanceChangeHistory(ctx, update)
}

// UpdateInstanceChangeHistoryAsFailed updates a change history to failed.
func (s *Store) UpdateInstanceChangeHistoryAsFailed(ctx context.Context, migrationDurationNs int64, insertedID string) error {
	status := db.Failed
	id, err := strconv.ParseInt(insertedID, 10, 64)
	if err != nil {
		return err
	}
	update := &UpdateInstanceChangeHistoryMessage{
		ID:                  id,
		ExecutionDurationNs: &migrationDurationNs,
		Status:              &status,
	}
	return s.UpdateInstanceChangeHistory(ctx, update)
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

func (*Store) getLargestInstanceChangeHistorySequenceImpl(ctx context.Context, tx *Tx, instanceID *int, databaseID *int, baseline bool) (int64, error) {
	query := `
	SELECT
		MAX(sequence)
	FROM instance_change_history
	WHERE instance_id = $1 AND database_id = $2`
	if baseline {
		query += fmt.Sprintf(" AND (type = '%s' OR type = '%s')", db.Baseline, db.Branch)
	}
	var sequence sql.NullInt64
	if err := tx.QueryRowContext(ctx, query, instanceID, databaseID).Scan(&sequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return -1, util.FormatErrorWithQuery(err, query)
	}

	if sequence.Valid {
		return sequence.Int64, nil
	}

	return 0, nil
}

// GetLargestInstanceChangeHistorySequence will get the largest sequence number.
func (s *Store) GetLargestInstanceChangeHistorySequence(ctx context.Context, instanceID *int, databaseID *int, baseline bool) (int64, error) {
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
func (s *Store) GetLargestInstanceChangeHistoryVersionSinceBaseline(ctx context.Context, instanceID *int, databaseID *int) (*string, error) {
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

	var version sql.NullString
	if err := tx.QueryRowContext(ctx, query, instanceID, databaseID, sequence).Scan(&version); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if version.Valid {
		return &version.String, nil
	}

	return nil, nil
}

// CreatePendingInstanceChangeHistory creates an instance change history.
// it deprecates the old InsertPendingHistory.
func (s *Store) CreatePendingInstanceChangeHistory(ctx context.Context, sequence int64, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	list, err := s.createInstanceChangeHistoryImpl(ctx, tx, &InstanceChangeHistoryMessage{
		CreatorID:           m.CreatorID,
		InstanceID:          m.InstanceID,
		DatabaseID:          m.DatabaseID,
		IssueID:             m.IssueIDInt,
		ReleaseVersion:      m.ReleaseVersion,
		Sequence:            sequence,
		Source:              m.Source,
		Type:                m.Type,
		Status:              db.Pending,
		Version:             storedVersion,
		Description:         m.Description,
		Statement:           statement,
		Schema:              prevSchema,
		SchemaPrev:          prevSchema,
		ExecutionDurationNs: 0,
		Payload:             m.Payload,
	})
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", list[0].ID), nil
}

// ListInstanceHavingInstanceChangeHistory finds the instance id lists that have instance change history.
func (s *Store) ListInstanceHavingInstanceChangeHistory(ctx context.Context) ([]int, error) {
	query := `
	SELECT DISTINCT
		instance_id
	FROM instance_change_history
	`
	rows, err := s.db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		list = append(list, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}
