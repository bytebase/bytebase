package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

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
	Host     string
	Port     string
	Options  api.DataSourceOptions
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
		Host:     raw.Host,
		Port:     raw.Port,
		Options:  raw.Options,
	}
}

// CreateDataSource creates an instance of DataSource.
func (s *Store) CreateDataSource(ctx context.Context, create *api.DataSourceCreate) (*api.DataSource, error) {
	dataSourceRaw, err := s.createDataSourceRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create data source with DataSourceCreate[%+v]", create)
	}
	dataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose data source with dataSourceRaw[%+v]", dataSourceRaw)
	}
	return dataSource, nil
}

// GetDataSource gets an instance of DataSource.
func (s *Store) GetDataSource(ctx context.Context, find *api.DataSourceFind) (*api.DataSource, error) {
	dataSourceRaw, err := s.getDataSourceRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get data source with DataSourceFind[%+v]", find)
	}
	if dataSourceRaw == nil {
		return nil, nil
	}
	dataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose data source with dataSourceRaw[%+v]", dataSourceRaw)
	}
	return dataSource, nil
}

// findDataSource finds a list of DataSource instances.
func (s *Store) findDataSource(ctx context.Context, find *api.DataSourceFind) ([]*api.DataSource, error) {
	dataSourceRawList, err := s.findDataSourceRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find DataSource list with DataSourceFind[%+v]", find)
	}

	var dataSourceList []*api.DataSource
	for _, raw := range dataSourceRawList {
		dataSource, err := s.composeDataSource(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose DataSource role with dataSourceRaw[%+v]", raw)
		}
		dataSourceList = append(dataSourceList, dataSource)
	}
	return dataSourceList, nil
}

// PatchDataSource patches an instance of DataSource.
func (s *Store) PatchDataSource(ctx context.Context, patch *api.DataSourcePatch) (*api.DataSource, error) {
	dataSourceRaw, err := s.patchDataSourceRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch DataSource with DataSourcePatch[%+v]", patch)
	}
	dataSource, err := s.composeDataSource(ctx, dataSourceRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose DataSource role with dataSourceRaw[%+v]", dataSourceRaw)
	}
	return dataSource, nil
}

// DeleteDataSource deletes an existing dataSource by ID.
func (s *Store) DeleteDataSource(ctx context.Context, deleteDataSource *api.DataSourceDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	if err := s.deleteDataSourceImpl(ctx, tx, deleteDataSource); err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	// Invalidate the cache.
	s.cache.DeleteCache(dataSourceCacheNamespace, deleteDataSource.InstanceID)
	return nil
}

//
// private functions
//

// createDataSourceRawTx creates an instance of DataSource.
// This uses an existing transaction object.
func (s *Store) createDataSourceRawTx(ctx context.Context, tx *Tx, create *api.DataSourceCreate) error {
	if _, err := s.createDataSourceImpl(ctx, tx, create); err != nil {
		return errors.Wrapf(err, "failed to create data source with DataSourceCreate[%+v]", create)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(dataSourceCacheNamespace, create.InstanceID)
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
	defer tx.Rollback()

	dataSource, err := s.createDataSourceImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(dataSourceCacheNamespace, dataSource.InstanceID)

	return dataSource, nil
}

// findDataSourceRaw retrieves a list of data sources based on find.
func (s *Store) findDataSourceRaw(ctx context.Context, find *api.DataSourceFind) ([]*dataSourceRaw, error) {
	findCopy := *find
	findCopy.InstanceID = nil
	isListDataSource := find.InstanceID != nil && findCopy == api.DataSourceFind{}
	var cacheList []*dataSourceRaw
	has, err := s.cache.FindCache(dataSourceCacheNamespace, *find.InstanceID, &cacheList)
	if err != nil {
		return nil, err
	}
	if has && isListDataSource {
		return cacheList, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDataSourceImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if isListDataSource {
		if err := s.cache.UpsertCache(dataSourceCacheNamespace, *find.InstanceID, list); err != nil {
			return nil, err
		}
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
	defer tx.Rollback()

	list, err := s.findDataSourceImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d data sources with filter %+v, expect 1", len(list), find)}
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
	defer tx.Rollback()

	dataSource, err := s.patchDataSourceImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(dataSourceCacheNamespace, dataSource.InstanceID)

	return dataSource, nil
}

// createDataSourceImpl creates a new dataSource.
func (*Store) createDataSourceImpl(ctx context.Context, tx *Tx, create *api.DataSourceCreate) (*dataSourceRaw, error) {
	// Insert row into dataSource.
	query := `
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
			ssl_ca,
			host,
			port,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options
	`
	var dataSourceRaw dataSourceRaw
	if err := tx.QueryRowContext(ctx, query,
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
		create.Host,
		create.Port,
		create.Options,
	).Scan(
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
		&dataSourceRaw.Host,
		&dataSourceRaw.Port,
		&dataSourceRaw.Options,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &dataSourceRaw, nil
}

func (*Store) findDataSourceImpl(ctx context.Context, tx *Tx, find *api.DataSourceFind) ([]*dataSourceRaw, error) {
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
			ssl_ca,
			host,
			port,
			options
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
			&dataSourceRaw.Host,
			&dataSourceRaw.Port,
			&dataSourceRaw.Options,
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
func (*Store) patchDataSourceImpl(ctx context.Context, tx *Tx, patch *api.DataSourcePatch) (*dataSourceRaw, error) {
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
	if v := patch.Host; v != nil {
		set, args = append(set, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Options; v != nil {
		set, args = append(set, fmt.Sprintf("options= $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	var dataSourceRaw dataSourceRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE data_source
			SET `+strings.Join(set, ", ")+`
			WHERE id = $%d
			RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options
		`, len(args)),
		args...,
	).Scan(
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
		&dataSourceRaw.Host,
		&dataSourceRaw.Port,
		&dataSourceRaw.Options,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("DataSource not found with ID %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &dataSourceRaw, nil
}

// deleteDataSourceImpl permanently deletes a dataSource by ID.
func (*Store) deleteDataSourceImpl(ctx context.Context, tx *Tx, delete *api.DataSourceDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM data_source WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}

// clearDataSourceImpl deletes dataSources by instance id and database id.
func (*Store) clearDataSourceImpl(ctx context.Context, tx *Tx, instanceID, databaseID int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM data_source WHERE instance_id = $1 AND database_id = $2`, instanceID, databaseID); err != nil {
		return FormatError(err)
	}
	return nil
}

// DataSourceMessage is the mssage for data source.
type DataSourceMessage struct {
	Title    string
	Type     api.DataSourceType
	Username string
	Password string
	SslCa    string
	SslCert  string
	SslKey   string
	Host     string
	Port     string
	Database string
}

func (s *Store) listDataSourceV2(ctx context.Context, tx *Tx, instanceID int) ([]*DataSourceMessage, error) {
	var dataSourceMessages []*DataSourceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			database_id,
			name,
			type,
			username,
			password,
			ssl_key,
			ssl_cert,
			ssl_ca,
			host,
			port
		FROM data_source
		WHERE instance_id = $1`,
		instanceID,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var dataSourceMessage DataSourceMessage
		var databaseID int
		if err := rows.Scan(
			&databaseID,
			&dataSourceMessage.Title,
			&dataSourceMessage.Type,
			&dataSourceMessage.Username,
			&dataSourceMessage.Password,
			&dataSourceMessage.SslKey,
			&dataSourceMessage.SslCert,
			&dataSourceMessage.SslCa,
			&dataSourceMessage.Host,
			&dataSourceMessage.Port,
		); err != nil {
			return nil, FormatError(err)
		}

		database, err := s.getDatabaseRaw(ctx, &api.DatabaseFind{
			ID: &databaseID,
		})
		if err != nil {
			return nil, err
		}
		dataSourceMessage.Database = database.Name
		dataSourceMessages = append(dataSourceMessages, &dataSourceMessage)
	}

	return dataSourceMessages, nil
}
