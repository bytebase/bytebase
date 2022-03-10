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
	_ api.DataSourceService = (*DataSourceService)(nil)
)

// DataSourceService represents a service for managing dataSource.
type DataSourceService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewDataSourceService returns a new instance of DataSourceService.
func NewDataSourceService(logger *zap.Logger, db *DB, cache api.CacheService) *DataSourceService {
	return &DataSourceService{l: logger, db: db, cache: cache}
}

// CreateDataSource creates a new dataSource.
func (s *DataSourceService) CreateDataSource(ctx context.Context, create *api.DataSourceCreate) (*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	dataSource, err := s.createDataSource(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// CreateDataSourceTx creates a data source with a transaction.
func (s *DataSourceService) CreateDataSourceTx(ctx context.Context, tx *sql.Tx, create *api.DataSourceCreate) (*api.DataSource, error) {
	return s.createDataSource(ctx, tx, create)
}

// FindDataSourceList retrieves a list of data sources based on find.
func (s *DataSourceService) FindDataSourceList(ctx context.Context, find *api.DataSourceFind) ([]*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDataSourceList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindDataSource retrieves a single dataSource based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *DataSourceService) FindDataSource(ctx context.Context, find *api.DataSourceFind) (*api.DataSource, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDataSourceList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d data sources with filter %+v, expect 1", len(list), find)}
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
	defer tx.PTx.Rollback()

	dataSource, err := s.patchDataSource(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// createDataSource creates a new dataSource.
func (s *DataSourceService) createDataSource(ctx context.Context, tx *sql.Tx, create *api.DataSourceCreate) (*api.DataSource, error) {
	// Insert row into dataSource.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO data_source (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			name,
			type,
			username,
			password
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password
	`,
		create.CreatorID,
		create.CreatorID,
		create.InstanceID,
		create.DatabaseID,
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
	if err := row.Scan(
		&dataSource.ID,
		&dataSource.CreatorID,
		&dataSource.CreatedTs,
		&dataSource.UpdaterID,
		&dataSource.UpdatedTs,
		&dataSource.InstanceID,
		&dataSource.DatabaseID,
		&dataSource.Name,
		&dataSource.Type,
		&dataSource.Username,
		&dataSource.Password,
	); err != nil {
		return nil, FormatError(err)
	}

	return &dataSource, nil
}

func (s *DataSourceService) findDataSourceList(ctx context.Context, tx *sql.Tx, find *api.DataSourceFind) ([]*api.DataSource, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, api.DataSourceType(*v))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
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

	// Iterate over result set and deserialize rows into dataSourceList.
	var dataSourceList []*api.DataSource
	for rows.Next() {
		var dataSource api.DataSource
		if err := rows.Scan(
			&dataSource.ID,
			&dataSource.CreatorID,
			&dataSource.CreatedTs,
			&dataSource.UpdaterID,
			&dataSource.UpdatedTs,
			&dataSource.InstanceID,
			&dataSource.DatabaseID,
			&dataSource.Name,
			&dataSource.Type,
			&dataSource.Username,
			&dataSource.Password,
		); err != nil {
			return nil, FormatError(err)
		}

		dataSourceList = append(dataSourceList, &dataSource)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return dataSourceList, nil
}

// patchDataSource updates a dataSource by ID. Returns the new state of the dataSource after update.
func (s *DataSourceService) patchDataSource(ctx context.Context, tx *sql.Tx, patch *api.DataSourcePatch) (*api.DataSource, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Username; v != nil {
		set, args = append(set, fmt.Sprintf("username = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Password; v != nil {
		set, args = append(set, fmt.Sprintf("password = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE data_source
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var dataSource api.DataSource
		if err := row.Scan(
			&dataSource.ID,
			&dataSource.CreatorID,
			&dataSource.CreatedTs,
			&dataSource.UpdaterID,
			&dataSource.UpdatedTs,
			&dataSource.InstanceID,
			&dataSource.DatabaseID,
			&dataSource.Name,
			&dataSource.Type,
			&dataSource.Username,
			&dataSource.Password,
		); err != nil {
			return nil, FormatError(err)
		}
		return &dataSource, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("dataSource ID not found: %d", patch.ID)}
}
