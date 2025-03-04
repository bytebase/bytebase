package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DataSourceMessage is the message for data source.
type DataSourceMessage struct {
	ID   string
	Type api.DataSourceType

	Username           string
	ObfuscatedPassword string
	ObfuscatedSslCa    string
	ObfuscatedSslCert  string
	ObfuscatedSslKey   string
	Host               string
	Port               string
	Database           string

	Options *storepb.DataSourceOptions
}

// FindDataSourceMessage is the message for finding a database.
type FindDataSourceMessage struct {
	ID         *string
	Name       *string
	InstanceID *string
	Type       *api.DataSourceType
}

// UpdateDataSourceMessage is the message for the data source.
type UpdateDataSourceMessage struct {
	InstanceID   string
	DataSourceID string

	Username           *string
	ObfuscatedPassword *string
	ObfuscatedSslCa    *string
	ObfuscatedSslCert  *string
	ObfuscatedSslKey   *string
	Host               *string
	Port               *string
	Database           *string
	Options            *storepb.DataSourceOptions
}

func (*Store) listInstanceDataSourceMap(ctx context.Context, tx *Tx, find *FindDataSourceMessage) (map[string][]*DataSourceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if find.ID != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *find.ID)
	}
	if find.Name != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *find.Name)
	}
	if find.InstanceID != nil {
		where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, *find.InstanceID)
	}
	if find.Type != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *find.Type)
	}

	instanceDataSourcesMap := make(map[string][]*DataSourceMessage)
	rows, err := tx.QueryContext(ctx, `
		SELECT
			instance,
			name,
			type,
			username,
			password,
			ssl_key,
			ssl_cert,
			ssl_ca,
			host,
			port,
			database,
			options
		FROM data_source
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var protoBytes []byte
		var instanceID string
		var dataSourceMessage DataSourceMessage
		if err := rows.Scan(
			&instanceID,
			&dataSourceMessage.ID,
			&dataSourceMessage.Type,
			&dataSourceMessage.Username,
			&dataSourceMessage.ObfuscatedPassword,
			&dataSourceMessage.ObfuscatedSslKey,
			&dataSourceMessage.ObfuscatedSslCert,
			&dataSourceMessage.ObfuscatedSslCa,
			&dataSourceMessage.Host,
			&dataSourceMessage.Port,
			&dataSourceMessage.Database,
			&protoBytes,
		); err != nil {
			return nil, err
		}
		dataSourceOptions := &storepb.DataSourceOptions{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(protoBytes, dataSourceOptions); err != nil {
			return nil, err
		}
		dataSourceMessage.Options = dataSourceOptions
		instanceDataSourcesMap[instanceID] = append(instanceDataSourcesMap[instanceID], &dataSourceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instanceDataSourcesMap, nil
}

func (s *Store) ListDataSourcesV2(ctx context.Context, find *FindDataSourceMessage) ([]*DataSourceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	dataSourceMap, err := s.listInstanceDataSourceMap(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	dataSources := []*DataSourceMessage{}
	for _, list := range dataSourceMap {
		dataSources = append(dataSources, list...)
	}
	return dataSources, nil
}

func (s *Store) GetDataSource(ctx context.Context, find *FindDataSourceMessage) (*DataSourceMessage, error) {
	dataSources, err := s.ListDataSourcesV2(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(dataSources) == 0 {
		return nil, nil
	}
	if len(dataSources) > 1 {
		return nil, errors.Errorf("found %d data sources, but expected 1", len(dataSources))
	}
	return dataSources[0], nil
}

// AddDataSourceToInstanceV2 adds a RO data source to an instance and return the instance where the data source is added.
func (s *Store) AddDataSourceToInstanceV2(ctx context.Context, instanceID string, dataSource *DataSourceMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceID, dataSource); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.New("Failed to commit transaction")
	}

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	return nil
}

// RemoveDataSourceV2 removes a RO data source from an instance.
func (s *Store) RemoveDataSourceV2(ctx context.Context, instanceID string, dataSourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		DELETE FROM data_source WHERE instance = $1 AND name = $2;
	`, instanceID, dataSourceID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return errors.Errorf("remove %d type data_sources for instance %s, but expected 1", rowsAffected, instanceID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	return nil
}

// UpdateDataSourceV2 updates a data source and returns the instance.
func (s *Store) UpdateDataSourceV2(ctx context.Context, patch *UpdateDataSourceMessage) error {
	set, args := []string{}, []any{}

	if v := patch.Username; v != nil {
		set, args = append(set, fmt.Sprintf("username = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ObfuscatedPassword; v != nil {
		set, args = append(set, fmt.Sprintf("password = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ObfuscatedSslKey; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_key = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ObfuscatedSslCert; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_cert = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ObfuscatedSslCa; v != nil {
		set, args = append(set, fmt.Sprintf("ssl_ca = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Host; v != nil {
		set, args = append(set, fmt.Sprintf("host = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Port; v != nil {
		set, args = append(set, fmt.Sprintf("port = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Database; v != nil {
		set, args = append(set, fmt.Sprintf("database = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Options; v != nil {
		o, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("options = $%d", len(args)+1)), append(args, o)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("failed to begin transaction")
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`UPDATE data_source SET %s WHERE instance = $%d AND name = $%d`, strings.Join(set, ", "), len(args)+1, len(args)+2)
	args = append(args, patch.InstanceID, patch.DataSourceID)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}
	if rowsAffected != 1 {
		return errors.Errorf("update %v data source records from instance %s, but expected one", rowsAffected, patch.InstanceID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.instanceCache.Remove(getInstanceCacheKey(patch.InstanceID))
	return nil
}

func (*Store) addDataSourceToInstanceImplV2(ctx context.Context, tx *Tx, instanceID string, dataSource *DataSourceMessage) error {
	protoBytes, err := protojson.Marshal(dataSource.Options)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data source options")
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO data_source (
			instance,
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, instanceID, dataSource.ID,
		dataSource.Type, dataSource.Username, dataSource.ObfuscatedPassword, dataSource.ObfuscatedSslKey,
		dataSource.ObfuscatedSslCert, dataSource.ObfuscatedSslCa, dataSource.Host, dataSource.Port,
		protoBytes, dataSource.Database,
	); err != nil {
		return err
	}

	return nil
}

// clearDataSourceImpl deletes dataSources by instance id and database id.
func (*Store) clearDataSourceImpl(ctx context.Context, tx *Tx, instanceID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM data_source WHERE instance = $1`, instanceID); err != nil {
		return err
	}
	return nil
}
