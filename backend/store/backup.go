package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store/model"
)

// BackupSettingsMatchMessage is the message to find backup settings matching the conditions.
type BackupSettingsMatchMessage struct {
	Hour      int
	DayOfWeek int
}

// FindBackupSettingsMatch finds a list of backup setting instances with match conditions.
func (s *Store) FindBackupSettingsMatch(ctx context.Context, match *BackupSettingsMatchMessage) ([]*BackupSettingMessage, error) {
	backupSettingList, err := s.findBackupSettingsMatchImpl(ctx, match)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find matching backup setting list with BackupSettingsMatch[%+v]", match)
	}

	return backupSettingList, nil
}

// findBackupSettingsMatchImpl retrieves a list of backup settings based on match condition.
func (s *Store) findBackupSettingsMatchImpl(ctx context.Context, match *BackupSettingsMatchMessage) ([]*BackupSettingMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT
			database_id,
			enabled,
			hour,
			day_of_week,
			retention_period_ts,
			hook_url
		FROM backup_setting
		WHERE
			enabled = true
			AND (
				(hour = $1 AND day_of_week = $2)
				OR
				(hour = $3 AND day_of_week = -1)
				OR
				(hour = -1 AND day_of_week = $4)
			)
		`,
		match.Hour, match.DayOfWeek, match.Hour, match.DayOfWeek,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupSettingList.
	var backupSettingList []*BackupSettingMessage
	for rows.Next() {
		var backupSetting BackupSettingMessage
		if err := rows.Scan(
			&backupSetting.DatabaseUID,
			&backupSetting.Enabled,
			&backupSetting.HourOfDay,
			&backupSetting.DayOfWeek,
			&backupSetting.RetentionPeriodTs,
			&backupSetting.HookURL,
		); err != nil {
			return nil, err
		}

		backupSettingList = append(backupSettingList, &backupSetting)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return backupSettingList, nil
}

// BackupSettingMessage is the message for backup setting.
type BackupSettingMessage struct {
	// ID is the ID of the backup setting.
	ID int
	// DatabaseUID is the UID of the database.
	DatabaseUID int
	// UpdatedTs is the timestamp when the backup setting is updated.
	UpdatedTs int64
	// Enable is true if the backup setting is enabled.
	Enabled bool
	// HourOfDay is the hour field in cron string to trigger the backup.
	HourOfDay int
	// DayOfWeek is the day of week field in cron string to trigger the backup.
	DayOfWeek int
	// RetentionPeriodTs is the retention period in seconds.
	RetentionPeriodTs int
	// HookURL is the URL to send the backup status.
	HookURL string
}

// FindBackupSettingMessage is the message for finding backup setting.
type FindBackupSettingMessage struct {
	// DatabaseUID is the UID of database.
	DatabaseUID *int
	// InstanceUID is the UID of instance.
	InstanceUID *int
}

// BackupMessage is the message for backup.
type BackupMessage struct {
	// Name is the name of the backup.
	Name string
	// Status is the status of the backup.
	Status api.BackupStatus
	// BackupType is the type of the backup.
	BackupType api.BackupType
	// Comment is the comment of the backup.
	Comment string
	// Storage Backend is the storage backend of the backup.
	StorageBackend api.BackupStorageBackend
	// MigrationHistoryVersion is the migration history version of the database.
	MigrationHistoryVersion model.Version
	// Path is the path of the backup file.
	Path string

	// Output only fields.
	//
	// ID is the UID of the backup.
	UID int
	// CreatedTs is the timestamp when the backup is created.
	CreatedTs int64
	// UpdatedTs is the timestamp when the backup is updated.
	UpdatedTs int64
	// RowStatus is the status of the row. ARCHIVED means the backup is deleted.
	RowStatus api.RowStatus
	// DatabaseUID is the UID of the database.
	DatabaseUID int
	// Payload is the payload of the backup.
	Payload api.BackupPayload
}

// SLogBackupArray is a helper to format []*BackupMessage.
type SLogBackupArray []*BackupMessage

// LogValue implements the LogValuer interface.
func (backups SLogBackupArray) LogValue() slog.Value {
	backupList := []string{}
	for _, backup := range backups {
		payload, err := json.Marshal(backup.Payload)
		if err != nil {
			backupList = append(backupList, err.Error())
			continue
		}
		backupList = append(backupList, fmt.Sprintf("{name:%s, id:%d, payload:%s}", backup.Name, backup.UID, payload))
	}
	return slog.StringValue(strings.Join(backupList, ","))
}

// FindBackupMessage is the message for finding backup.
type FindBackupMessage struct {
	// DatabaseUID is the UID of the database.
	DatabaseUID *int
	// Name is the name of the backup.
	Name *string
	// RowStatus is the status of the row.
	RowStatus *api.RowStatus
	// Status is the status of the backup.
	Status *api.BackupStatus
	// backupUID is the UID of the backup.
	backupUID *int
}

// UpdateBackupMessage is the message for updating backup.
type UpdateBackupMessage struct {
	// UID is the UID of the backup.
	UID int

	// Standard fields
	RowStatus *api.RowStatus
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  *string
	Comment *string
	Payload *string
}

// GetBackupSettingV2 retrieves the backup setting for the given database.
func (s *Store) GetBackupSettingV2(ctx context.Context, databaseUID int) (*BackupSettingMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	find := &FindBackupSettingMessage{
		DatabaseUID: &databaseUID,
	}

	backupSettings, err := s.listBackupSettingImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup setting with %+v", find)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(backupSettings) == 0 {
		return nil, nil
	}
	if len(backupSettings) > 1 {
		return nil, errors.Wrapf(err, "find %d backup settings with %+v", len(backupSettings), find)
	}

	return backupSettings[0], nil
}

// UpsertBackupSettingV2 upserts the backup setting for the given database.
func (s *Store) UpsertBackupSettingV2(ctx context.Context, principalUID int, upsert *BackupSettingMessage) (*BackupSettingMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var backupSetting BackupSettingMessage
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO backup_setting (
			creator_id,
			updater_id,
			database_id,
			enabled,
			hour,
			day_of_week,
			retention_period_ts,
			hook_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (database_id)
		DO UPDATE SET
			enabled = EXCLUDED.enabled,
			hour = EXCLUDED.hour,
			day_of_week = EXCLUDED.day_of_week,
			retention_period_ts = EXCLUDED.retention_period_ts,
			updater_id = EXCLUDED.updater_id,
			hook_url = EXCLUDED.hook_url
		RETURNING id, database_id, updated_ts, enabled, hour, day_of_week, retention_period_ts, hook_url
		`,
		principalUID,
		principalUID,
		upsert.DatabaseUID,
		upsert.Enabled,
		upsert.HourOfDay,
		upsert.DayOfWeek,
		upsert.RetentionPeriodTs,
		upsert.HookURL,
	).Scan(
		&backupSetting.ID,
		&backupSetting.DatabaseUID,
		&backupSetting.UpdatedTs,
		&backupSetting.Enabled,
		&backupSetting.HourOfDay,
		&backupSetting.DayOfWeek,
		&backupSetting.RetentionPeriodTs,
		&backupSetting.HookURL,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &backupSetting, nil
}

// UpdateBackupSettingsForEnvironmentV2 upserts an instance of backup setting.
func (s *Store) UpdateBackupSettingsForEnvironmentV2(ctx context.Context, environmentID string, enabled bool, updaterID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt := `
		UPDATE backup_setting
		SET
			updater_id = $1,
			enabled = $2
		WHERE id IN (
			SELECT backup_setting.id
			FROM backup_setting
			INNER JOIN db ON backup_setting.database_id = db.id
			INNER JOIN instance ON db.instance_id = instance.id
			INNER JOIN environment ON instance.environment_id = environment.id
			WHERE environment.resource_id = $3
		);
	`
	if _, err := tx.ExecContext(ctx, stmt, updaterID, enabled, environmentID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListBackupSettingV2 finds the backup setting.
func (s *Store) ListBackupSettingV2(ctx context.Context, find *FindBackupSettingMessage) ([]*BackupSettingMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	backupSettings, err := s.listBackupSettingImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup setting with %+v", find)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return backupSettings, nil
}

// CreateBackupV2 creates a backup for the given database.
func (s *Store) CreateBackupV2(ctx context.Context, create *BackupMessage, databaseUID int, principalUID int) (*BackupMessage, error) {
	storedVersion, err := create.MigrationHistoryVersion.Marshal()
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	query := `
		INSERT INTO backup (
			creator_id,
			updater_id,
			database_id,
			name,
			status,
			type,
			storage_backend,
			migration_history_version,
			path,
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, row_status, name, storage_backend, path, created_ts, updated_ts, status, type, comment, database_id
	`
	var backup BackupMessage
	if err := tx.QueryRowContext(ctx, query,
		principalUID,
		principalUID,
		databaseUID,
		create.Name,
		create.Status,
		create.BackupType,
		create.StorageBackend,
		storedVersion,
		create.Path,
		create.Comment,
	).Scan(
		&backup.UID,
		&backup.RowStatus,
		&backup.Name,
		&backup.StorageBackend,
		&backup.Path,
		&backup.CreatedTs,
		&backup.UpdatedTs,
		&backup.Status,
		&backup.BackupType,
		&backup.Comment,
		&backup.DatabaseUID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &backup, nil
}

// GetBackupByUID gets the backup for the given database by backup UID.
func (s *Store) GetBackupByUID(ctx context.Context, backupUID int) (*BackupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	find := &FindBackupMessage{backupUID: &backupUID}
	backupList, err := s.listBackupImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with %+v", find)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(backupList) == 0 {
		return nil, nil
	}
	if len(backupList) > 1 {
		return nil, errors.Errorf("found %d backup with backup uid %d", len(backupList), backupUID)
	}

	return backupList[0], nil
}

// GetBackupV2 gets the backup for the given database.
func (s *Store) GetBackupV2(ctx context.Context, find *FindBackupMessage) (*BackupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	backupList, err := s.listBackupImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with %+v", find)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(backupList) == 0 {
		return nil, nil
	}
	if len(backupList) > 1 {
		return nil, errors.Errorf("found %d backup with backup find %+v", len(backupList), find)
	}

	return backupList[0], nil
}

// ListBackupV2 lists the backups for the given database.
func (s *Store) ListBackupV2(ctx context.Context, find *FindBackupMessage) ([]*BackupMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	backupList, err := s.listBackupImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with %+v", find)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return backupList, nil
}

// UpdateBackupV2 patches an instance of Backup.
func (s *Store) UpdateBackupV2(ctx context.Context, patch *UpdateBackupMessage) (*BackupMessage, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		if *v == "" {
			*v = "{}"
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.UID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	var backup BackupMessage
	var payload []byte
	var storedVersion string
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE backup
			SET `+strings.Join(set, ", ")+`
			WHERE id = $%d
			RETURNING id, row_status, created_ts, updated_ts, database_id, name, status, type, storage_backend, migration_history_version, path, comment, payload
		`, len(args)),
		args...,
	).Scan(
		&backup.UID,
		&backup.RowStatus,
		&backup.CreatedTs,
		&backup.UpdatedTs,
		&backup.DatabaseUID,
		&backup.Name,
		&backup.Status,
		&backup.BackupType,
		&backup.StorageBackend,
		&storedVersion,
		&backup.Path,
		&backup.Comment,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("backup ID not found: %d", patch.UID)}
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	version, err := model.NewVersion(storedVersion)
	if err != nil {
		return nil, err
	}
	backup.MigrationHistoryVersion = version

	if err := json.Unmarshal(payload, &backup.Payload); err != nil {
		return nil, err
	}

	return &backup, nil
}

func (*Store) listBackupImplV2(ctx context.Context, tx *Tx, find *FindBackupMessage) ([]*BackupMessage, error) {
	// Build where clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.backupUID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		where, args = append(where, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			row_status,
			name,
			storage_backend,
			migration_history_version,
			path,
			created_ts,
			updated_ts,
			status,
			type,
			comment,
			database_id,
			payload
		FROM backup WHERE %s;`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backupList []*BackupMessage
	for rows.Next() {
		var backup BackupMessage
		var storedVersion, payload string
		if err := rows.Scan(
			&backup.UID,
			&backup.RowStatus,
			&backup.Name,
			&backup.StorageBackend,
			&storedVersion,
			&backup.Path,
			&backup.CreatedTs,
			&backup.UpdatedTs,
			&backup.Status,
			&backup.BackupType,
			&backup.Comment,
			&backup.DatabaseUID,
			&payload,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &backup.Payload); err != nil {
			return nil, err
		}
		version, err := model.NewVersion(storedVersion)
		if err != nil {
			return nil, err
		}
		backup.MigrationHistoryVersion = version

		backupList = append(backupList, &backup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return backupList, nil
}

func (*Store) listBackupSettingImplV2(ctx context.Context, tx *Tx, find *FindBackupSettingMessage) ([]*BackupSettingMessage, error) {
	// Build WHERE and JOIN clause.
	where, args := []string{"TRUE"}, []any{}
	var join []string
	if v := find.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("backup_setting.database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceUID; v != nil {
		// Relation backup_setting doesn't have instance_id column, so we need to join the db table to get the instance_id.
		join = append(join, "JOIN db ON db.id = backup_setting.database_id")
		where, args = append(where, fmt.Sprintf("db.instance_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			backup_setting.id,
			backup_setting.updated_ts,
			backup_setting.database_id,
			backup_setting.enabled,
			backup_setting.hour,
			backup_setting.day_of_week,
			backup_setting.retention_period_ts,
			backup_setting.hook_url
		FROM backup_setting `+
		strings.Join(join, " ")+
		` WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backupSettingList []*BackupSettingMessage
	for rows.Next() {
		var backupSetting BackupSettingMessage
		if err := rows.Scan(
			&backupSetting.ID,
			&backupSetting.UpdatedTs,
			&backupSetting.DatabaseUID,
			&backupSetting.Enabled,
			&backupSetting.HourOfDay,
			&backupSetting.DayOfWeek,
			&backupSetting.RetentionPeriodTs,
			&backupSetting.HookURL,
		); err != nil {
			return nil, err
		}
		backupSettingList = append(backupSettingList, &backupSetting)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return backupSettingList, nil
}
