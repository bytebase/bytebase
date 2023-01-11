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
	Database string
}

// toDataSource creates an instance of DataSource based on the dataSourceRaw.
// This is intended to be called when we need to compose an DataSource relationship.
func (raw *dataSourceRaw) toDataSource() *api.DataSource {
	return &api.DataSource{
		ID: raw.ID,

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
		Database: raw.Database,
	}
}

// CreateDataSource creates an instance of DataSource.
func (s *Store) CreateDataSource(ctx context.Context, instance *api.Instance, create *api.DataSourceCreate) (*api.DataSource, error) {
	dataSourceRaw, err := s.createDataSourceRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create data source with DataSourceCreate[%+v]", create)
	}
	s.instanceCache.Delete(getInstanceCacheKey(instance.Environment.ResourceID, instance.ResourceID))
	s.instanceIDCache.Delete(instance.ID)
	return composeDataSource(dataSourceRaw), nil
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
	return composeDataSource(dataSourceRaw), nil
}

// findDataSource finds a list of DataSource instances.
func (s *Store) findDataSource(ctx context.Context, find *api.DataSourceFind) ([]*api.DataSource, error) {
	dataSourceRawList, err := s.findDataSourceRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find DataSource list with DataSourceFind[%+v]", find)
	}

	var dataSourceList []*api.DataSource
	for _, raw := range dataSourceRawList {
		dataSourceList = append(dataSourceList, composeDataSource(raw))
	}
	return dataSourceList, nil
}

// PatchDataSource patches an instance of DataSource.
func (s *Store) PatchDataSource(ctx context.Context, instance *api.Instance, patch *api.DataSourcePatch) (*api.DataSource, error) {
	dataSourceRaw, err := s.patchDataSourceRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch DataSource with DataSourcePatch[%+v]", patch)
	}
	s.instanceCache.Delete(getInstanceCacheKey(instance.Environment.ResourceID, instance.ResourceID))
	s.instanceIDCache.Delete(instance.ID)
	return composeDataSource(dataSourceRaw), nil
}

// DeleteDataSource deletes an existing dataSource by ID.
func (s *Store) DeleteDataSource(ctx context.Context, instance *api.Instance, deleteDataSource *api.DataSourceDelete) error {
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
	s.instanceCache.Delete(getInstanceCacheKey(instance.Environment.ResourceID, instance.ResourceID))
	s.instanceIDCache.Delete(instance.ID)
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

func composeDataSource(raw *dataSourceRaw) *api.DataSource {
	return raw.toDataSource()
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

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
			options,
			database
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database
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
		create.Database,
	).Scan(
		&dataSourceRaw.ID,
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
		&dataSourceRaw.Database,
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
			options,
			database
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
			&dataSourceRaw.Database,
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
	if v := patch.Database; v != nil {
		set, args = append(set, fmt.Sprintf("database = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	var dataSourceRaw dataSourceRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE data_source
			SET `+strings.Join(set, ", ")+`
			WHERE id = $%d
			RETURNING id, instance_id, database_id, name, type, username, password, ssl_key, ssl_cert, ssl_ca, host, port, options, database
		`, len(args)),
		args...,
	).Scan(
		&dataSourceRaw.ID,
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
		&dataSourceRaw.Database,
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

// DataSourceMessage is the message for data source.
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
	// Flatten data source options.
	SRV                    bool
	AuthenticationDatabase string
}

// UpdateDataSourceMessage is the message for the data source.
type UpdateDataSourceMessage struct {
	UpdaterID   int
	InstanceUID int
	Type        api.DataSourceType

	Username *string
	Password *string
	SslCa    *string
	SslCert  *string
	SslKey   *string
	Host     *string
	Port     *string
	// Flatten data source options.
	SRV                    *bool
	AuthenticationDatabase *string
}

func (*Store) listDataSourceV2(ctx context.Context, tx *Tx, instanceID string) ([]*DataSourceMessage, error) {
	var dataSourceMessages []*DataSourceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			data_source.name,
			data_source.type,
			data_source.username,
			data_source.password,
			data_source.ssl_key,
			data_source.ssl_cert,
			data_source.ssl_ca,
			data_source.host,
			data_source.port,
			data_source.database,
			data_source.options
		FROM data_source
		LEFT JOIN instance ON instance.id = data_source.instance_id
		WHERE instance.resource_id = $1`,
		instanceID,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	var dataSourceOptions api.DataSourceOptions
	for rows.Next() {
		var dataSourceMessage DataSourceMessage
		if err := rows.Scan(
			&dataSourceMessage.Title,
			&dataSourceMessage.Type,
			&dataSourceMessage.Username,
			&dataSourceMessage.Password,
			&dataSourceMessage.SslKey,
			&dataSourceMessage.SslCert,
			&dataSourceMessage.SslCa,
			&dataSourceMessage.Host,
			&dataSourceMessage.Port,
			&dataSourceMessage.Database,
			&dataSourceOptions,
		); err != nil {
			return nil, FormatError(err)
		}
		dataSourceMessage.SRV = dataSourceOptions.SRV
		dataSourceMessage.AuthenticationDatabase = dataSourceOptions.AuthenticationDatabase

		dataSourceMessages = append(dataSourceMessages, &dataSourceMessage)
	}

	return dataSourceMessages, nil
}

// AddDataSourceToInstanceV2 adds a RO data source to an instance and return the instance where the data source is added.
func (s *Store) AddDataSourceToInstanceV2(ctx context.Context, instanceUID, creatorID int, dataSource *DataSourceMessage) (*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	instance, err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceUID, creatorID, dataSource)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.New("Failed to commit transaction")
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// RemoveDataSourceV2 removes a RO data source from an instance and return the instance where the data source had beed removed from.
func (s *Store) RemoveDataSourceV2(ctx context.Context, instanceUID int, dataSourceTp api.DataSourceType) (*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		DELETE FROM data_source WHERE instance_id = $1 AND type = $2;
	`, instanceUID, dataSourceTp)
	if err != nil {
		return nil, FormatError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return nil, errors.Errorf("deleted %v read-only data source records from instance %v, but expected one", rowsAffected, instanceUID)
	}

	instance, err := s.findInstanceImplV2(ctx, tx, &FindInstanceMessage{
		UID: &instanceUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find instance")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// UpdateDataSourceV2 updates a data source and returns the instance.
func (s *Store) UpdateDataSourceV2(ctx context.Context, patch *UpdateDataSourceMessage) (*InstanceMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}

	if v := patch.Username; v != nil {
		set, args = append(set, fmt.Sprintf("username = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Password; v != nil {
		set, args = append(set, fmt.Sprintf("password = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslKey; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_key = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslCert; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_cert = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SslCa; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_ca = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Host; v != nil {
		set, args = append(set, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}

	// Use jsonb_build_object to build the jsonb object to update some fields in jsonb instead of whole column.
	var optionSet []string
	if v := patch.SRV; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('srv', to_jsonb($%d::BOOLEAN))", len(args)+1)), append(args, *v)
	}
	if v := patch.AuthenticationDatabase; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('authenticationDatabase', to_jsonb($%d::TEXT))", len(args)+1)), append(args, *v)
	}
	if len(optionSet) != 0 {
		set = append(set, fmt.Sprintf(`options = options || %s`, strings.Join(optionSet, "||")))
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	// Only update the data source if the
	query := `UPDATE data_source SET ` + strings.Join(set, ", ") +
		` WHERE instance_id = ` + fmt.Sprintf("%d", patch.InstanceUID) +
		` AND type = ` + fmt.Sprintf(`'%s'`, patch.Type)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, FormatError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return nil, errors.Errorf("update %v data source records from instance %v, but expected one", rowsAffected, patch.InstanceUID)
	}

	instance, err := s.findInstanceImplV2(ctx, tx, &FindInstanceMessage{
		UID: &patch.InstanceUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find instance")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

func (s *Store) addDataSourceToInstanceImplV2(ctx context.Context, tx *Tx, instanceUID, creatorID int, dataSource *DataSourceMessage) (*InstanceMessage, error) {
	// We flatten the data source fields in DataSourceMessage, so we need to compose them in store layer before INSERT.
	dataSourceOptions := api.DataSourceOptions{
		SRV:                    dataSource.SRV,
		AuthenticationDatabase: dataSource.AuthenticationDatabase,
	}

	if _, err := tx.QueryContext(ctx, `
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
			options,
			database
		)
		SELECT $1, $2, $3, data_source.database_id, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			FROM 
				data_source JOIN db ON data_source.database_id = db.id
			WHERE data_source.instance_id = $15 AND db.name = $16;
	`, creatorID, creatorID, instanceUID, dataSource.Title,
		dataSource.Type, dataSource.Username, dataSource.Password, dataSource.SslKey,
		dataSource.SslCert, dataSource.SslCa, dataSource.Host, dataSource.Port,
		dataSourceOptions, dataSource.Database, instanceUID, api.AllDatabaseName,
	); err != nil {
		return nil, FormatError(err)
	}

	instance, err := s.findInstanceImplV2(ctx, tx, &FindInstanceMessage{
		UID: &instanceUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find instance with instance uid %d", instanceUID)
	}

	return instance, nil
}
