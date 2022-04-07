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

	cache api.CacheService
	store *Store
}

// NewDatabaseService returns a new instance of DatabaseService.
func NewDatabaseService(logger *zap.Logger, db *DB, cache api.CacheService, store *Store) *DatabaseService {
	return &DatabaseService{
		l:     logger,
		db:    db,
		cache: cache,
		store: store,
	}
}

// CreateDatabase creates a new database.
func (s *DatabaseService) CreateDatabase(ctx context.Context, create *api.DatabaseCreate) (*api.DatabaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	database, err := s.CreateDatabaseTx(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return database, nil
}

// CreateDatabaseTx creates a database with a transaction.
func (s *DatabaseService) CreateDatabaseTx(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*api.DatabaseRaw, error) {
	backupPlanPolicy, err := s.store.GetBackupPlanPolicyByEnvID(ctx, create.EnvironmentID)
	if err != nil {
		return nil, err
	}

	databaseRaw, err := s.createDatabase(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	// Enable automatic backup setting based on backup plan policy.
	if backupPlanPolicy.Schedule != api.BackupPlanPolicyScheduleUnset {
		backupSettingUpsert := &api.BackupSettingUpsert{
			UpdaterID:  api.SystemBotID,
			DatabaseID: databaseRaw.ID,
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
		if _, err := s.store.upsertBackupSettingImpl(ctx, tx, backupSettingUpsert); err != nil {
			return nil, err
		}
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, databaseRaw.ID, databaseRaw); err != nil {
		return nil, err
	}

	return databaseRaw, nil
}

// FindDatabaseList retrieves a list of databases based on find.
func (s *DatabaseService) FindDatabaseList(ctx context.Context, find *api.DatabaseFind) ([]*api.DatabaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDatabaseList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
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
func (s *DatabaseService) FindDatabase(ctx context.Context, find *api.DatabaseFind) (*api.DatabaseRaw, error) {
	if find.ID != nil {
		databaseRaw := &api.DatabaseRaw{}
		has, err := s.cache.FindCache(api.DatabaseCache, *find.ID, databaseRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return databaseRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDatabaseList(ctx, tx.PTx, find)
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
func (s *DatabaseService) PatchDatabase(ctx context.Context, patch *api.DatabasePatch) (*api.DatabaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	database, err := s.patchDatabase(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, database.ID, database); err != nil {
		return nil, err
	}

	return database, nil
}

// createDatabase creates a new database.
func (s *DatabaseService) createDatabase(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*api.DatabaseRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO db (
			creator_id,
			updater_id,
			instance_id,
			project_id,
			name,
			character_set,
			"collation",
			sync_status,
			last_successful_sync_ts,
			schema_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'OK', EXTRACT(epoch from NOW()), $8)
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
			"collation",
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
	var databaseRaw api.DatabaseRaw
	if err := row.Scan(
		&databaseRaw.ID,
		&databaseRaw.CreatorID,
		&databaseRaw.CreatedTs,
		&databaseRaw.UpdaterID,
		&databaseRaw.UpdatedTs,
		&databaseRaw.InstanceID,
		&databaseRaw.ProjectID,
		&databaseRaw.Name,
		&databaseRaw.CharacterSet,
		&databaseRaw.Collation,
		&databaseRaw.SyncStatus,
		&databaseRaw.LastSuccessfulSyncTs,
		&databaseRaw.SchemaVersion,
	); err != nil {
		return nil, FormatError(err)
	}

	return &databaseRaw, nil
}

func (s *DatabaseService) findDatabaseList(ctx context.Context, tx *sql.Tx, find *api.DatabaseFind) ([]*api.DatabaseRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
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
			"collation",
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

	// Iterate over result set and deserialize rows into databaseRawList.
	var databaseRawList []*api.DatabaseRaw
	for rows.Next() {
		var databaseRaw api.DatabaseRaw
		var nullSourceBackupID sql.NullInt64
		if err := rows.Scan(
			&databaseRaw.ID,
			&databaseRaw.CreatorID,
			&databaseRaw.CreatedTs,
			&databaseRaw.UpdaterID,
			&databaseRaw.UpdatedTs,
			&databaseRaw.InstanceID,
			&databaseRaw.ProjectID,
			&nullSourceBackupID,
			&databaseRaw.Name,
			&databaseRaw.CharacterSet,
			&databaseRaw.Collation,
			&databaseRaw.SyncStatus,
			&databaseRaw.LastSuccessfulSyncTs,
			&databaseRaw.SchemaVersion,
		); err != nil {
			return nil, FormatError(err)
		}
		if nullSourceBackupID.Valid {
			databaseRaw.SourceBackupID = int(nullSourceBackupID.Int64)
		}

		databaseRawList = append(databaseRawList, &databaseRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return databaseRawList, nil
}

// patchDatabase updates a database by ID. Returns the new state of the database after update.
func (s *DatabaseService) patchDatabase(ctx context.Context, tx *sql.Tx, patch *api.DatabasePatch) (*api.DatabaseRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.ProjectID; v != nil {
		set, args = append(set, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SourceBackupID; v != nil {
		set, args = append(set, fmt.Sprintf("source_backup_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaVersion; v != nil {
		set, args = append(set, fmt.Sprintf("schema_version = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SyncStatus; v != nil {
		set, args = append(set, fmt.Sprintf("sync_status = $%d", len(args)+1)), append(args, api.SyncStatus(*v))
	}
	if v := patch.LastSuccessfulSyncTs; v != nil {
		set, args = append(set, fmt.Sprintf("last_successful_sync_ts = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
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
			"collation",
			sync_status,
			last_successful_sync_ts,
			schema_version
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var databaseRaw api.DatabaseRaw
		var nullSourceBackupID sql.NullInt64
		if err := row.Scan(
			&databaseRaw.ID,
			&databaseRaw.CreatorID,
			&databaseRaw.CreatedTs,
			&databaseRaw.UpdaterID,
			&databaseRaw.UpdatedTs,
			&databaseRaw.InstanceID,
			&databaseRaw.ProjectID,
			&nullSourceBackupID,
			&databaseRaw.Name,
			&databaseRaw.CharacterSet,
			&databaseRaw.Collation,
			&databaseRaw.SyncStatus,
			&databaseRaw.LastSuccessfulSyncTs,
			&databaseRaw.SchemaVersion,
		); err != nil {
			return nil, FormatError(err)
		}
		if nullSourceBackupID.Valid {
			databaseRaw.SourceBackupID = int(nullSourceBackupID.Int64)
		}
		return &databaseRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("database ID not found: %d", patch.ID)}
}
