package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// InstanceChangeHistoryMessage records the change history of an instance.
// it deprecates the old MigrationHistory.
type InstanceChangeHistoryMessage struct {
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64
	// nil means bytebase meta instance.
	InstanceUID         *int
	DatabaseUID         *int
	IssueUID            *int
	ReleaseVersion      string
	Sequence            int64
	Source              db.MigrationSource
	Type                db.MigrationType
	Status              db.MigrationStatus
	Version             string
	Description         string
	Statement           string
	SheetID             *int
	Schema              string
	SchemaPrev          string
	ExecutionDurationNs int64
	Payload             *storepb.InstanceChangeHistoryPayload

	// Output only
	UID            string
	Deleted        bool
	Creator        *UserMessage
	Updater        *UserMessage
	InstanceID     string
	DatabaseName   string
	IssueProjectID string
}

// instanceChangeHistoryTruncateLength is the maximum size (1M) of a sheet for displaying.
const instanceChangeHistoryTruncateLength = 1024 * 1024

// FindInstanceChangeHistoryMessage is for listing a list of instance change history.
type FindInstanceChangeHistoryMessage struct {
	ID         *int64
	InstanceID *int
	DatabaseID *int
	SheetID    *int
	Source     *db.MigrationSource
	Version    *string
	Limit      *int
	Offset     *int

	// Truncate Statement, Schema, SchemaPrev unless ShowFull.
	ShowFull bool
}

// UpdateInstanceChangeHistoryMessage is for updating an instance change history.
type UpdateInstanceChangeHistoryMessage struct {
	ID string

	Status              *db.MigrationStatus
	ExecutionDurationNs *int64
	Schema              *string
	Sheet               *int
}

// CreateInstanceChangeHistory creates instance change history in batch.
func (s *Store) CreateInstanceChangeHistory(ctx context.Context, create *InstanceChangeHistoryMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if create.InstanceUID == nil {
		if _, err := s.createInstanceChangeHistoryImplForMigrator(ctx, tx, create); err != nil {
			return err
		}
	} else {
		if _, err := s.createInstanceChangeHistoryImpl(ctx, tx, create); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (*Store) createInstanceChangeHistoryImpl(ctx context.Context, tx *Tx, create *InstanceChangeHistoryMessage) (string, error) {
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
			sheet_id,
			schema_prev,
			execution_duration_ns,
			payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id`

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return "", err
	}

	var uid string
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.InstanceUID,
		create.DatabaseUID,
		create.IssueUID,
		create.ReleaseVersion,
		create.Sequence,
		create.Source,
		create.Type,
		create.Status,
		create.Version,
		create.Description,
		create.Statement,
		create.Schema,
		create.SheetID,
		create.SchemaPrev,
		create.ExecutionDurationNs,
		payload,
	).Scan(&uid); err != nil {
		return "", err
	}

	return uid, nil
}

func (*Store) createInstanceChangeHistoryImplForMigrator(ctx context.Context, tx *Tx, create *InstanceChangeHistoryMessage) (string, error) {
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
		RETURNING id`

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return "", err
	}

	var uid string
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.InstanceUID,
		create.DatabaseUID,
		create.IssueUID,
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
		payload,
	).Scan(&uid); err != nil {
		return "", err
	}

	return uid, nil
}

func convertInstanceChangeHistoryToMigrationHistory(change *InstanceChangeHistoryMessage) (*db.MigrationHistory, error) {
	var issueID string
	if v := change.IssueUID; v != nil {
		issueID = strconv.Itoa(*v)
	}

	useSemanticVersion, version, semanticVersionSuffix, err := util.FromStoredVersion(change.Version)
	if err != nil {
		return nil, err
	}

	payload, err := protojson.Marshal(change.Payload)
	if err != nil {
		return nil, err
	}

	return &db.MigrationHistory{
		ID:                    change.UID,
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
		SheetID:               change.SheetID,
		SchemaPrev:            change.SchemaPrev,
		ExecutionDurationNs:   change.ExecutionDurationNs,
		IssueID:               issueID,
		Payload:               string(payload),
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
		ShowFull:   true,
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
		if change.DatabaseUID != nil {
			database, err := s.GetDatabaseV2(ctx, &FindDatabaseMessage{UID: change.DatabaseUID})
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
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.id = $%d", len(args)+1)), append(args, *v)
	}
	sheetField := "instance_change_history.sheet_id"
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.instance_id = $%d", len(args)+1)), append(args, *v)
	} else {
		where = append(where, "instance_change_history.instance_id is NULL AND instance_change_history.database_id is NULL")
		sheetField = "NULL"
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SheetID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.sheet_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Version; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.version = $%d", len(args)+1)), append(args, *v)
	}

	statementField := fmt.Sprintf("LEFT(instance_change_history.statement, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		statementField = "instance_change_history.statement"
	}
	schemaField := fmt.Sprintf("LEFT(instance_change_history.schema, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		schemaField = "instance_change_history.schema"
	}
	schemaPrevField := fmt.Sprintf("LEFT(instance_change_history.schema_prev, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		schemaPrevField = "instance_change_history.schema_prev"
	}

	query := fmt.Sprintf(`
		SELECT
			instance_change_history.id,
			instance_change_history.row_status,
			instance_change_history.creator_id,
			instance_change_history.created_ts,
			instance_change_history.updater_id,
			instance_change_history.updated_ts,
			instance_change_history.instance_id,
			instance_change_history.database_id,
			instance_change_history.issue_id,
			instance_change_history.release_version,
			instance_change_history.sequence,
			instance_change_history.source,
			instance_change_history.type,
			instance_change_history.status,
			instance_change_history.version,
			instance_change_history.description,
			%s,
			%s,
			%s,
			%s,
			instance_change_history.execution_duration_ns,
			instance_change_history.payload,
			COALESCE(instance.resource_id, ''),
			COALESCE(db.name, '')
		FROM instance_change_history
		LEFT JOIN instance on instance.id = instance_change_history.instance_id
		LEFT JOIN db on db.id = instance_change_history.database_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY instance_change_history.instance_id, instance_change_history.database_id, instance_change_history.sequence DESC`, statementField, schemaField, schemaPrevField, sheetField)
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
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
		var rowStatus, payload string
		var instanceID, databaseID, issueID, sheetID sql.NullInt32
		if err := rows.Scan(
			&changeHistory.UID,
			&rowStatus,
			&changeHistory.CreatorID,
			&changeHistory.CreatedTs,
			&changeHistory.UpdaterID,
			&changeHistory.UpdatedTs,
			&instanceID,
			&databaseID,
			&issueID,
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
			&sheetID,
			&changeHistory.ExecutionDurationNs,
			&payload,
			&changeHistory.InstanceID,
			&changeHistory.DatabaseName,
		); err != nil {
			return nil, err
		}
		if instanceID.Valid {
			n := int(instanceID.Int32)
			changeHistory.InstanceUID = &n
		}
		if databaseID.Valid {
			n := int(databaseID.Int32)
			changeHistory.DatabaseUID = &n
		}
		if issueID.Valid {
			n := int(issueID.Int32)
			changeHistory.IssueUID = &n
		}
		if sheetID.Valid {
			n := int(sheetID.Int32)
			changeHistory.SheetID = &n
		}
		changeHistory.Payload = &storepb.InstanceChangeHistoryPayload{}
		if err := protojson.Unmarshal([]byte(payload), changeHistory.Payload); err != nil {
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

	for _, changeHistory := range list {
		creator, err := s.GetUserByID(ctx, changeHistory.CreatorID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get creator by creatorID %q", changeHistory.CreatorID)
		}
		changeHistory.Creator = creator
		updater, err := s.GetUserByID(ctx, changeHistory.UpdaterID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get updater by updaterID %q", changeHistory.UpdaterID)
		}
		changeHistory.Updater = updater
		if changeHistory.IssueUID != nil {
			issue, err := s.GetIssueV2(ctx, &FindIssueMessage{UID: changeHistory.IssueUID})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get issue by issueUID %q", *changeHistory.IssueUID)
			}
			changeHistory.IssueProjectID = issue.Project.ResourceID
		}
	}

	return list, nil
}

// GetInstanceChangeHistory gets the instance change history.
func (s *Store) GetInstanceChangeHistory(ctx context.Context, find *FindInstanceChangeHistoryMessage) (*InstanceChangeHistoryMessage, error) {
	list, err := s.ListInstanceChangeHistory(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > 1 {
		return nil, errors.Errorf("expected 1 change history, got %d", len(list))
	}

	return list[0], nil
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

func (*Store) getNextInstanceChangeHistorySequence(ctx context.Context, tx *Tx, instanceID *int, databaseID *int) (int64, error) {
	where, args := []string{"TRUE"}, []any{}
	if instanceID != nil && databaseID != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *instanceID)
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *databaseID)
	} else {
		where = append(where, "instance_id is NULL AND database_id is NULL")
	}

	query := `
		SELECT
			COALESCE(MAX(sequence), 0)+1
		FROM instance_change_history
		WHERE ` + strings.Join(where, " AND ")
	var sequence int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

// CreatePendingInstanceChangeHistory creates an instance change history.
// it deprecates the old InsertPendingHistory.
func (s *Store) CreatePendingInstanceChangeHistory(ctx context.Context, prevSchema string, m *db.MigrationInfo, storedVersion, statement string, sheetID *int) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	nextSequence, err := s.getNextInstanceChangeHistorySequence(ctx, tx, m.InstanceID, m.DatabaseID)
	if err != nil {
		return "", err
	}
	instanceChange := &InstanceChangeHistoryMessage{
		CreatorID:           m.CreatorID,
		InstanceUID:         m.InstanceID,
		DatabaseUID:         m.DatabaseID,
		IssueUID:            m.IssueIDInt,
		ReleaseVersion:      m.ReleaseVersion,
		Sequence:            nextSequence,
		Source:              m.Source,
		Type:                m.Type,
		Status:              db.Pending,
		Version:             storedVersion,
		Description:         m.Description,
		Statement:           statement,
		SheetID:             sheetID,
		Schema:              prevSchema,
		SchemaPrev:          prevSchema,
		ExecutionDurationNs: 0,
		Payload:             m.Payload,
	}
	var uid string
	if instanceChange.InstanceUID == nil {
		id, err := s.createInstanceChangeHistoryImplForMigrator(ctx, tx, instanceChange)
		if err != nil {
			return "", err
		}
		uid = id
	} else {
		id, err := s.createInstanceChangeHistoryImpl(ctx, tx, instanceChange)
		if err != nil {
			return "", err
		}
		uid = id
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return uid, nil
}

// ListInstanceChangeHistoryForMigrator finds the instance change history for the migrator,
// the users are not composed.
// The sheet_id is not loaded.
func (s *Store) ListInstanceChangeHistoryForMigrator(ctx context.Context, find *FindInstanceChangeHistoryMessage) ([]*InstanceChangeHistoryMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.instance_id = $%d", len(args)+1)), append(args, *v)
	} else {
		where = append(where, "instance_change_history.instance_id is NULL AND instance_change_history.database_id is NULL")
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Version; v != nil {
		where, args = append(where, fmt.Sprintf("instance_change_history.version = $%d", len(args)+1)), append(args, *v)
	}

	statementField := fmt.Sprintf("LEFT(instance_change_history.statement, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		statementField = "instance_change_history.statement"
	}
	schemaField := fmt.Sprintf("LEFT(instance_change_history.schema, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		schemaField = "instance_change_history.schema"
	}
	schemaPrevField := fmt.Sprintf("LEFT(instance_change_history.schema_prev, %d)", instanceChangeHistoryTruncateLength)
	if find.ShowFull {
		schemaPrevField = "instance_change_history.schema_prev"
	}

	query := fmt.Sprintf(`
		SELECT
			instance_change_history.id,
			instance_change_history.row_status,
			instance_change_history.creator_id,
			instance_change_history.created_ts,
			instance_change_history.updater_id,
			instance_change_history.updated_ts,
			instance_change_history.instance_id,
			instance_change_history.database_id,
			instance_change_history.issue_id,
			instance_change_history.release_version,
			instance_change_history.sequence,
			instance_change_history.source,
			instance_change_history.type,
			instance_change_history.status,
			instance_change_history.version,
			instance_change_history.description,
			%s,
			%s,
			%s,
			instance_change_history.execution_duration_ns,
			instance_change_history.payload,
			COALESCE(instance.resource_id, ''),
			COALESCE(db.name, '')
		FROM instance_change_history
		LEFT JOIN instance on instance.id = instance_change_history.instance_id
		LEFT JOIN db on db.id = instance_change_history.database_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY instance_change_history.instance_id, instance_change_history.database_id, instance_change_history.sequence DESC`, statementField, schemaField, schemaPrevField)
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
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
		var rowStatus, payload string
		var instanceID, databaseID, issueID sql.NullInt32
		if err := rows.Scan(
			&changeHistory.UID,
			&rowStatus,
			&changeHistory.CreatorID,
			&changeHistory.CreatedTs,
			&changeHistory.UpdaterID,
			&changeHistory.UpdatedTs,
			&instanceID,
			&databaseID,
			&issueID,
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
			&payload,
			&changeHistory.InstanceID,
			&changeHistory.DatabaseName,
		); err != nil {
			return nil, err
		}
		if instanceID.Valid {
			n := int(instanceID.Int32)
			changeHistory.InstanceUID = &n
		}
		if databaseID.Valid {
			n := int(databaseID.Int32)
			changeHistory.DatabaseUID = &n
		}
		if issueID.Valid {
			n := int(issueID.Int32)
			changeHistory.IssueUID = &n
		}
		changeHistory.Payload = &storepb.InstanceChangeHistoryPayload{}
		if err := protojson.Unmarshal([]byte(payload), changeHistory.Payload); err != nil {
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
