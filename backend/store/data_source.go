package store

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DataSourceMessage is the message for data source.
type DataSourceMessage struct {
	ID                 string
	Type               api.DataSourceType
	Username           string
	ObfuscatedPassword string
	ObfuscatedSslCa    string
	ObfuscatedSslCert  string
	ObfuscatedSslKey   string
	Host               string
	Port               string
	Database           string
	// Flatten data source options.
	SRV                    bool
	AuthenticationDatabase string
	SID                    string
	ServiceName            string
	// SSH related.
	SSHHost                 string
	SSHPort                 string
	SSHUser                 string
	SSHObfuscatedPassword   string
	SSHObfuscatedPrivateKey string
	// SASL.
	SASLConfig *storepb.SASLConfig
	// Authentication
	AuthenticationPrivateKeyObfuscated string
	// external secret
	ExternalSecret           *storepb.DataSourceExternalSecret
	AuthenticationType       storepb.DataSourceOptions_AuthenticationType
	AdditionalAddresses      []*storepb.DataSourceOptions_Address
	ReplicaSet               string
	DirectConnection         bool
	Region                   string
	WarehouseID              string
	UseSSL                   bool
	RedisType                storepb.DataSourceOptions_RedisType
	MasterName               string
	MasterUsername           string
	MasterObfuscatedPassword string
	// Extra connection parameters
	ExtraConnectionParameters map[string]string
}

// FindDataSourceMessage is the message for finding a database.
type FindDataSourceMessage struct {
	ID         *string
	Name       *string
	InstanceID *string
	Type       *api.DataSourceType
}

// Copy returns a copy of the data source message.
func (m *DataSourceMessage) Copy() *DataSourceMessage {
	deepCopyAdditionalAddresses := slices.Clone[[]*storepb.DataSourceOptions_Address](m.AdditionalAddresses)
	return &DataSourceMessage{
		ID:                                 m.ID,
		Type:                               m.Type,
		Username:                           m.Username,
		ObfuscatedPassword:                 m.ObfuscatedPassword,
		ObfuscatedSslCa:                    m.ObfuscatedSslCa,
		ObfuscatedSslCert:                  m.ObfuscatedSslCert,
		ObfuscatedSslKey:                   m.ObfuscatedSslKey,
		Host:                               m.Host,
		Port:                               m.Port,
		Database:                           m.Database,
		SRV:                                m.SRV,
		AuthenticationDatabase:             m.AuthenticationDatabase,
		SID:                                m.SID,
		ServiceName:                        m.ServiceName,
		SSHHost:                            m.SSHHost,
		SSHPort:                            m.SSHPort,
		SSHUser:                            m.SSHUser,
		SSHObfuscatedPassword:              m.SSHObfuscatedPassword,
		SSHObfuscatedPrivateKey:            m.SSHObfuscatedPrivateKey,
		AuthenticationPrivateKeyObfuscated: m.AuthenticationPrivateKeyObfuscated,
		ExternalSecret:                     m.ExternalSecret,
		AuthenticationType:                 m.AuthenticationType,
		SASLConfig:                         m.SASLConfig,
		AdditionalAddresses:                deepCopyAdditionalAddresses,
		ReplicaSet:                         m.ReplicaSet,
		DirectConnection:                   m.DirectConnection,
		Region:                             m.Region,
		UseSSL:                             m.UseSSL,
		RedisType:                          m.RedisType,
		MasterName:                         m.MasterName,
		MasterUsername:                     m.MasterUsername,
		MasterObfuscatedPassword:           m.MasterObfuscatedPassword,
		ExtraConnectionParameters:          m.ExtraConnectionParameters,
	}
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
	// Flatten data source options.
	SRV                    *bool
	AuthenticationDatabase *string
	SID                    *string
	ServiceName            *string
	// SSH related.
	SSHHost                 *string
	SSHPort                 *string
	SSHUser                 *string
	SSHObfuscatedPassword   *string
	SSHObfuscatedPrivateKey *string
	// Authentication
	AuthenticationPrivateKeyObfuscated *string
	// external secret
	ExternalSecret       *storepb.DataSourceExternalSecret
	RemoveExternalSecret bool
	AuthenticationType   *storepb.DataSourceOptions_AuthenticationType
	// SASLConfig.
	SASLConfig        *storepb.SASLConfig
	AdditionalAddress *[]*storepb.DataSourceOptions_Address
	ReplicaSet        *string
	RemoveSASLConfig  bool
	DirectConnection  *bool
	Region            *string
	WarehouseID       *string
	UseSSL            *bool

	RedisType                *storepb.DataSourceOptions_RedisType
	MasterName               *string
	MasterUsername           *string
	MasterObfuscatedPassword *string
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
		var dataSourceOptions storepb.DataSourceOptions
		if err := common.ProtojsonUnmarshaler.Unmarshal(protoBytes, &dataSourceOptions); err != nil {
			return nil, err
		}
		dataSourceMessage.SRV = dataSourceOptions.Srv
		dataSourceMessage.AuthenticationDatabase = dataSourceOptions.AuthenticationDatabase
		dataSourceMessage.SID = dataSourceOptions.Sid
		dataSourceMessage.ServiceName = dataSourceOptions.ServiceName
		dataSourceMessage.SSHHost = dataSourceOptions.SshHost
		dataSourceMessage.SSHPort = dataSourceOptions.SshPort
		dataSourceMessage.SSHUser = dataSourceOptions.SshUser
		dataSourceMessage.SSHObfuscatedPassword = dataSourceOptions.SshObfuscatedPassword
		dataSourceMessage.SSHObfuscatedPrivateKey = dataSourceOptions.SshObfuscatedPrivateKey
		dataSourceMessage.AuthenticationPrivateKeyObfuscated = dataSourceOptions.AuthenticationPrivateKeyObfuscated
		dataSourceMessage.ExternalSecret = dataSourceOptions.ExternalSecret
		dataSourceMessage.SASLConfig = dataSourceOptions.SaslConfig
		dataSourceMessage.AuthenticationType = dataSourceOptions.AuthenticationType
		dataSourceMessage.AdditionalAddresses = dataSourceOptions.AdditionalAddresses
		dataSourceMessage.ReplicaSet = dataSourceOptions.ReplicaSet
		dataSourceMessage.DirectConnection = dataSourceOptions.DirectConnection
		dataSourceMessage.Region = dataSourceOptions.Region
		dataSourceMessage.WarehouseID = dataSourceOptions.WarehouseId
		dataSourceMessage.UseSSL = dataSourceOptions.UseSsl
		dataSourceMessage.RedisType = dataSourceOptions.RedisType
		dataSourceMessage.MasterName = dataSourceOptions.MasterName
		dataSourceMessage.MasterObfuscatedPassword = dataSourceOptions.MasterObfuscatedPassword
		dataSourceMessage.MasterUsername = dataSourceOptions.MasterUsername
		dataSourceMessage.ExtraConnectionParameters = dataSourceOptions.ExtraConnectionParameters
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

func getDataSourceOption(ctx context.Context, tx *Tx, find *FindDataSourceMessage) (*storepb.DataSourceOptions, error) {
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
	row := tx.QueryRowContext(ctx, `
		SELECT
			options
		FROM data_source
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if row.Err() != nil {
		return nil, row.Err()
	}
	var protoBytes []byte
	if err := row.Scan(&protoBytes); err != nil {
		return nil, err
	}
	var dataSourceOptions storepb.DataSourceOptions
	if err := common.ProtojsonUnmarshaler.Unmarshal(protoBytes, &dataSourceOptions); err != nil {
		return nil, err
	}

	return &dataSourceOptions, nil
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("failed to begin transaction")
	}
	defer tx.Rollback()

	dataSource, err := getDataSourceOption(ctx, tx, &FindDataSourceMessage{
		InstanceID: &patch.InstanceID,
		Name:       &patch.DataSourceID,
	})
	if err != nil {
		return err
	}
	if dataSource == nil {
		return status.Errorf(codes.NotFound, "data source not found: %s", patch.DataSourceID)
	}

	if v := patch.SRV; v != nil {
		dataSource.Srv = *v
	}
	if v := patch.AuthenticationDatabase; v != nil {
		dataSource.AuthenticationDatabase = *v
	}
	if v := patch.SID; v != nil {
		dataSource.Sid = *v
	}
	if v := patch.ServiceName; v != nil {
		dataSource.ServiceName = *v
	}
	if v := patch.SSHHost; v != nil {
		dataSource.SshHost = *v
	}
	if v := patch.SSHPort; v != nil {
		dataSource.SshPort = *v
	}
	if v := patch.SSHUser; v != nil {
		dataSource.SshUser = *v
	}
	if v := patch.SSHObfuscatedPassword; v != nil {
		dataSource.SshObfuscatedPassword = *v
	}
	if v := patch.SSHObfuscatedPrivateKey; v != nil {
		dataSource.SshObfuscatedPrivateKey = *v
	}
	if v := patch.AuthenticationPrivateKeyObfuscated; v != nil {
		dataSource.AuthenticationPrivateKeyObfuscated = *v
	}
	if v := patch.AuthenticationType; v != nil {
		dataSource.AuthenticationType = *v
	}
	if v := patch.ExternalSecret; v != nil {
		dataSource.ExternalSecret = v
	} else if patch.RemoveExternalSecret {
		dataSource.ExternalSecret = nil
	}
	if v := patch.SASLConfig; v != nil {
		dataSource.SaslConfig = v
	} else if patch.RemoveSASLConfig {
		dataSource.SaslConfig = nil
	}
	if v := patch.AdditionalAddress; v != nil {
		dataSource.AdditionalAddresses = *v
	}
	if v := patch.ReplicaSet; v != nil {
		dataSource.ReplicaSet = *v
	}
	if v := patch.DirectConnection; v != nil {
		dataSource.DirectConnection = *v
	}
	if v := patch.Region; v != nil {
		dataSource.Region = *v
	}
	if v := patch.WarehouseID; v != nil {
		dataSource.WarehouseId = *v
	}
	if v := patch.UseSSL; v != nil {
		dataSource.UseSsl = *v
	}
	if v := patch.RedisType; v != nil {
		dataSource.RedisType = *v
	}
	if v := patch.MasterName; v != nil {
		dataSource.MasterName = *v
	}
	if v := patch.MasterUsername; v != nil {
		dataSource.MasterUsername = *v
	}
	if v := patch.MasterObfuscatedPassword; v != nil {
		dataSource.MasterObfuscatedPassword = *v
	}
	protoBytes, err := protojson.Marshal(dataSource)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data source options")
	}
	set, args = append(set, fmt.Sprintf("options = $%d", len(args)+1)), append(args, protoBytes)

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
	// We flatten the data source fields in DataSourceMessage, so we need to compose them in store layer before INSERT.
	dataSourceOptions := storepb.DataSourceOptions{
		Srv:                                dataSource.SRV,
		AuthenticationDatabase:             dataSource.AuthenticationDatabase,
		Sid:                                dataSource.SID,
		ServiceName:                        dataSource.ServiceName,
		SshHost:                            dataSource.SSHHost,
		SshPort:                            dataSource.SSHPort,
		SshUser:                            dataSource.SSHUser,
		SshObfuscatedPassword:              dataSource.SSHObfuscatedPassword,
		SshObfuscatedPrivateKey:            dataSource.SSHObfuscatedPrivateKey,
		AuthenticationPrivateKeyObfuscated: dataSource.AuthenticationPrivateKeyObfuscated,
		ExternalSecret:                     dataSource.ExternalSecret,
		AuthenticationType:                 dataSource.AuthenticationType,
		SaslConfig:                         dataSource.SASLConfig,
		AdditionalAddresses:                dataSource.AdditionalAddresses,
		ReplicaSet:                         dataSource.ReplicaSet,
		DirectConnection:                   dataSource.DirectConnection,
		Region:                             dataSource.Region,
		WarehouseId:                        dataSource.WarehouseID,
		UseSsl:                             dataSource.UseSSL,
		RedisType:                          dataSource.RedisType,
		MasterName:                         dataSource.MasterName,
		MasterUsername:                     dataSource.MasterName,
		MasterObfuscatedPassword:           dataSource.MasterObfuscatedPassword,
		ExtraConnectionParameters:          dataSource.ExtraConnectionParameters,
	}
	protoBytes, err := protojson.Marshal(&dataSourceOptions)
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
