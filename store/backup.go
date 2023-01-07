package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// backupRaw is the store model for an Backup.
// Fields have exactly the same meanings as Backup.
type backupRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name                    string
	Status                  api.BackupStatus
	Type                    api.BackupType
	StorageBackend          api.BackupStorageBackend
	MigrationHistoryVersion string
	Path                    string
	Comment                 string
	// Payload contains data such as PITR info, which will not be created at first.
	// When backup runner executes the real backup job, it will fill this field.
	Payload api.BackupPayload
}

// toBackup creates an instance of Backup based on the backupRaw.
// This is intended to be called when we need to compose an Backup relationship.
func (raw *backupRaw) toBackup() *api.Backup {
	return &api.Backup{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatedTs: raw.CreatedTs,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:                    raw.Name,
		Status:                  raw.Status,
		Type:                    raw.Type,
		StorageBackend:          raw.StorageBackend,
		MigrationHistoryVersion: raw.MigrationHistoryVersion,
		Path:                    raw.Path,
		Comment:                 raw.Comment,
		Payload:                 raw.Payload,
	}
}

// backupSettingRaw is the store model for an BackupSetting.
// Fields have exactly the same meanings as BackupSetting.
type backupSettingRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Enabled           bool
	Hour              int
	DayOfWeek         int
	RetentionPeriodTs int
	// HookURL is the callback url to be requested (using HTTP GET) after a successful backup.
	HookURL string
}

// toBackupSetting creates an instance of BackupSetting based on the backupSettingRaw.
// This is intended to be called when we need to compose an BackupSetting relationship.
func (raw *backupSettingRaw) toBackupSetting() *api.BackupSetting {
	return &api.BackupSetting{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Enabled:           raw.Enabled,
		Hour:              raw.Hour,
		DayOfWeek:         raw.DayOfWeek,
		RetentionPeriodTs: raw.RetentionPeriodTs,
		// HookURL is the callback url to be requested (using HTTP GET) after a successful backup.
		HookURL: raw.HookURL,
	}
}

// CreateBackup creates an instance of Backup.
func (s *Store) CreateBackup(ctx context.Context, create *api.BackupCreate) (*api.Backup, error) {
	backupRaw, err := s.createBackupRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Backup with BackupCreate[%+v]", create)
	}
	return composeBackup(backupRaw), nil
}

// GetBackupByID gets an instance of Backup by ID.
func (s *Store) GetBackupByID(ctx context.Context, id int) (*api.Backup, error) {
	backupRaw, err := s.getBackupRawByID(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get backup setting by ID %d", id)
	}
	if backupRaw == nil {
		return nil, nil
	}
	return composeBackup(backupRaw), nil
}

// FindBackup finds a list of Backup instances.
func (s *Store) FindBackup(ctx context.Context, find *api.BackupFind) ([]*api.Backup, error) {
	backupRawList, err := s.findBackupRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Backup list with BackupFind[%+v]", find)
	}
	var backupList []*api.Backup
	for _, raw := range backupRawList {
		backupList = append(backupList, composeBackup(raw))
	}
	return backupList, nil
}

// PatchBackup patches an instance of Backup.
func (s *Store) PatchBackup(ctx context.Context, patch *api.BackupPatch) (*api.Backup, error) {
	backupRaw, err := s.patchBackupRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Backup with BackupPatch[%+v]", patch)
	}
	return composeBackup(backupRaw), nil
}

// FindBackupSetting finds a list of BackupSetting of databases in the instance.
func (s *Store) FindBackupSetting(ctx context.Context, find api.BackupSettingFind) ([]*api.BackupSetting, error) {
	backupSettingRawList, err := s.findBackupSettingRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup setting list with BackupSettingFind %+v", find)
	}
	var backupSettingList []*api.BackupSetting
	for _, raw := range backupSettingRawList {
		backupSetting, err := s.composeBackupSetting(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose BackupSetting with backupSettingRaw %+v", raw)
		}
		backupSettingList = append(backupSettingList, backupSetting)
	}
	return backupSettingList, nil
}

// GetBackupSettingByDatabaseID gets an instance of BackupSetting by ID.
func (s *Store) GetBackupSettingByDatabaseID(ctx context.Context, id int) (*api.BackupSetting, error) {
	backupSettingRaw, err := s.getBackupSettingRaw(ctx, &api.BackupSettingFind{DatabaseID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get backup setting by ID %d", id)
	}
	if backupSettingRaw == nil {
		return nil, nil
	}
	backupSetting, err := s.composeBackupSetting(ctx, backupSettingRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose BackupSetting with backupSettingRaw[%+v]", backupSettingRaw)
	}
	return backupSetting, nil
}

// UpsertBackupSetting upserts an instance of backup setting.
func (s *Store) UpsertBackupSetting(ctx context.Context, upsert *api.BackupSettingUpsert) (*api.BackupSetting, error) {
	backupSettingRaw, err := s.upsertBackupSettingRaw(ctx, upsert)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upsert backup setting with BackupSettingUpsert[%+v]", upsert)
	}
	backup, err := s.composeBackupSetting(ctx, backupSettingRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Backup with backupRaw[%+v]", backupSettingRaw)
	}
	return backup, nil
}

// UpdateBackupSettingsInEnvironment upserts an instance of backup setting.
func (s *Store) UpdateBackupSettingsInEnvironment(ctx context.Context, upsert *api.BackupSettingUpsert) error {
	if err := s.validateBackupSettingUpsert(ctx, upsert); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	stmt := `
		UPDATE backup_setting
		SET enabled = $1
		WHERE id IN (
			SELECT backup_setting.id
			FROM backup_setting
			INNER JOIN db ON backup_setting.database_id = db.id
			INNER JOIN instance ON db.instance_id = instance.id
			INNER JOIN environment ON instance.environment_id = environment.id
			WHERE environment.id = $2
		);
	`
	if _, err := tx.ExecContext(ctx, stmt, upsert.Enabled, upsert.EnvironmentID); err != nil {
		return FormatError(err)
	}
	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}
	return nil
}

// FindBackupSettingsMatch finds a list of backup setting instances with match conditions.
func (s *Store) FindBackupSettingsMatch(ctx context.Context, match *api.BackupSettingsMatch) ([]*api.BackupSetting, error) {
	backupSettingRawList, err := s.findBackupSettingsMatchImpl(ctx, match)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find matching backup setting list with BackupSettingsMatch[%+v]", match)
	}
	var backupSettingList []*api.BackupSetting
	for _, raw := range backupSettingRawList {
		backupSetting, err := s.composeBackupSetting(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Backup with backupRaw[%+v]", raw)
		}
		backupSettingList = append(backupSettingList, backupSetting)
	}
	return backupSettingList, nil
}

//
// private functions
//

// composeBackup composes an instance of Backup by backupRaw.
func composeBackup(raw *backupRaw) *api.Backup {
	return raw.toBackup()
}

func (s *Store) composeBackupSetting(ctx context.Context, raw *backupSettingRaw) (*api.BackupSetting, error) {
	backupSetting := raw.toBackupSetting()

	creator, err := s.GetPrincipalByID(ctx, backupSetting.CreatorID)
	if err != nil {
		return nil, err
	}
	backupSetting.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, backupSetting.UpdaterID)
	if err != nil {
		return nil, err
	}
	backupSetting.Updater = updater

	database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: &backupSetting.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("failed to get database with databaseID %v", backupSetting.DatabaseID)
	}
	backupSetting.Database = database

	return backupSetting, nil
}

// createBackupRaw creates a new backup.
func (s *Store) createBackupRaw(ctx context.Context, create *api.BackupCreate) (*backupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backupRaw, err := s.createBackupImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

// getBackupRawByID retrieves a single backup based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getBackupRawByID(ctx context.Context, id int) (*backupRaw, error) {
	find := &api.BackupFind{ID: &id}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backupRawList, err := s.findBackupImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(backupRawList) == 0 {
		return nil, nil
	} else if len(backupRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d backups with filter %+v, expect 1. ", len(backupRawList), find)}
	}
	return backupRawList[0], nil
}

// findBackupRaw retrieves a list of backups based on find.
func (s *Store) findBackupRaw(ctx context.Context, find *api.BackupFind) ([]*backupRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backupRawList, err := s.findBackupImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return backupRawList, nil
}

// patchBackupRaw updates an existing backup by ID.
// Returns ENOTFOUND if backup does not exist.
func (s *Store) patchBackupRaw(ctx context.Context, patch *api.BackupPatch) (*backupRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backupRaw, err := s.patchBackupImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

// upsertBackupSettingRaw sets the backup settings for a database.
func (s *Store) upsertBackupSettingRaw(ctx context.Context, upsert *api.BackupSettingUpsert) (*backupSettingRaw, error) {
	if err := s.validateBackupSettingUpsert(ctx, upsert); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	backupRaw, err := s.upsertBackupSettingImpl(ctx, tx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return backupRaw, nil
}

func (s *Store) validateBackupSettingUpsert(ctx context.Context, upsert *api.BackupSettingUpsert) error {
	backupPlanPolicy, err := s.GetBackupPlanPolicyByEnvID(ctx, upsert.EnvironmentID)
	if err != nil {
		return err
	}
	// Backup plan policy check for backup setting mutation.
	if backupPlanPolicy.Schedule != api.BackupPlanPolicyScheduleUnset {
		if !upsert.Enabled {
			return &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting should not be disabled for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
		}
		switch backupPlanPolicy.Schedule {
		case api.BackupPlanPolicyScheduleDaily:
			if upsert.DayOfWeek != -1 {
				return &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting DayOfWeek should be unset for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		case api.BackupPlanPolicyScheduleWeekly:
			if upsert.DayOfWeek == -1 {
				return &common.Error{Code: common.Invalid, Err: errors.Errorf("backup setting DayOfWeek should be set for backup plan policy schedule %q", backupPlanPolicy.Schedule)}
			}
		}
	}
	return nil
}

// createBackupImpl creates a new backup.
func (*Store) createBackupImpl(ctx context.Context, tx *Tx, create *api.BackupCreate) (*backupRaw, error) {
	// Insert row into backup.
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
			path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, name, status, type, storage_backend, migration_history_version, path, comment
	`
	var backupRaw backupRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.DatabaseID,
		create.Name,
		api.BackupStatusPendingCreate,
		create.Type,
		create.StorageBackend,
		create.MigrationHistoryVersion,
		create.Path,
	).Scan(
		&backupRaw.ID,
		&backupRaw.RowStatus,
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &backupRaw, nil
}

func (*Store) findBackupImpl(ctx context.Context, tx *Tx, find *api.BackupFind) ([]*backupRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
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
			row_status,
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
			comment,
			payload
		FROM backup
		WHERE `+strings.Join(where, " AND ")+` ORDER BY updated_ts DESC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupRawList.
	var backupRawList []*backupRaw
	for rows.Next() {
		var backupRaw backupRaw
		var payload []byte
		if err := rows.Scan(
			&backupRaw.ID,
			&backupRaw.RowStatus,
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
			&payload,
		); err != nil {
			return nil, FormatError(err)
		}
		if err := json.Unmarshal(payload, &backupRaw.Payload); err != nil {
			return nil, err
		}
		backupRawList = append(backupRawList, &backupRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return backupRawList, nil
}

// patchBackupImpl updates a backup by ID. Returns the new state of the backup after update.
func (*Store) patchBackupImpl(ctx context.Context, tx *Tx, patch *api.BackupPatch) (*backupRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
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

	args = append(args, patch.ID)

	var backupRaw backupRaw
	var payload []byte
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE backup
			SET `+strings.Join(set, ", ")+`
			WHERE id = $%d
			RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, database_id, name, status, type, storage_backend, migration_history_version, path, comment, payload
		`, len(args)),
		args...,
	).Scan(
		&backupRaw.ID,
		&backupRaw.RowStatus,
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
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("backup ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	if err := json.Unmarshal(payload, &backupRaw.Payload); err != nil {
		return nil, err
	}
	return &backupRaw, nil
}

// getBackupSettingRaw finds the backup setting for a database.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getBackupSettingRaw(ctx context.Context, find *api.BackupSettingFind) (*backupSettingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findBackupSettingImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d backup settings with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

func (s *Store) findBackupSettingRaw(ctx context.Context, find api.BackupSettingFind) ([]*backupSettingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.InstanceID; v != nil {
		// Relation backup_setting do not have the column "instance_id", so we should join relation db to add the condition.
		where, args = append(where, fmt.Sprintf("db.instance_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			bs.id,
			bs.creator_id,
			bs.created_ts,
			bs.updater_id,
			bs.updated_ts,
			bs.database_id,
			bs.enabled,
			bs.hour,
			bs.day_of_week,
			bs.retention_period_ts,
			bs.hook_url
		FROM backup_setting AS bs
		JOIN db on db.id = bs.database_id
		WHERE `+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupSettingRawList.
	var backupSettingRawList []*backupSettingRaw
	for rows.Next() {
		var backupSettingRaw backupSettingRaw
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
			&backupSettingRaw.RetentionPeriodTs,
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

func (*Store) findBackupSettingImpl(ctx context.Context, tx *Tx, find *api.BackupSettingFind) ([]*backupSettingRaw, error) {
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
			retention_period_ts,
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
	var backupSettingRawList []*backupSettingRaw
	for rows.Next() {
		var backupSettingRaw backupSettingRaw
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
			&backupSettingRaw.RetentionPeriodTs,
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

// upsertBackupSettingImpl updates an existing backup setting.
func (*Store) upsertBackupSettingImpl(ctx context.Context, tx *Tx, upsert *api.BackupSettingUpsert) (*backupSettingRaw, error) {
	// Upsert row into backup_setting.
	query := `
		INSERT INTO backup_setting (
			creator_id,
			updater_id,
			database_id,
			enabled,
			hour,
			day_of_week,
			retention_period_ts,
			hook_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(database_id) DO UPDATE SET
				enabled = EXCLUDED.enabled,
				hour = EXCLUDED.hour,
				day_of_week = EXCLUDED.day_of_week,
				retention_period_ts = EXCLUDED.retention_period_ts,
				hook_url = EXCLUDED.hook_url
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, enabled, hour, day_of_week, retention_period_ts, hook_url
	`
	var backupSettingRaw backupSettingRaw
	if err := tx.QueryRowContext(ctx, query,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.DatabaseID,
		upsert.Enabled,
		upsert.Hour,
		upsert.DayOfWeek,
		upsert.RetentionPeriodTs,
		upsert.HookURL,
	).Scan(
		&backupSettingRaw.ID,
		&backupSettingRaw.CreatorID,
		&backupSettingRaw.CreatedTs,
		&backupSettingRaw.UpdaterID,
		&backupSettingRaw.UpdatedTs,
		&backupSettingRaw.DatabaseID,
		&backupSettingRaw.Enabled,
		&backupSettingRaw.Hour,
		&backupSettingRaw.DayOfWeek,
		&backupSettingRaw.RetentionPeriodTs,
		&backupSettingRaw.HookURL,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &backupSettingRaw, nil
}

// findBackupSettingsMatchImpl retrieves a list of backup settings based on match condition.
func (s *Store) findBackupSettingsMatchImpl(ctx context.Context, match *api.BackupSettingsMatch) ([]*backupSettingRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

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
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into backupSettingRawList.
	var backupSettingRawList []*backupSettingRaw
	for rows.Next() {
		var backupSettingRaw backupSettingRaw
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
			&backupSettingRaw.RetentionPeriodTs,
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
