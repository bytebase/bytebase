package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.DataSourceService = (*DataSourceService)(nil)
)

// DataSourceService represents a service for managing dataSource.
type DataSourceService struct {
	l  *bytebase.Logger
	db *DB
}

// NewDataSourceService returns a new instance of DataSourceService.
func NewDataSourceService(logger *bytebase.Logger, db *DB) *DataSourceService {
	return &DataSourceService{l: logger, db: db}
}

// CreateDataSource creates a new dataSource.
func (s *DataSourceService) CreateDataSource(ctx context.Context, create *api.DataSourceCreate) (*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	dataSource, err := s.createDataSource(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// FindDataSourceList retrieves a list of data sources based on find.
func (s *DataSourceService) FindDataSourceList(ctx context.Context, find *api.DataSourceFind) ([]*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDataSourceList(ctx, tx, find)
	if err != nil {
		return []*api.DataSource{}, err
	}

	return list, nil
}

// FindDataSource retrieves a single dataSource based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *DataSourceService) FindDataSource(ctx context.Context, find *api.DataSourceFind) (*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDataSourceList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("data source not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Warnf("found mulitple data sources: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchDataSource updates an existing dataSource by ID.
// Returns ENOTFOUND if dataSource does not exist.
func (s *DataSourceService) PatchDataSource(ctx context.Context, patch *api.DataSourcePatch) (*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	dataSource, err := s.patchDataSource(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// createDataSource creates a new dataSource.
func (s *DataSourceService) createDataSource(ctx context.Context, tx *Tx, create *api.DataSourceCreate) (*api.DataSource, error) {
	// Insert row into dataSource.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO data_source (
			workspace_id,
			creator_id,
			updater_id,
			instance_id,
			database_id,
			name,
			type,
			username,
			password
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password
	`,
		create.WorkspaceId,
		create.CreatorId,
		create.CreatorId,
		create.InstanceId,
		create.DatabaseId,
		create.Name,
		create.Type,
		create.Username,
		create.Password,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var dataSource api.DataSource
	dataSource.Instance = &api.ResourceObject{}
	dataSource.Database = &api.ResourceObject{}
	if err := row.Scan(
		&dataSource.ID,
		&dataSource.WorkspaceId,
		&dataSource.CreatorId,
		&dataSource.CreatedTs,
		&dataSource.UpdaterId,
		&dataSource.UpdatedTs,
		&dataSource.Instance.ID,
		&dataSource.Database.ID,
		&dataSource.Name,
		&dataSource.Type,
		&dataSource.Username,
		&dataSource.Password,
	); err != nil {
		return nil, FormatError(err)
	}

	return &dataSource, nil
}

func (s *DataSourceService) findDataSourceList(ctx context.Context, tx *Tx, find *api.DataSourceFind) (_ []*api.DataSource, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}
	if v := find.InstanceId; v != nil {
		where, args = append(where, "instance_id = ?"), append(args, *v)
	}
	if v := find.DatabaseId; v != nil {
		where, args = append(where, "`database_id` = ?"), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, "`type` = ?"), append(args, api.DataSourceType(*v))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
			workspace_id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			instance_id,
			database_id,
		    name,
		    type,
			username,
			password
		FROM data_source
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.DataSource, 0)
	for rows.Next() {
		var dataSource api.DataSource
		dataSource.Instance = &api.ResourceObject{}
		dataSource.Database = &api.ResourceObject{}
		if err := rows.Scan(
			&dataSource.ID,
			&dataSource.WorkspaceId,
			&dataSource.CreatorId,
			&dataSource.CreatedTs,
			&dataSource.UpdaterId,
			&dataSource.UpdatedTs,
			&dataSource.Instance.ID,
			&dataSource.Database.ID,
			&dataSource.Name,
			&dataSource.Type,
			&dataSource.Username,
			&dataSource.Password,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &dataSource)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchDataSource updates a dataSource by ID. Returns the new state of the dataSource after update.
func (s *DataSourceService) patchDataSource(ctx context.Context, tx *Tx, patch *api.DataSourcePatch) (*api.DataSource, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.Username; v != nil {
		set, args = append(set, "username = ?"), append(args, *v)
	}
	if v := patch.Password; v != nil {
		set, args = append(set, "password = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE data_source
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var dataSource api.DataSource
		dataSource.Instance = &api.ResourceObject{}
		dataSource.Database = &api.ResourceObject{}
		if err := row.Scan(
			&dataSource.ID,
			&dataSource.WorkspaceId,
			&dataSource.CreatorId,
			&dataSource.CreatedTs,
			&dataSource.UpdaterId,
			&dataSource.UpdatedTs,
			&dataSource.Instance.ID,
			&dataSource.Database.ID,
			&dataSource.Name,
			&dataSource.Type,
			&dataSource.Username,
			&dataSource.Password,
		); err != nil {
			return nil, FormatError(err)
		}
		return &dataSource, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("dataSource ID not found: %d", patch.ID)}
}
