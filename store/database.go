package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/metric"
)

// databaseRaw is the store model for an Database.
// Fields have exactly the same meanings as Database.
type databaseRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID      int
	InstanceID     int
	SourceBackupID int

	// Domain specific fields
	Name                 string
	CharacterSet         string
	Collation            string
	SchemaVersion        string
	SyncStatus           api.SyncStatus
	LastSuccessfulSyncTs int64
}

// toDatabase creates an instance of Database based on the databaseRaw.
// This is intended to be called when we need to compose an Database relationship.
func (raw *databaseRaw) toDatabase() *api.Database {
	return &api.Database{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID:      raw.ProjectID,
		InstanceID:     raw.InstanceID,
		SourceBackupID: raw.SourceBackupID,

		// Domain specific fields
		Name:                 raw.Name,
		CharacterSet:         raw.CharacterSet,
		Collation:            raw.Collation,
		SchemaVersion:        raw.SchemaVersion,
		SyncStatus:           raw.SyncStatus,
		LastSuccessfulSyncTs: raw.LastSuccessfulSyncTs,
	}
}

// CreateDatabase creates an instance of Database
func (s *Store) CreateDatabase(ctx context.Context, create *api.DatabaseCreate) (*api.Database, error) {
	databaseRaw, err := s.createDatabaseRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Database with DatabaseCreate[%+v], error[%w]", create, err)
	}
	database, err := s.composeDatabase(ctx, databaseRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Database with databaseRaw[%+v], error[%w]", databaseRaw, err)
	}
	return database, nil
}

// FindDatabase finds a list of Database instances
func (s *Store) FindDatabase(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	databaseRawList, err := s.findDatabaseRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Database list with DatabaseFind[%+v], error[%w]", find, err)
	}
	var databaseList []*api.Database
	for _, raw := range databaseRawList {
		database, err := s.composeDatabase(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Database with databaseRaw[%+v], error[%w]", raw, err)
		}
		databaseList = append(databaseList, database)
	}
	return databaseList, nil
}

// GetDatabase gets an instance of Database
func (s *Store) GetDatabase(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	databaseRaw, err := s.getDatabaseRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Database with DatabaseFind[%+v], error[%w]", find, err)
	}
	if databaseRaw == nil {
		return nil, nil
	}
	database, err := s.composeDatabase(ctx, databaseRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Database with databaseRaw[%+v], error[%w]", databaseRaw, err)
	}
	return database, nil
}

// PatchDatabase patches an instance of Database
func (s *Store) PatchDatabase(ctx context.Context, patch *api.DatabasePatch) (*api.Database, error) {
	databaseRaw, err := s.patchDatabaseRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch Database with DatabasePatch[%+v], error[%w]", patch, err)
	}
	database, err := s.composeDatabase(ctx, databaseRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Database with databaseRaw[%+v], error[%w]", databaseRaw, err)
	}
	return database, nil
}

// CountDatabaseGroupByBackupScheduleAndEnabled counts database, group by backup schedule and enabled
func (s *Store) CountDatabaseGroupByBackupScheduleAndEnabled(ctx context.Context) ([]*metric.DatabaseCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	rows, err := tx.PTx.QueryContext(ctx, `
		WITH database_backup_policy AS (
			SELECT db.id AS database_id, backup_policy.payload AS payload
			FROM db, instance LEFT JOIN (
				SELECT environment_id, payload
				FROM policy
				WHERE type = 'bb.policy.backup-plan'
			) AS backup_policy ON instance.environment_id = backup_policy.environment_id
			WHERE db.instance_id = instance.id
		), database_backup_setting AS(
			SELECT db.id AS database_id, backup_setting.enabled AS enabled
			FROM db LEFT JOIN backup_setting ON db.id = backup_setting.database_id
		)
		SELECT database_backup_policy.payload, database_backup_setting.enabled, COUNT(*)
		FROM database_backup_policy FULL JOIN database_backup_setting
			ON database_backup_policy.database_id = database_backup_setting.database_id
		GROUP BY database_backup_policy.payload, database_backup_setting.enabled
		`)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var databaseCountMetricList []*metric.DatabaseCountMetric
	for rows.Next() {
		var optionalPayload sql.NullString
		var optionalEnabled sql.NullBool
		var count int
		if err := rows.Scan(&optionalPayload, &optionalEnabled, &count); err != nil {
			return nil, FormatError(err)
		}
		var backupPlanPolicySchedule *api.BackupPlanPolicySchedule
		if optionalPayload.Valid {
			backupPlanPolicy, err := api.UnmarshalBackupPlanPolicy(optionalPayload.String)
			if err != nil {
				return nil, FormatError(err)
			}
			backupPlanPolicySchedule = &backupPlanPolicy.Schedule
		}
		var enabled *bool
		if optionalEnabled.Valid {
			enabled = &optionalEnabled.Bool
		}
		databaseCountMetricList = append(databaseCountMetricList, &metric.DatabaseCountMetric{
			BackupPlanPolicySchedule: backupPlanPolicySchedule,
			BackupSettingEnabled:     enabled,
			Count:                    count,
		})
	}

	return databaseCountMetricList, nil
}

//
// private functions
//

func (s *Store) composeDatabase(ctx context.Context, raw *databaseRaw) (*api.Database, error) {
	db := raw.toDatabase()

	creator, err := s.GetPrincipalByID(ctx, db.CreatorID)
	if err != nil {
		return nil, err
	}
	db.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, db.UpdaterID)
	if err != nil {
		return nil, err
	}
	db.Updater = updater

	project, err := s.GetProjectByID(ctx, db.ProjectID)
	if err != nil {
		return nil, err
	}
	db.Project = project

	instance, err := s.GetInstanceByID(ctx, db.InstanceID)
	if err != nil {
		return nil, err
	}
	db.Instance = instance

	if db.SourceBackupID != 0 {
		sourceBackup, err := s.GetBackupByID(ctx, db.SourceBackupID)
		if err != nil {
			return nil, err
		}
		db.SourceBackup = sourceBackup
	}

	// For now, only wildcard(*) database has data sources and we disallow it to be returned to the client.
	// So we set this value to an empty array until we need to develop a data source for a non-wildcard database.
	db.DataSourceList = []*api.DataSource{}

	rowStatus := api.Normal
	anomalyList, err := s.FindAnomaly(ctx, &api.AnomalyFind{
		RowStatus:  &rowStatus,
		DatabaseID: &db.ID,
	})
	if err != nil {
		return nil, err
	}
	db.AnomalyList = anomalyList

	rowStatus = api.Normal
	labelList, err := s.FindDatabaseLabel(ctx, &api.DatabaseLabelFind{
		DatabaseID: &db.ID,
		RowStatus:  &rowStatus,
	})
	if err != nil {
		return nil, err
	}

	// Since tenants are identified by labels in deployment config, we need an environment
	// label to identify tenants from different environment in a schema update deployment.
	// If we expose the environment label concept in the deployment config, it should look consistent in the label API.

	// Each database instance is created under a particular environment.
	// The value of bb.environment is identical to the name of the environment.

	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentKeyName,
		Value: db.Instance.Environment.Name,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return nil, err
	}
	db.Labels = string(labels)

	return db, nil
}

// createDatabaseRaw creates a new database.
func (s *Store) createDatabaseRaw(ctx context.Context, create *api.DatabaseCreate) (*databaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	database, err := s.createDatabaseRawTx(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return database, nil
}

// createDatabaseRawTx creates a database with a transaction.
func (s *Store) createDatabaseRawTx(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*databaseRaw, error) {
	backupPlanPolicy, err := s.GetBackupPlanPolicyByEnvID(ctx, create.EnvironmentID)
	if err != nil {
		return nil, err
	}

	databaseRaw, err := s.createDatabaseImpl(ctx, tx, create)
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
		if _, err := s.upsertBackupSettingImpl(ctx, tx, backupSettingUpsert); err != nil {
			return nil, err
		}
	}

	if err := s.cache.UpsertCache(api.DatabaseCache, databaseRaw.ID, databaseRaw); err != nil {
		return nil, err
	}

	return databaseRaw, nil
}

// findDatabaseRaw retrieves a list of databases based on find.
func (s *Store) findDatabaseRaw(ctx context.Context, find *api.DatabaseFind) ([]*databaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDatabaseImpl(ctx, tx.PTx, find)
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

// getDatabaseRaw retrieves a single database based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getDatabaseRaw(ctx context.Context, find *api.DatabaseFind) (*databaseRaw, error) {
	if find.ID != nil {
		databaseRaw := &databaseRaw{}
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

	list, err := s.findDatabaseImpl(ctx, tx.PTx, find)
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

// patchDatabaseRaw updates an existing database by ID.
// Returns ENOTFOUND if database does not exist.
func (s *Store) patchDatabaseRaw(ctx context.Context, patch *api.DatabasePatch) (*databaseRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	database, err := s.patchDatabaseImpl(ctx, tx.PTx, patch)
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

// createDatabaseImpl creates a new database.
func (s *Store) createDatabaseImpl(ctx context.Context, tx *sql.Tx, create *api.DatabaseCreate) (*databaseRaw, error) {
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
	var databaseRaw databaseRaw
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

func (s *Store) findDatabaseImpl(ctx context.Context, tx *sql.Tx, find *api.DatabaseFind) ([]*databaseRaw, error) {
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
	var databaseRawList []*databaseRaw
	for rows.Next() {
		var databaseRaw databaseRaw
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

// patchDatabaseImpl updates a database by ID. Returns the new state of the database after update.
func (s *Store) patchDatabaseImpl(ctx context.Context, tx *sql.Tx, patch *api.DatabasePatch) (*databaseRaw, error) {
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
		var databaseRaw databaseRaw
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
