package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

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
	// (deprecated) Output only.
	UID        int
	DatabaseID int
}

// UpdateDataSourceMessage is the message for the data source.
type UpdateDataSourceMessage struct {
	UpdaterID     int
	InstanceUID   int
	EnvironmentID string
	InstanceID    string

	Type api.DataSourceType

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
			data_source.id,
			data_source.database_id,
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
			&dataSourceMessage.UID,
			&dataSourceMessage.DatabaseID,
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
func (s *Store) AddDataSourceToInstanceV2(ctx context.Context, instanceUID, creatorID int, environmentID, instanceID string, dataSource *DataSourceMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	allDatabaseName := api.AllDatabaseName
	allDatabase, err := s.getDatabaseImplV2(ctx, tx, &FindDatabaseMessage{
		InstanceID:         &instanceID,
		DatabaseName:       &allDatabaseName,
		IncludeAllDatabase: true,
	})
	if err != nil {
		return err
	}

	if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceUID, allDatabase.UID, creatorID, dataSource); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.New("Failed to commit transaction")
	}

	s.instanceCache.Delete(getInstanceCacheKey(environmentID, instanceID))
	s.instanceIDCache.Delete(instanceUID)
	return nil
}

// RemoveDataSourceV2 removes a RO data source from an instance.
func (s *Store) RemoveDataSourceV2(ctx context.Context, instanceUID int, environmentID, instanceID string, dataSourceTp api.DataSourceType) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		DELETE FROM data_source WHERE data_source.instance_id = $1 AND data_source.type = $2;
	`, instanceUID, dataSourceTp)
	if err != nil {
		return FormatError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return errors.Errorf("remove %d type data_sources for instance uid %d, but expected 1", rowsAffected, instanceUID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Delete(getInstanceCacheKey(environmentID, instanceID))
	s.instanceIDCache.Delete(instanceUID)
	return nil
}

// UpdateDataSourceV2 updates a data source and returns the instance.
func (s *Store) UpdateDataSourceV2(ctx context.Context, patch *UpdateDataSourceMessage) error {
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
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	// Only update the data source if the
	query := `UPDATE data_source SET ` + strings.Join(set, ", ") +
		` WHERE instance_id = ` + fmt.Sprintf("%d", patch.InstanceUID) +
		` AND type = ` + fmt.Sprintf(`'%s'`, patch.Type)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return FormatError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return errors.Errorf("update %v data source records from instance %v, but expected one", rowsAffected, patch.InstanceUID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Delete(getInstanceCacheKey(patch.EnvironmentID, patch.InstanceID))
	s.instanceIDCache.Delete(patch.InstanceUID)
	return nil
}

func (*Store) addDataSourceToInstanceImplV2(ctx context.Context, tx *Tx, instanceUID, databaseUID, creatorID int, dataSource *DataSourceMessage) error {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, creatorID, creatorID, instanceUID, databaseUID, dataSource.Title,
		dataSource.Type, dataSource.Username, dataSource.Password, dataSource.SslKey,
		dataSource.SslCert, dataSource.SslCa, dataSource.Host, dataSource.Port,
		dataSourceOptions, dataSource.Database,
	); err != nil {
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
