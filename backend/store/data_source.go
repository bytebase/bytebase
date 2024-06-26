package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

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
	// (deprecated) Output only.
	UID int
	// external secret
	ExternalSecret      *storepb.DataSourceExternalSecret
	AuthenticationType  storepb.DataSourceOptions_AuthenticationType
	AdditionalAddresses []*storepb.DataSourceOptions_Address
	ReplicaSet          string
	DirectConnection    bool
	Region              string
	AccountID           string
	WarehouseID         string
}

// Copy returns a copy of the data source message.
func (m *DataSourceMessage) Copy() *DataSourceMessage {
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
		UID:                                m.UID,
		AuthenticationPrivateKeyObfuscated: m.AuthenticationPrivateKeyObfuscated,
		ExternalSecret:                     m.ExternalSecret,
		AuthenticationType:                 m.AuthenticationType,
		SASLConfig:                         m.SASLConfig,
		AdditionalAddresses:                m.AdditionalAddresses,
		ReplicaSet:                         m.ReplicaSet,
		DirectConnection:                   m.DirectConnection,
		Region:                             m.Region,
	}
}

// UpdateDataSourceMessage is the message for the data source.
type UpdateDataSourceMessage struct {
	UpdaterID    int
	InstanceUID  int
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
	AccountID         *string
	WarehouseID       *string
}

func (*Store) listDataSourceV2(ctx context.Context, tx *Tx, instanceID string) ([]*DataSourceMessage, error) {
	var dataSourceMessages []*DataSourceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			data_source.id,
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
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var protoBytes []byte
		var dataSourceMessage DataSourceMessage
		if err := rows.Scan(
			&dataSourceMessage.UID,
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
		dataSourceMessage.AccountID = dataSourceOptions.AccountId
		dataSourceMessage.WarehouseID = dataSourceOptions.WarehouseId
		dataSourceMessages = append(dataSourceMessages, &dataSourceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dataSourceMessages, nil
}

// AddDataSourceToInstanceV2 adds a RO data source to an instance and return the instance where the data source is added.
func (s *Store) AddDataSourceToInstanceV2(ctx context.Context, instanceUID, creatorID int, instanceID string, dataSource *DataSourceMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceUID, creatorID, dataSource); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.New("Failed to commit transaction")
	}

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	s.instanceIDCache.Remove(instanceUID)
	return nil
}

// RemoveDataSourceV2 removes a RO data source from an instance.
func (s *Store) RemoveDataSourceV2(ctx context.Context, instanceUID int, instanceID string, dataSourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("Failed to begin transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		DELETE FROM data_source WHERE data_source.instance_id = $1 AND data_source.name = $2;
	`, instanceUID, dataSourceID)
	if err != nil {
		return err
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

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	s.instanceIDCache.Remove(instanceUID)
	return nil
}

// UpdateDataSourceV2 updates a data source and returns the instance.
func (s *Store) UpdateDataSourceV2(ctx context.Context, patch *UpdateDataSourceMessage) error {
	set, args := []string{"updater_id = $1", "updated_ts = $2"}, []any{patch.UpdaterID, time.Now().Unix()}

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

	// Use jsonb_build_object to build the jsonb object to update some fields in jsonb instead of whole column.
	// To view the json tag, please refer to the struct definition of storepb.DataSourceOptions.
	var optionSet []string
	if v := patch.SRV; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('srv', $%d::BOOLEAN)", len(args)+1)), append(args, *v)
	}
	if v := patch.AuthenticationDatabase; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('authenticationDatabase', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SID; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sid', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.ServiceName; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('serviceName', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SSHHost; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sshHost', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SSHPort; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sshPort', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SSHUser; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sshUser', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SSHObfuscatedPassword; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sshObfuscatedPassword', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.SSHObfuscatedPrivateKey; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('sshObfuscatedPrivateKey', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.AuthenticationPrivateKeyObfuscated; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('authenticationPrivateKeyObfuscated', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.AuthenticationType; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('authenticationType', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalSecret; v != nil {
		protoBytes, err := protojson.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "failed to marshal external secret")
		}
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('externalSecret', $%d::JSONB)", len(args)+1)), append(args, protoBytes)
	} else if patch.RemoveExternalSecret {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('externalSecret', $%d::JSONB)", len(args)+1)), append(args, nil)
	}
	if v := patch.SASLConfig; v != nil {
		protoBytes, err := protojson.Marshal(v)
		if err != nil {
			return errors.Wrap(err, "failed to marshal sasl config")
		}
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('saslConfig', $%d::JSONB)", len(args)+1)), append(args, protoBytes)
	} else if patch.RemoveSASLConfig {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('saslConfig', $%d::JSONB)", len(args)+1)), append(args, nil)
	}
	if v := patch.AdditionalAddress; v != nil {
		partialDataSourceOptions := &storepb.DataSourceOptions{
			AdditionalAddresses: *v,
		}
		protoBytes, err := protojson.Marshal(partialDataSourceOptions)
		if err != nil {
			return errors.Wrap(err, "failed to marshal additional address")
		}
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('additionalAddresses', ($%d::JSONB)->'additionalAddresses')", len(args)+1)), append(args, protoBytes)
	}
	if v := patch.ReplicaSet; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('replicaSet', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.DirectConnection; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('directConnection', $%d::BOOLEAN)", len(args)+1)), append(args, *v)
	}
	if v := patch.Region; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('region', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.AccountID; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('accountId', $%d::TEXT)", len(args)+1)), append(args, *v)
	}
	if v := patch.WarehouseID; v != nil {
		optionSet, args = append(optionSet, fmt.Sprintf("jsonb_build_object('warehouseId', $%d::TEXT)", len(args)+1)), append(args, *v)
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
		` AND name = ` + fmt.Sprintf(`'%s'`, patch.DataSourceID)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
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

	s.instanceCache.Remove(getInstanceCacheKey(patch.InstanceID))
	s.instanceIDCache.Remove(patch.InstanceUID)
	return nil
}

func (*Store) addDataSourceToInstanceImplV2(ctx context.Context, tx *Tx, instanceUID, creatorID int, dataSource *DataSourceMessage) error {
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
		AccountId:                          dataSource.AccountID,
	}
	protoBytes, err := protojson.Marshal(&dataSourceOptions)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data source options")
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO data_source (
			creator_id,
			updater_id,
			instance_id,
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, creatorID, creatorID, instanceUID, dataSource.ID,
		dataSource.Type, dataSource.Username, dataSource.ObfuscatedPassword, dataSource.ObfuscatedSslKey,
		dataSource.ObfuscatedSslCert, dataSource.ObfuscatedSslCa, dataSource.Host, dataSource.Port,
		protoBytes, dataSource.Database,
	); err != nil {
		return err
	}

	return nil
}

// clearDataSourceImpl deletes dataSources by instance id and database id.
func (*Store) clearDataSourceImpl(ctx context.Context, tx *Tx, instanceID int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM data_source WHERE instance_id = $1`, instanceID); err != nil {
		return err
	}
	return nil
}
