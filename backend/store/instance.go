package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// InstanceMessage is the message for instance.
type InstanceMessage struct {
	ResourceID    string
	EnvironmentID string
	Deleted       bool
	Metadata      *storepb.Instance
}

// UpdateInstanceMessage is the message for updating an instance.
type UpdateInstanceMessage struct {
	ResourceID string

	Deleted       *bool
	EnvironmentID *string
	Metadata      *storepb.Instance
}

// FindInstanceMessage is the message for finding instances.
type FindInstanceMessage struct {
	ResourceID  *string
	ResourceIDs *[]string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	Filter      *ListResourceFilter
}

// GetInstanceV2 gets an instance by the resource_id.
func (s *Store) GetInstanceV2(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.instanceCache.Get(getInstanceCacheKey(*find.ResourceID)); ok && s.enableCache {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	instances, err := s.ListInstancesV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list instances with find instance message %+v", find)
	}
	if len(instances) == 0 {
		return nil, nil
	}
	if len(instances) > 1 {
		return nil, errors.Errorf("find %d instances with find instance message %+v, expected 1", len(instances), find)
	}

	instance := instances[0]
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	return instance, nil
}

// ListInstancesV2 lists all instance.
func (s *Store) ListInstancesV2(ctx context.Context, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, instance := range instances {
		s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	}
	return instances, nil
}

// CreateInstanceV2 creates an instance.
func (s *Store) CreateInstanceV2(ctx context.Context, instanceCreate *InstanceMessage) (*InstanceMessage, error) {
	if err := validateDataSources(instanceCreate.Metadata); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	redacted, err := s.obfuscateInstance(ctx, instanceCreate.Metadata)
	if err != nil {
		return nil, err
	}
	metadataBytes, err := protojson.Marshal(redacted)
	if err != nil {
		return nil, err
	}
	var environment *string
	if instanceCreate.EnvironmentID != "" {
		environment = &instanceCreate.EnvironmentID
	}
	if _, err := tx.ExecContext(ctx, `
			INSERT INTO instance (
				resource_id,
				environment,
				metadata
			) VALUES ($1, $2, $3)
		`,
		instanceCreate.ResourceID,
		environment,
		metadataBytes,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	instance := &InstanceMessage{
		EnvironmentID: instanceCreate.EnvironmentID,
		ResourceID:    instanceCreate.ResourceID,
		Metadata:      instanceCreate.Metadata,
	}
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage) (*InstanceMessage, error) {
	set, args, where := []string{}, []any{}, []string{}
	if v := patch.EnvironmentID; v != nil {
		set, args = append(set, fmt.Sprintf("environment = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Deleted; v != nil {
		set, args = append(set, fmt.Sprintf(`deleted = $%d`, len(args)+1)), append(args, *v)
	}
	if v := patch.Metadata; v != nil {
		redacted, err := s.obfuscateInstance(ctx, v)
		if err != nil {
			return nil, err
		}
		metadata, err := protojson.Marshal(redacted)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("metadata = $%d", len(args)+1)), append(args, metadata)
	}
	where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, patch.ResourceID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if len(set) > 0 {
		query := fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE %s
		`, strings.Join(where, " AND "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.instanceCache.Remove(getInstanceCacheKey(patch.ResourceID))
	return s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &patch.ResourceID})
}

func (s *Store) listInstanceImplV2(ctx context.Context, txn *sql.Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	joinDSQuery := ""
	joinDBQuery := ""
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
		if hasHostPortFilter(filter.Where) {
			joinDSQuery = "CROSS JOIN jsonb_array_elements(instance.metadata -> 'dataSources') AS ds"
		}
		if strings.Contains(filter.Where, "db.project") {
			joinDBQuery = "LEFT JOIN db ON db.instance = instance.resource_id"
		}
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceIDs; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("instance.deleted = $%d", len(args)+1)), append(args, false)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (resource_id)
			instance.resource_id,
			instance.environment,
			instance.deleted,
			instance.metadata
		FROM instance
		%s
		%s
		WHERE %s
		ORDER BY resource_id`, joinDSQuery, joinDBQuery, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	var instanceMessages []*InstanceMessage
	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var environment sql.NullString
		var metadata []byte
		if err := rows.Scan(
			&instanceMessage.ResourceID,
			&environment,
			&instanceMessage.Deleted,
			&metadata,
		); err != nil {
			return nil, err
		}
		if environment.Valid {
			instanceMessage.EnvironmentID = environment.String
		}

		instanceMetadata := &storepb.Instance{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, instanceMetadata); err != nil {
			return nil, err
		}
		if err := s.unObfuscateInstance(ctx, instanceMetadata); err != nil {
			return nil, err
		}
		instanceMessage.Metadata = instanceMetadata
		instanceMessages = append(instanceMessages, &instanceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instanceMessages, nil
}

func hasHostPortFilter(where string) bool {
	return strings.Contains(where, "ds ->> 'host'") || strings.Contains(where, "ds ->> 'port'")
}

var countActivateInstanceQuery = "SELECT COUNT(1) FROM instance WHERE (metadata ? 'activation') AND (metadata->>'activation')::boolean = TRUE AND deleted = FALSE"

// GetActivatedInstanceCount gets the number of activated instances.
func (s *Store) GetActivatedInstanceCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, countActivateInstanceQuery).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func validateDataSources(metadata *storepb.Instance) error {
	dataSourceMap := map[string]bool{}
	adminCount := 0
	for _, dataSource := range metadata.GetDataSources() {
		if dataSourceMap[dataSource.GetId()] {
			return errors.Errorf("duplicate data source ID %s", dataSource.GetId())
		}
		dataSourceMap[dataSource.GetId()] = true
		if dataSource.GetType() == storepb.DataSourceType_ADMIN {
			adminCount++
		}
	}
	if adminCount != 1 {
		return errors.Errorf("require exactly one admin data source")
	}
	return nil
}

// IsObjectCaseSensitive returns true if the engine ignores database and table case sensitive.
func IsObjectCaseSensitive(instance *InstanceMessage) bool {
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_TIDB:
		return false
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return instance.Metadata == nil || instance.Metadata.MysqlLowerCaseTableNames == 0
	case storepb.Engine_MSSQL:
		// In fact, SQL Server is possible to create a case-sensitive database and case-insensitive database on one instance.
		// https://www.webucator.com/article/how-to-check-case-sensitivity-in-sql-server/
		// But by default, SQL Server is case-insensitive.
		return false
	default:
		return true
	}
}

func (s *Store) obfuscateInstance(ctx context.Context, instance *storepb.Instance) (*storepb.Instance, error) {
	secret, err := s.GetSecret(ctx)
	if err != nil {
		return nil, err
	}

	redacted := proto.CloneOf(instance)
	for _, ds := range redacted.GetDataSources() {
		ds.ObfuscatedPassword = common.Obfuscate(ds.GetPassword(), secret)
		ds.Password = ""
		ds.ObfuscatedSslCa = common.Obfuscate(ds.GetSslCa(), secret)
		ds.SslCa = ""
		ds.ObfuscatedSslCert = common.Obfuscate(ds.GetSslCert(), secret)
		ds.SslCert = ""
		ds.ObfuscatedSslKey = common.Obfuscate(ds.GetSslKey(), secret)
		ds.SslKey = ""
		ds.ObfuscatedSshPassword = common.Obfuscate(ds.GetSshPassword(), secret)
		ds.SshPassword = ""
		ds.ObfuscatedSshPrivateKey = common.Obfuscate(ds.GetSshPrivateKey(), secret)
		ds.SshPrivateKey = ""
		ds.ObfuscatedAuthenticationPrivateKey = common.Obfuscate(ds.GetAuthenticationPrivateKey(), secret)
		ds.AuthenticationPrivateKey = ""
		ds.ObfuscatedMasterPassword = common.Obfuscate(ds.GetMasterPassword(), secret)
		ds.MasterPassword = ""

		if azureCredential := ds.GetAzureCredential(); azureCredential != nil {
			azureCredential.ObfuscatedClientSecret = common.Obfuscate(azureCredential.ClientSecret, secret)
			azureCredential.ClientSecret = ""
		}
		if awsCredential := ds.GetAwsCredential(); awsCredential != nil {
			awsCredential.ObfuscatedAccessKeyId = common.Obfuscate(awsCredential.AccessKeyId, secret)
			awsCredential.AccessKeyId = ""

			awsCredential.ObfuscatedSecretAccessKey = common.Obfuscate(awsCredential.SecretAccessKey, secret)
			awsCredential.SecretAccessKey = ""

			awsCredential.ObfuscatedSessionToken = common.Obfuscate(awsCredential.SessionToken, secret)
			awsCredential.SessionToken = ""
		}
		if gcpCredential := ds.GetGcpCredential(); gcpCredential != nil {
			gcpCredential.ObfuscatedContent = common.Obfuscate(gcpCredential.Content, secret)
			gcpCredential.Content = ""
		}
	}
	return redacted, nil
}

func (s *Store) unObfuscateInstance(ctx context.Context, instance *storepb.Instance) error {
	secret, err := s.GetSecret(ctx)
	if err != nil {
		return err
	}

	for _, ds := range instance.GetDataSources() {
		password, err := common.Unobfuscate(ds.GetObfuscatedPassword(), secret)
		if err != nil {
			return err
		}
		ds.Password = password

		sslCa, err := common.Unobfuscate(ds.GetObfuscatedSslCa(), secret)
		if err != nil {
			return err
		}
		ds.SslCa = sslCa

		sslCert, err := common.Unobfuscate(ds.GetObfuscatedSslCert(), secret)
		if err != nil {
			return err
		}
		ds.SslCert = sslCert

		sslKey, err := common.Unobfuscate(ds.GetObfuscatedSslKey(), secret)
		if err != nil {
			return err
		}
		ds.SslKey = sslKey

		sshPassword, err := common.Unobfuscate(ds.GetObfuscatedSshPassword(), secret)
		if err != nil {
			return err
		}
		ds.SshPassword = sshPassword

		sshPrivateKey, err := common.Unobfuscate(ds.GetObfuscatedSshPrivateKey(), secret)
		if err != nil {
			return err
		}
		ds.SshPrivateKey = sshPrivateKey

		authenticationPrivateKey, err := common.Unobfuscate(ds.GetObfuscatedAuthenticationPrivateKey(), secret)
		if err != nil {
			return err
		}
		ds.AuthenticationPrivateKey = authenticationPrivateKey

		masterPassword, err := common.Unobfuscate(ds.GetObfuscatedMasterPassword(), secret)
		if err != nil {
			return err
		}
		ds.MasterPassword = masterPassword

		if azureCredential := ds.GetAzureCredential(); azureCredential != nil {
			clientSecret, err := common.Unobfuscate(azureCredential.ObfuscatedClientSecret, secret)
			if err != nil {
				return err
			}
			ds.GetAzureCredential().ClientSecret = clientSecret
		}

		if awsCredential := ds.GetAwsCredential(); awsCredential != nil {
			accessKeyID, err := common.Unobfuscate(awsCredential.ObfuscatedAccessKeyId, secret)
			if err != nil {
				return err
			}
			awsCredential.AccessKeyId = accessKeyID

			secretAccessKey, err := common.Unobfuscate(awsCredential.ObfuscatedSecretAccessKey, secret)
			if err != nil {
				return err
			}
			awsCredential.SecretAccessKey = secretAccessKey

			sessionToken, err := common.Unobfuscate(awsCredential.ObfuscatedSessionToken, secret)
			if err != nil {
				return err
			}
			awsCredential.SessionToken = sessionToken
		}

		if gcpCredential := ds.GetGcpCredential(); gcpCredential != nil {
			content, err := common.Unobfuscate(gcpCredential.ObfuscatedContent, secret)
			if err != nil {
				return err
			}
			gcpCredential.Content = content
		}
	}
	return nil
}
