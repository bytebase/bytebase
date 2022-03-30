package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.BackupService = (*BackupService)(nil)
)

// BackupService represents a service for managing backup.
type BackupService struct {
	l     *zap.Logger
	db    *DB
	store *Store
}

// NewBackupService returns a new instance of BackupService.
func NewBackupService(logger *zap.Logger, db *DB, store *Store) *BackupService {
	return &BackupService{l: logger, db: db, store: store}
}

// CreateBackup creates a new backup.
func (s *BackupService) CreateBackup(ctx context.Context, create *api.BackupCreate) (*api.BackupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	backupRaw, err := s.createBackup(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

// FindBackup retrieves a single backup based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *BackupService) FindBackup(ctx context.Context, find *api.BackupFind) (*api.BackupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	backupRawList, err := s.findBackupList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(backupRawList) == 0 {
		return nil, nil
	} else if len(backupRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d backups with filter %+v, expect 1. ", len(backupRawList), find)}
	}
	return backupRawList[0], nil
}

// FindBackupList retrieves a list of backups based on find.
func (s *BackupService) FindBackupList(ctx context.Context, find *api.BackupFind) ([]*api.BackupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	backupRawList, err := s.findBackupList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return backupRawList, nil
}

// PatchBackup updates an existing backup by ID.
// Returns ENOTFOUND if backup does not exist.
func (s *BackupService) PatchBackup(ctx context.Context, patch *api.BackupPatch) (*api.BackupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	backupRaw, err := s.patchBackup(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

// createBackup creates a new backup.
func (s *BackupService) createBackup(ctx context.Context, tx *sql.Tx, create *api.BackupCreate) (*api.BackupRaw, error) {
	// Insert row into backup.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO backup (
			creator_id,
			updater_id,
			database_id,
			name,
			status,
			type,
			storage_backend,
			migration_history_version,
			path
		)
		VALUES ($1, $2, $3, $4, 'PENDING_CREATE', $5, $6, $7, $8)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, status, type, storage_backend, migration_history_version, path, comment
	`,
		create.CreatorID,
		create.CreatorID,
		create.DatabaseID,
		create.Name,
		create.Type,
		create.StorageBackend,
		create.MigrationHistoryVersion,
		create.Path,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var backupRaw api.BackupRaw
	if err := row.Scan(
		&backupRaw.ID,
		&backupRaw.CreatorID,
		&backupRaw.CreatedTs,
		&backupRaw.UpdaterID,
		&backupRaw.UpdatedTs,
		&backupRaw.DatabaseID,
		&backupRaw.Name,
		&backupRaw.Status,
		&backupRaw.Type,
		&backupRaw.StorageBackend,
		&backupRaw.MigrationHistoryVersion,
		&backupRaw.Path,
		&backupRaw.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &backupRaw, nil
}

func (s *BackupService) findBackupList(ctx context.Context, tx *sql.Tx, find *api.BackupFind) ([]*api.BackupRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Status; v != nil {
		where, args = append(where, fmt.Sprintf("status = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			name,
			status,
			type,
			storage_backend,
			migration_history_version,
			path,
			comment
		FROM backup
		WHERE `+strings.Join(where, " AND ")+` ORDER BY updated_ts DESC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupRawList.
	var backupRawList []*api.BackupRaw
	for rows.Next() {
		var backupRaw api.BackupRaw
		if err := rows.Scan(
			&backupRaw.ID,
			&backupRaw.CreatorID,
			&backupRaw.CreatedTs,
			&backupRaw.UpdaterID,
			&backupRaw.UpdatedTs,
			&backupRaw.DatabaseID,
			&backupRaw.Name,
			&backupRaw.Status,
			&backupRaw.Type,
			&backupRaw.StorageBackend,
			&backupRaw.MigrationHistoryVersion,
			&backupRaw.Path,
			&backupRaw.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		backupRawList = append(backupRawList, &backupRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return backupRawList, nil
}

// patchBackup updates a backup by ID. Returns the new state of the backup after update.
func (s *BackupService) patchBackup(ctx context.Context, tx *sql.Tx, patch *api.BackupPatch) (*api.BackupRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "status = $2"), append(args, patch.Status)
	set, args = append(set, "comment = $3"), append(args, patch.Comment)

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE backup
		SET `+strings.Join(set, ", ")+`
		WHERE id = $4
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, status, type, storage_backend, migration_history_version, path, comment
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var backupRaw api.BackupRaw
		if err := row.Scan(
			&backupRaw.ID,
			&backupRaw.CreatorID,
			&backupRaw.CreatedTs,
			&backupRaw.UpdaterID,
			&backupRaw.UpdatedTs,
			&backupRaw.DatabaseID,
			&backupRaw.Name,
			&backupRaw.Status,
			&backupRaw.Type,
			&backupRaw.StorageBackend,
			&backupRaw.MigrationHistoryVersion,
			&backupRaw.Path,
			&backupRaw.Comment,
		); err != nil {
			return nil, FormatError(err)
		}
		return &backupRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("backup ID not found: %d", patch.ID)}
}

// FindBackupSetting finds the backup setting for a database.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *BackupService) FindBackupSetting(ctx context.Context, find *api.BackupSettingFind) (*api.BackupSettingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findBackupSetting(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d backup settings with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

func (s *BackupService) findBackupSetting(ctx context.Context, tx *sql.Tx, find *api.BackupSettingFind) ([]*api.BackupSettingRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			enabled,
			hour,
			day_of_week,
			hook_url
		FROM backup_setting
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupSettingRawList.
	var backupSettingRawList []*api.BackupSettingRaw
	for rows.Next() {
		var backupSettingRaw api.BackupSettingRaw
		if err := rows.Scan(
			&backupSettingRaw.ID,
			&backupSettingRaw.CreatorID,
			&backupSettingRaw.CreatedTs,
			&backupSettingRaw.UpdaterID,
			&backupSettingRaw.UpdatedTs,
			&backupSettingRaw.DatabaseID,
			&backupSettingRaw.Enabled,
			&backupSettingRaw.Hour,
			&backupSettingRaw.DayOfWeek,
			&backupSettingRaw.HookURL,
		); err != nil {
			return nil, FormatError(err)
		}

		backupSettingRawList = append(backupSettingRawList, &backupSettingRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return backupSettingRawList, nil
}

// UpsertBackupSetting sets the backup settings for a database.
func (s *BackupService) UpsertBackupSetting(ctx context.Context, upsert *api.BackupSettingUpsert) (*api.BackupSettingRaw, error) {
	backupPlanPolicy, err := s.store.GetBackupPlanPolicyByEnvID(ctx, upsert.EnvironmentID)
	if err != nil {
		return nil, err
	}
	// Backup plan policy check for backup setting mutation.
	if backupPlanPolicy.Schedule != api.BackupPlanPolicyScheduleUnset {
		if !upsert.Enabled {
			return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("backup setting should not be disabled for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
		}
		switch backupPlanPolicy.Schedule {
		case api.BackupPlanPolicyScheduleDaily:
			if upsert.DayOfWeek != -1 {
				return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("backup setting DayOfWeek should be unset for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		case api.BackupPlanPolicyScheduleWeekly:
			if upsert.DayOfWeek == -1 {
				return nil, &common.Error{Code: common.Invalid, Err: fmt.Errorf("backup setting DayOfWeek should be set for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	backupRaw, err := s.UpsertBackupSettingTx(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

// UpsertBackupSettingTx updates an existing backup setting.
func (s *BackupService) UpsertBackupSettingTx(ctx context.Context, tx *sql.Tx, upsert *api.BackupSettingUpsert) (*api.BackupSettingRaw, error) {
	// Upsert row into backup_setting.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO backup_setting (
			creator_id,
			updater_id,
			database_id,
			enabled,
			hour,
			day_of_week,
			hook_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(database_id) DO UPDATE SET
				enabled = EXCLUDED.enabled,
				hour = EXCLUDED.hour,
				day_of_week = EXCLUDED.day_of_week,
				hook_url = EXCLUDED.hook_url
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, enabled, hour, day_of_week, hook_url
		`,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.DatabaseID,
		upsert.Enabled,
		upsert.Hour,
		upsert.DayOfWeek,
		upsert.HookURL,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var backupSettingRaw api.BackupSettingRaw
	if err := row.Scan(
		&backupSettingRaw.ID,
		&backupSettingRaw.CreatorID,
		&backupSettingRaw.CreatedTs,
		&backupSettingRaw.UpdaterID,
		&backupSettingRaw.UpdatedTs,
		&backupSettingRaw.DatabaseID,
		&backupSettingRaw.Enabled,
		&backupSettingRaw.Hour,
		&backupSettingRaw.DayOfWeek,
		&backupSettingRaw.HookURL,
	); err != nil {
		return nil, FormatError(err)
	}

	return &backupSettingRaw, nil
}

// FindBackupSettingsMatch retrieves a list of backup settings based on match condition.
func (s *BackupService) FindBackupSettingsMatch(ctx context.Context, match *api.BackupSettingsMatch) ([]*api.BackupSettingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	rows, err := tx.PTx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			enabled,
			hour,
			day_of_week,
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
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupSettingRawList.
	var backupSettingRawList []*api.BackupSettingRaw
	for rows.Next() {
		var backupSettingRaw api.BackupSettingRaw
		if err := rows.Scan(
			&backupSettingRaw.ID,
			&backupSettingRaw.CreatorID,
			&backupSettingRaw.CreatedTs,
			&backupSettingRaw.UpdaterID,
			&backupSettingRaw.UpdatedTs,
			&backupSettingRaw.DatabaseID,
			&backupSettingRaw.Enabled,
			&backupSettingRaw.Hour,
			&backupSettingRaw.DayOfWeek,
			&backupSettingRaw.HookURL,
		); err != nil {
			return nil, FormatError(err)
		}

		backupSettingRawList = append(backupSettingRawList, &backupSettingRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return backupSettingRawList, nil
}
