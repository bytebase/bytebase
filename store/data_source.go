package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// dataSourceRaw is the store model for an DataSource.
// Fields have exactly the same meanings as DataSource.
type dataSourceRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	InstanceID int
	DatabaseID int

	// Domain specific fields
	Name     string
	Type     api.DataSourceType
	Username string
	Password string
	SslCa    string
	SslCert  string
	SslKey   string
}

// toDataSource creates an instance of DataSource based on the dataSourceRaw.
// This is intended to be called when we need to compose an DataSource relationship.
func (raw *dataSourceRaw) toDataSource() *api.DataSource {
	return &api.DataSource{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		InstanceID: raw.InstanceID,
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:     raw.Name,
		Type:     raw.Type,
		Username: raw.Username,
		Password: raw.Password,
		SslCa:    raw.SslCa,
		SslCert:  raw.SslCert,
		SslKey:   raw.SslKey,
	}
}

// CreateDataSource creates an instance of DataSource.
func (s *Store) CreateDataSource(ctx context.Context, create *api.DataSourceCreate) (*api.DataSource, error) {
	dataSourceRaw, err := s.createDataSourceRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create data source with DataSourceCreate[%+v], error[%w]", create, err)
	}
	dataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose data source with dataSourceRaw[%+v], error[%w]", dataSourceRaw, err)
	}
	return dataSource, nil
}

// GetDataSource gets an instance of DataSource
func (s *Store) GetDataSource(ctx context.Context, find *api.DataSourceFind) (*api.DataSource, error) {
	dataSourceRaw, err := s.getDataSourceRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source with DataSourceFind[%+v], error[%w]", find, err)
	}
	if dataSourceRaw == nil {
		return nil, nil
	}
	dataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose data source with dataSourceRaw[%+v], error[%w]", dataSourceRaw, err)
	}
	return dataSource, nil
}

// FindDataSource finds a list of DataSource instances
func (s *Store) FindDataSource(ctx context.Context, find *api.DataSourceFind) ([]*api.DataSource, error) {
	DataSourceRawList, err := s.findDataSourceRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find DataSource list with DataSourceFind[%+v], error[%w]", find, err)
	}
	var DataSourceList []*api.DataSource
	for _, raw := range DataSourceRawList {
		DataSource, err := s.composeDataSource(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose DataSource role with dataSourceRaw[%+v], error[%w]", raw, err)
		}
		DataSourceList = append(DataSourceList, DataSource)
	}
	return DataSourceList, nil
}

// PatchDataSource patches an instance of DataSource
func (s *Store) PatchDataSource(ctx context.Context, patch *api.DataSourcePatch) (*api.DataSource, error) {
	dataSourceRaw, err := s.patchDataSourceRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch DataSource with DataSourcePatch[%+v], error[%w]", patch, err)
	}
	DataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose DataSource role with dataSourceRaw[%+v], error[%w]", dataSourceRaw, err)
	}
	return DataSource, nil
}

//
// private functions
//

// createDataSourceRawTx creates an instance of DataSource.
// This uses an existing transaction object.
func (s *Store) createDataSourceRawTx(ctx context.Context, tx *sql.Tx, create *api.DataSourceCreate) error {
	_, err := s.createDataSourceImpl(ctx, tx, create)
	if err != nil {
		return fmt.Errorf("failed to create data source with DataSourceCreate[%+v], error[%w]", create, err)
	}
	return nil
}

func (s *Store) composeDataSource(ctx context.Context, raw *dataSourceRaw) (*api.DataSource, error) {
	dataSource := raw.toDataSource()

	creator, err := s.GetPrincipalByID(ctx, dataSource.CreatorID)
	if err != nil {
		return nil, err
	}
	dataSource.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, dataSource.UpdaterID)
	if err != nil {
		return nil, err
	}
	dataSource.Updater = updater

	return dataSource, nil
}

// createDataSourceRaw creates a new dataSource.
func (s *Store) createDataSourceRaw(ctx context.Context, create *api.DataSourceCreate) (*dataSourceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	dataSource, err := s.createDataSourceImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// findDataSourceRaw retrieves a list of data sources based on find.
func (s *Store) findDataSourceRaw(ctx context.Context, find *api.DataSourceFind) ([]*dataSourceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDataSourceImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getDataSourceRaw retrieves a single dataSource based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getDataSourceRaw(ctx context.Context, find *api.DataSourceFind) (*dataSourceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDataSourceImpl(ctx, tx.PTx, find)
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

// patchDataSourceRaw updates an existing dataSource by ID.
// Returns ENOTFOUND if dataSource does not exist.
func (s *Store) patchDataSourceRaw(ctx context.Context, patch *api.DataSourcePatch) (*dataSourceRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	dataSource, err := s.patchDataSourceImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dataSource, nil
}

// createDataSourceImpl creates a new dataSource.
func (s *Store) createDataSourceImpl(ctx context.Context, tx *sql.Tx, create *api.DataSourceCreate) (*dataSourceRaw, error) {
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
			password,
			ssl_key,
			ssl_cert,
			ssl_ca
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca
	`,
		create.CreatorID,
		create.CreatorID,
		create.InstanceID,
		create.DatabaseID,
		create.Name,
		create.Type,
		create.Username,
		create.Password,
		create.SslKey,
		create.SslCert,
		create.SslCa,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var dataSourceRaw dataSourceRaw
	if err := row.Scan(
		&dataSourceRaw.ID,
		&dataSourceRaw.CreatorID,
		&dataSourceRaw.CreatedTs,
		&dataSourceRaw.UpdaterID,
		&dataSourceRaw.UpdatedTs,
		&dataSourceRaw.InstanceID,
		&dataSourceRaw.DatabaseID,
		&dataSourceRaw.Name,
		&dataSourceRaw.Type,
		&dataSourceRaw.Username,
		&dataSourceRaw.Password,
		&dataSourceRaw.SslKey,
		&dataSourceRaw.SslCert,
		&dataSourceRaw.SslCa,
	); err != nil {
		return nil, FormatError(err)
	}

	return &dataSourceRaw, nil
}

func (s *Store) findDataSourceImpl(ctx context.Context, tx *sql.Tx, find *api.DataSourceFind) ([]*dataSourceRaw, error) {
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
			password,
			ssl_key,
			ssl_cert,
			ssl_ca
		FROM data_source
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into dataSourceRawList.
	var dataSourceRawList []*dataSourceRaw
	for rows.Next() {
		var dataSourceRaw dataSourceRaw
		if err := rows.Scan(
			&dataSourceRaw.ID,
			&dataSourceRaw.CreatorID,
			&dataSourceRaw.CreatedTs,
			&dataSourceRaw.UpdaterID,
			&dataSourceRaw.UpdatedTs,
			&dataSourceRaw.InstanceID,
			&dataSourceRaw.DatabaseID,
			&dataSourceRaw.Name,
			&dataSourceRaw.Type,
			&dataSourceRaw.Username,
			&dataSourceRaw.Password,
			&dataSourceRaw.SslKey,
			&dataSourceRaw.SslCert,
			&dataSourceRaw.SslCa,
		); err != nil {
			return nil, FormatError(err)
		}

		dataSourceRawList = append(dataSourceRawList, &dataSourceRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return dataSourceRawList, nil
}

// patchDataSourceImpl updates a dataSource by ID. Returns the new state of the dataSource after update.
func (s *Store) patchDataSourceImpl(ctx context.Context, tx *sql.Tx, patch *api.DataSourcePatch) (*dataSourceRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Username; v != nil {
		set, args = append(set, fmt.Sprintf("username = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Password; v != nil {
		set, args = append(set, fmt.Sprintf("password = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslCa; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_ca= $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslKey; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_key= $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslCert; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_cert= $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE data_source
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var dataSourceRaw dataSourceRaw
		if err := row.Scan(
			&dataSourceRaw.ID,
			&dataSourceRaw.CreatorID,
			&dataSourceRaw.CreatedTs,
			&dataSourceRaw.UpdaterID,
			&dataSourceRaw.UpdatedTs,
			&dataSourceRaw.InstanceID,
			&dataSourceRaw.DatabaseID,
			&dataSourceRaw.Name,
			&dataSourceRaw.Type,
			&dataSourceRaw.Username,
			&dataSourceRaw.Password,
			&dataSourceRaw.SslKey,
			&dataSourceRaw.SslCert,
			&dataSourceRaw.SslCa,
		); err != nil {
			return nil, FormatError(err)
		}
		return &dataSourceRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("DataSource not found with ID %d", patch.ID)}
}
