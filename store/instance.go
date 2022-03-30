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
	_ api.InstanceService = (*InstanceService)(nil)
)

// InstanceService represents a service for managing instance.
type InstanceService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService

	databaseService api.DatabaseService
	store           *Store
}

// NewInstanceService returns a new instance of InstanceService.
func NewInstanceService(logger *zap.Logger, db *DB, cache api.CacheService, databaseService api.DatabaseService, store *Store) *InstanceService {
	return &InstanceService{l: logger, db: db, cache: cache, databaseService: databaseService, store: store}
}

// CreateInstance creates a new instance.
func (s *InstanceService) CreateInstance(ctx context.Context, create *api.InstanceCreate) (*api.InstanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	instance, err := createInstance(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	// Create * database
	databaseCreate := &api.DatabaseCreate{
		CreatorID:     create.CreatorID,
		ProjectID:     api.DefaultProjectID,
		InstanceID:    instance.ID,
		EnvironmentID: instance.EnvironmentID,
		Name:          api.AllDatabaseName,
		CharacterSet:  api.DefaultCharactorSetName,
		Collation:     api.DefaultCollationName,
	}
	allDatabase, err := s.databaseService.CreateDatabaseTx(ctx, tx.PTx, databaseCreate)
	if err != nil {
		return nil, err
	}

	// Create admin data source
	adminDataSourceCreate := &api.DataSourceCreate{
		CreatorID:  create.CreatorID,
		InstanceID: instance.ID,
		DatabaseID: allDatabase.ID,
		Name:       api.AdminDataSourceName,
		Type:       api.Admin,
		Username:   create.Username,
		Password:   create.Password,
	}
	if err := s.store.createDataSourceRawTx(ctx, tx.PTx, adminDataSourceCreate); err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.InstanceCache, instance.ID, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// FindInstanceList retrieves a list of instances based on find.
func (s *InstanceService) FindInstanceList(ctx context.Context, find *api.InstanceFind) ([]*api.InstanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findInstanceList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// CountInstance counts the number of instances.
func (s *InstanceService) CountInstance(ctx context.Context, find *api.InstanceFind) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, FormatError(err)
	}
	defer tx.PTx.Rollback()

	where, args := findInstanceQuery(find)

	row, err := tx.PTx.QueryContext(ctx, `
		SELECT COUNT(*)
		FROM instance
		WHERE `+where,
		args...,
	)
	if err != nil {
		return 0, FormatError(err)
	}
	defer row.Close()

	count := 0
	if row.Next() {
		if err := row.Scan(&count); err != nil {
			return 0, FormatError(err)
		}
	}

	return count, nil
}

// FindInstance retrieves a single instance based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *InstanceService) FindInstance(ctx context.Context, find *api.InstanceFind) (*api.InstanceRaw, error) {
	if find.ID != nil {
		instanceRaw := &api.InstanceRaw{}
		has, err := s.cache.FindCache(api.InstanceCache, *find.ID, instanceRaw)
		if err != nil {
			return nil, err
		}
		if has {
			return instanceRaw, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findInstanceList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d instances with filter %+v, expect 1", len(list), find)}
	}

	instance := list[0]
	if err := s.cache.UpsertCache(api.InstanceCache, instance.ID, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// PatchInstance updates an existing instance by ID.
// Returns ENOTFOUND if instance does not exist.
func (s *InstanceService) PatchInstance(ctx context.Context, patch *api.InstancePatch) (*api.InstanceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	instance, err := patchInstance(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.InstanceCache, instance.ID, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// createInstance creates a new instance.
func createInstance(ctx context.Context, tx *sql.Tx, create *api.InstanceCreate) (*api.InstanceRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO instance (
			creator_id,
			updater_id,
			environment_id,
			name,
			engine,
			external_link,
			host,
			port
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port
	`,
		create.CreatorID,
		create.CreatorID,
		create.EnvironmentID,
		create.Name,
		create.Engine,
		create.ExternalLink,
		create.Host,
		create.Port,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var instanceRaw api.InstanceRaw
	if err := row.Scan(
		&instanceRaw.ID,
		&instanceRaw.RowStatus,
		&instanceRaw.CreatorID,
		&instanceRaw.CreatedTs,
		&instanceRaw.UpdaterID,
		&instanceRaw.UpdatedTs,
		&instanceRaw.EnvironmentID,
		&instanceRaw.Name,
		&instanceRaw.Engine,
		&instanceRaw.EngineVersion,
		&instanceRaw.ExternalLink,
		&instanceRaw.Host,
		&instanceRaw.Port,
	); err != nil {
		return nil, FormatError(err)
	}

	return &instanceRaw, nil
}

func findInstanceList(ctx context.Context, tx *sql.Tx, find *api.InstanceFind) ([]*api.InstanceRaw, error) {
	where, args := findInstanceQuery(find)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			environment_id,
			name,
			engine,
			engine_version,
			external_link,
			host,
			port
		FROM instance
		WHERE `+where,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into instanceRawList.
	var instanceRawList []*api.InstanceRaw
	for rows.Next() {
		var instanceRaw api.InstanceRaw
		if err := rows.Scan(
			&instanceRaw.ID,
			&instanceRaw.RowStatus,
			&instanceRaw.CreatorID,
			&instanceRaw.CreatedTs,
			&instanceRaw.UpdaterID,
			&instanceRaw.UpdatedTs,
			&instanceRaw.EnvironmentID,
			&instanceRaw.Name,
			&instanceRaw.Engine,
			&instanceRaw.EngineVersion,
			&instanceRaw.ExternalLink,
			&instanceRaw.Host,
			&instanceRaw.Port,
		); err != nil {
			return nil, FormatError(err)
		}

		instanceRawList = append(instanceRawList, &instanceRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return instanceRawList, nil
}

// patchInstance updates a instance by ID. Returns the new state of the instance after update.
func patchInstance(ctx context.Context, tx *sql.Tx, patch *api.InstancePatch) (*api.InstanceRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EngineVersion; v != nil {
		set, args = append(set, fmt.Sprintf("engine_version = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, fmt.Sprintf("external_link = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Host; v != nil {
		set, args = append(set, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE instance
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, environment_id, name, engine, engine_version, external_link, host, port
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var instanceRaw api.InstanceRaw
		if err := row.Scan(
			&instanceRaw.ID,
			&instanceRaw.RowStatus,
			&instanceRaw.CreatorID,
			&instanceRaw.CreatedTs,
			&instanceRaw.UpdaterID,
			&instanceRaw.UpdatedTs,
			&instanceRaw.EnvironmentID,
			&instanceRaw.Name,
			&instanceRaw.Engine,
			&instanceRaw.EngineVersion,
			&instanceRaw.ExternalLink,
			&instanceRaw.Host,
			&instanceRaw.Port,
		); err != nil {
			return nil, FormatError(err)
		}

		return &instanceRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("instance ID not found: %d", patch.ID)}
}

func findInstanceQuery(find *api.InstanceFind) (string, []interface{}) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}

	return strings.Join(where, " AND "), args
}
