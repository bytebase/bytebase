package store

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.DatabaseService = (*DatabaseService)(nil)
)

// DatabaseService represents a service for managing database.
type DatabaseService struct {
	l  *zap.Logger
	db *DB

	cache         api.CacheService
	policyService api.PolicyService
	backupService api.BackupService
}

// NewDatabaseService returns a new instance of DatabaseService.
func NewDatabaseService(logger *zap.Logger, db *DB, cache api.CacheService, policyService api.PolicyService, backupService api.BackupService) *DatabaseService {
	return &DatabaseService{
		l:             logger,
		db:            db,
		cache:         cache,
		policyService: policyService,
		backupService: backupService,
	}
}

// CreateDatabase creates a new database.
func (s *DatabaseService) CreateDatabase(ctx context.Context, create *api.DatabaseCreate) (*api.Database, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	database, err := s.CreateDatabaseTx(ctx, tx.Tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return database, nil
}

// CreateDatabaseTx creates a database with a transaction.
func (s *DatabaseService) CreateDatabaseTx(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*api.Database, error) {
	backupPlanPolicy, err := s.policyService.GetBackupPlanPolicy(ctx, create.EnvironmentID)
	if err != nil {
		return nil, err
	}

	database, err := s.createDatabase(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	// Enable automatic backup setting based on backup plan policy.
	if backupPlanPolicy.Schedule != api.BackupPlanPolicyScheduleUnset {
		backupSettingUpsert := &api.BackupSettingUpsert{
			UpdaterID:  api.SystemBotID,
			DatabaseID: database.ID,
			Enabled:    true,
			Hour:       rand.Intn(24),
			HookURL:    "",
		}
		switch backupPlanPolicy.Schedule {
		case api.BackupPlanPolicyScheduleDaily:
			backupSettingUpsert.DayOfWeek = -1
		case api.BackupPlanPolicyScheduleWeekly:
			backupSettingUpsert.DayOfWeek = rand.Intn(7)
		}
		if _, err := s.backupService.UpsertBackupSettingTx(ctx, tx, backupSettingUpsert); err != nil {
			return nil, err
		}
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, database.ID, database); err != nil {
		return nil, err
	}

	return database, nil
}

// FindDatabaseList retrieves a list of databases based on find.
func (s *DatabaseService) FindDatabaseList(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDatabaseList(ctx, tx, find)
	if err != nil {
		return []*api.Database{}, err
	}

	if err == nil {
		for _, database := range list {
			if err := s.cache.UpsertCache(api.DatabaseCache, database.ID, database); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// FindDatabase retrieves a single database based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *DatabaseService) FindDatabase(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	if find.ID != nil {
		database := &api.Database{}
		has, err := s.cache.FindCache(api.DatabaseCache, *find.ID, database)
		if err != nil {
			return nil, err
		}
		if has {
			return database, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDatabaseList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d databases with filter %+v, expect 1. ", len(list), find)}
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, list[0].ID, list[0]); err != nil {
		return nil, err
	}

	return list[0], nil
}

// PatchDatabase updates an existing database by ID.
// Returns ENOTFOUND if database does not exist.
func (s *DatabaseService) PatchDatabase(ctx context.Context, patch *api.DatabasePatch) (*api.Database, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	database, err := s.patchDatabase(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, database.ID, database); err != nil {
		return nil, err
	}

	return database, nil
}

// createDatabase creates a new database.
func (s *DatabaseService) createDatabase(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*api.Database, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO db (
			creator_id,
			updater_id,
			instance_id,
			project_id,
			name,
			character_set,
			collation,
			sync_status,
			last_successful_sync_ts,
			schema_version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'OK', (strftime('%s', 'now')), ?)
		RETURNING
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			instance_id,
			project_id,
			name,
			character_set,
			collation,
			sync_status,
			last_successful_sync_ts,
			schema_version
	`,
		create.CreatorID,
		create.CreatorID,
		create.InstanceID,
		create.ProjectID,
		create.Name,
		create.CharacterSet,
		create.Collation,
		create.SchemaVersion,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var database api.Database
	if err := row.Scan(
		&database.ID,
		&database.CreatorID,
		&database.CreatedTs,
		&database.UpdaterID,
		&database.UpdatedTs,
		&database.InstanceID,
		&database.ProjectID,
		&database.Name,
		&database.CharacterSet,
		&database.Collation,
		&database.SyncStatus,
		&database.LastSuccessfulSyncTs,
		&database.SchemaVersion,
	); err != nil {
		return nil, FormatError(err)
	}

	return &database, nil
}

func (s *DatabaseService) findDatabaseList(ctx context.Context, tx *Tx, find *api.DatabaseFind) (_ []*api.Database, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, "instance_id = ?"), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, "project_id = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}
	if !find.IncludeAllDatabase {
		where = append(where, "name != '"+api.AllDatabaseName+"'")
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			instance_id,
			project_id,
			source_backup_id,
			name,
			character_set,
			collation,
			sync_status,
			last_successful_sync_ts,
			schema_version
		FROM db
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Database, 0)
	for rows.Next() {
		var database api.Database
		var nullSourceBackupID sql.NullInt64
		if err := rows.Scan(
			&database.ID,
			&database.CreatorID,
			&database.CreatedTs,
			&database.UpdaterID,
			&database.UpdatedTs,
			&database.InstanceID,
			&database.ProjectID,
			&nullSourceBackupID,
			&database.Name,
			&database.CharacterSet,
			&database.Collation,
			&database.SyncStatus,
			&database.LastSuccessfulSyncTs,
			&database.SchemaVersion,
		); err != nil {
			return nil, FormatError(err)
		}
		if nullSourceBackupID.Valid {
			database.SourceBackupID = int(nullSourceBackupID.Int64)
		}

		list = append(list, &database)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchDatabase updates a database by ID. Returns the new state of the database after update.
func (s *DatabaseService) patchDatabase(ctx context.Context, tx *Tx, patch *api.DatabasePatch) (*api.Database, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.ProjectID; v != nil {
		set, args = append(set, "project_id = ?"), append(args, *v)
	}
	if v := patch.SourceBackupID; v != nil {
		set, args = append(set, "source_backup_id = ?"), append(args, *v)
	}
	if v := patch.SchemaVersion; v != nil {
		set, args = append(set, "schema_version = ?"), append(args, *v)
	}
	if v := patch.SyncStatus; v != nil {
		set, args = append(set, "sync_status = ?"), append(args, api.SyncStatus(*v))
	}
	if v := patch.LastSuccessfulSyncTs; v != nil {
		set, args = append(set, "last_successful_sync_ts = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			instance_id,
			project_id,
			source_backup_id,
			name,
			character_set,
			collation,
			sync_status,
			last_successful_sync_ts,
			schema_version
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var database api.Database
		var nullSourceBackupID sql.NullInt64
		if err := row.Scan(
			&database.ID,
			&database.CreatorID,
			&database.CreatedTs,
			&database.UpdaterID,
			&database.UpdatedTs,
			&database.InstanceID,
			&database.ProjectID,
			&nullSourceBackupID,
			&database.Name,
			&database.CharacterSet,
			&database.Collation,
			&database.SyncStatus,
			&database.LastSuccessfulSyncTs,
			&database.SchemaVersion,
		); err != nil {
			return nil, FormatError(err)
		}
		if nullSourceBackupID.Valid {
			database.SourceBackupID = int(nullSourceBackupID.Int64)
		}
		return &database, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("database ID not found: %d", patch.ID)}
}
