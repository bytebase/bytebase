package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.DatabaseService = (*DatabaseService)(nil)
)

// DatabaseService represents a service for managing database.
type DatabaseService struct {
	l  *bytebase.Logger
	db *DB
}

// NewDatabaseService returns a new instance of DatabaseService.
func NewDatabaseService(logger *bytebase.Logger, db *DB) *DatabaseService {
	return &DatabaseService{l: logger, db: db}
}

// CreateDatabase creates a new database.
func (s *DatabaseService) CreateDatabase(ctx context.Context, create *api.DatabaseCreate) (*api.Database, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	database, err := s.createDatabase(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
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

	return list, nil
}

// FindDatabase retrieves a single database based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *DatabaseService) FindDatabase(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDatabaseList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("database not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found 5 databases with filter %v, expect 1. ", len(list), find)
	}
	return list[0], nil
}

// PatchDatabaseByID updates an existing database by ID.
// Returns ENOTFOUND if database does not exist.
func (s *DatabaseService) PatchDatabaseByID(ctx context.Context, patch *api.DatabasePatch) (*api.Database, error) {
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

	return database, nil
}

// createDatabase creates a new database.
func (s *DatabaseService) createDatabase(ctx context.Context, tx *Tx, create *api.DatabaseCreate) (*api.Database, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO db (
			creator_id,
			updater_id,
			workspace_id,
			instance_id,
			project_id,
			name,
			sync_status,
			last_successful_sync_ts,
			fingerprint
		)
		VALUES (?, ?, ?, ?, ?, ?, 'NOT_FOUND', 0, '')
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, instance_id, project_id, name, sync_status, last_successful_sync, fingerprint
	`,
		create.WorkspaceId,
		create.CreatorId,
		create.CreatorId,
		create.InstanceId,
		create.ProjectId,
		create.Name,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var database api.Database
	if err := row.Scan(
		&database.ID,
		&database.CreatorId,
		&database.CreatedTs,
		&database.UpdaterId,
		&database.UpdatedTs,
		&database.WorkspaceId,
		&database.InstanceId,
		&database.ProjectId,
		&database.Name,
		&database.SyncStatus,
		&database.LastSuccessfulSyncTs,
		&database.Fingerprint,
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
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}
	if v := find.InstanceId; v != nil {
		where, args = append(where, "instance_id = ?"), append(args, *v)
	}
	if !find.IncludeAllDatabase {
		where = append(where, "name != '"+api.ALL_DATABASE_NAME+"'")
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			workspace_id,
			instance_id,
			project_id,
		    name,
		    sync_status,
			last_successful_sync_ts,
			fingerprint
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
		if err := rows.Scan(
			&database.ID,
			&database.CreatorId,
			&database.CreatedTs,
			&database.UpdaterId,
			&database.UpdatedTs,
			&database.WorkspaceId,
			&database.InstanceId,
			&database.ProjectId,
			&database.Name,
			&database.SyncStatus,
			&database.LastSuccessfulSyncTs,
			&database.Fingerprint,
		); err != nil {
			return nil, FormatError(err)
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
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.ProjectId; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE db
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, workspace_id, instance_id, project_id, name, sync_status, last_successful_sync, fingerprint
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var database api.Database
		if err := row.Scan(
			&database.ID,
			&database.CreatorId,
			&database.CreatedTs,
			&database.UpdaterId,
			&database.UpdatedTs,
			&database.WorkspaceId,
			&database.InstanceId,
			&database.ProjectId,
			&database.Name,
			&database.SyncStatus,
			&database.LastSuccessfulSyncTs,
			&database.Fingerprint,
		); err != nil {
			return nil, FormatError(err)
		}
		return &database, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("database ID not found: %d", patch.ID)}
}
