package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// InstanceMessage is the message for instance.
type InstanceMessage struct {
	ResourceID    string
	EnvironmentID *string
	Deleted       bool
	Metadata      *storepb.Instance
}

// UpdateInstanceMessage is the message for updating an instance.
type UpdateInstanceMessage struct {
	// allow batch update
	ResourceID          *string
	FindByEnvironmentID *string

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
	FilterQ     *qb.Query
	OrderByKeys []*OrderByKey
}

// GetInstance gets an instance by the resource_id.
func (s *Store) GetInstance(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.instanceCache.Get(getInstanceCacheKey(*find.ResourceID)); ok && s.enableCache {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	instances, err := s.ListInstances(ctx, find)
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

// ListInstances lists all instance.
func (s *Store) ListInstances(ctx context.Context, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImpl(ctx, tx, find)
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

// CreateInstance creates an instance.
func (s *Store) CreateInstance(ctx context.Context, instanceCreate *InstanceMessage) (*InstanceMessage, error) {
	if err := validateDataSources(instanceCreate.Metadata); err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
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
	if instanceCreate.EnvironmentID != nil && *instanceCreate.EnvironmentID != "" {
		environment = instanceCreate.EnvironmentID
	}
	q := qb.Q().Space(`
			INSERT INTO instance (
				resource_id,
				environment,
				metadata
			) VALUES (?, ?, ?)
		`, instanceCreate.ResourceID, environment, metadataBytes)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
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

// UpdateInstance updates an instance.
func (s *Store) UpdateInstance(ctx context.Context, patch *UpdateInstanceMessage) (*InstanceMessage, error) {
	set := qb.Q()

	if v := patch.EnvironmentID; v != nil {
		if *v == "" {
			// Unset the environment by setting it to NULL
			set.Comma("environment = ?", nil)
		} else {
			set.Comma("environment = ?", *v)
		}
	}
	if v := patch.Deleted; v != nil {
		set.Comma("deleted = ?", *v)
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
		set.Comma("metadata = ?", metadata)
	}

	if set.Len() == 0 {
		return nil, errors.New("no update field specified")
	}

	where := qb.Q()
	if v := patch.ResourceID; v != nil {
		where.And("resource_id = ?", *v)
	}
	if v := patch.FindByEnvironmentID; v != nil {
		where.And("environment = ?", *v)
	}

	if where.Len() == 0 {
		return nil, errors.Errorf("empty where")
	}

	q := qb.Q().Space("UPDATE instance SET ?", set).
		Space("WHERE ?", where)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if v := patch.ResourceID; v != nil {
		s.instanceCache.Remove(getInstanceCacheKey(*v))
		return s.GetInstance(ctx, &FindInstanceMessage{ResourceID: v})
	}

	return nil, nil
}

func (s *Store) listInstanceImpl(ctx context.Context, txn *sql.Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where := qb.Q().Space("TRUE")
	if filterQ := find.FilterQ; filterQ != nil {
		where.And("?", filterQ)
	}
	if v := find.ResourceID; v != nil {
		where.And("instance.resource_id = ?", *v)
	}
	if v := find.ResourceIDs; v != nil {
		where.And("instance.resource_id = ANY(?)", *v)
	}
	if !find.ShowDeleted {
		where.And("instance.deleted = ?", false)
	}

	q := qb.Q().Space(`
		SELECT
			instance.resource_id,
			instance.environment,
			instance.deleted,
			instance.metadata
		FROM instance
		WHERE ?
	`, where)

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY resource_id ASC")
	}

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, queryArgs, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var instanceMessages []*InstanceMessage
	rows, err := txn.QueryContext(ctx, query, queryArgs...)
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
			instanceMessage.EnvironmentID = &environment.String
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

// GetActivatedInstanceCount gets the number of activated instances.
func (s *Store) GetActivatedInstanceCount(ctx context.Context) (int, error) {
	q := qb.Q().Space("SELECT COUNT(1) FROM instance WHERE (metadata ?? 'activation') AND (metadata->>'activation')::boolean = TRUE AND deleted = FALSE")
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}
	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
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
	systemSetting, err := s.GetSystemSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get system setting")
	}
	secret := systemSetting.AuthSecret

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
		ds.ObfuscatedAuthenticationPrivateKeyPassphrase = common.Obfuscate(ds.GetAuthenticationPrivateKeyPassphrase(), secret)
		ds.AuthenticationPrivateKeyPassphrase = ""
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
		if externalSecret := ds.GetExternalSecret(); externalSecret != nil {
			externalSecret.ObfuscatedVaultSslCa = common.Obfuscate(externalSecret.GetVaultSslCa(), secret)
			externalSecret.VaultSslCa = ""
			externalSecret.ObfuscatedVaultSslCert = common.Obfuscate(externalSecret.GetVaultSslCert(), secret)
			externalSecret.VaultSslCert = ""
			externalSecret.ObfuscatedVaultSslKey = common.Obfuscate(externalSecret.GetVaultSslKey(), secret)
			externalSecret.VaultSslKey = ""
		}
	}
	return redacted, nil
}

func (s *Store) unObfuscateInstance(ctx context.Context, instance *storepb.Instance) error {
	systemSetting, err := s.GetSystemSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get system setting")
	}
	secret := systemSetting.AuthSecret

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

		authenticationPrivateKeyPassphrase, err := common.Unobfuscate(ds.GetObfuscatedAuthenticationPrivateKeyPassphrase(), secret)
		if err != nil {
			return err
		}
		ds.AuthenticationPrivateKeyPassphrase = authenticationPrivateKeyPassphrase

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

		if externalSecret := ds.GetExternalSecret(); externalSecret != nil {
			sslCa, err := common.Unobfuscate(externalSecret.GetObfuscatedVaultSslCa(), secret)
			if err != nil {
				return err
			}
			externalSecret.VaultSslCa = sslCa

			sslCert, err := common.Unobfuscate(externalSecret.GetObfuscatedVaultSslCert(), secret)
			if err != nil {
				return err
			}
			externalSecret.VaultSslCert = sslCert

			sslKey, err := common.Unobfuscate(externalSecret.GetObfuscatedVaultSslKey(), secret)
			if err != nil {
				return err
			}
			externalSecret.VaultSslKey = sslKey
		}
	}
	return nil
}

// HasSampleInstances checks if there are sample instances in the database.
func (s *Store) HasSampleInstances(ctx context.Context) (bool, error) {
	instances, err := s.ListInstances(ctx, &FindInstanceMessage{
		ResourceIDs: &[]string{"test-sample-instance", "prod-sample-instance"},
		ShowDeleted: false,
	})
	if err != nil {
		return false, err
	}
	return len(instances) > 0, nil
}

// DeleteInstance permanently purges a soft-deleted instance and all related resources.
// This operation is irreversible and should only be used for:
// - Administrative cleanup of old soft-deleted instances
// - Test cleanup
// Following AIP-164/165, this only works on instances where deleted = TRUE.
func (s *Store) DeleteInstance(ctx context.Context, resourceID string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Delete query_history entries that reference this instance
	// The database field contains instance reference like 'instances/{instance}/databases/{database}'
	q := qb.Q().Space(`
		DELETE FROM query_history
		WHERE database LIKE 'instances/' || ? || '/databases/%'
	`, resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete query_history for instance %s", resourceID)
	}

	// Delete task_run_log entries for tasks associated with this instance
	q = qb.Q().Space(`
		DELETE FROM task_run_log
		WHERE task_run_id IN (
			SELECT tr.id FROM task_run tr
			JOIN task t ON tr.task_id = t.id
			WHERE t.instance = ?
		)
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run_log for instance %s", resourceID)
	}

	// Delete task_run entries for tasks associated with this instance
	q = qb.Q().Space(`
		DELETE FROM task_run
		WHERE task_id IN (
			SELECT id FROM task WHERE instance = ?
		)
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run for instance %s", resourceID)
	}

	// Delete tasks associated with this instance
	q = qb.Q().Space(`
		DELETE FROM task WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete tasks for instance %s", resourceID)
	}

	// Delete changelogs associated with this instance
	q = qb.Q().Space(`
		DELETE FROM changelog WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete changelogs for instance %s", resourceID)
	}

	// Delete sync_history associated with this instance
	q = qb.Q().Space(`
		DELETE FROM sync_history WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete sync_history for instance %s", resourceID)
	}

	// Delete revisions associated with this instance
	q = qb.Q().Space(`
		DELETE FROM revision WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete revisions for instance %s", resourceID)
	}

	// Update worksheets to nullify instance and db_name references
	q = qb.Q().Space(`
		UPDATE worksheet
		SET instance = NULL, db_name = NULL
		WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update worksheets for instance %s", resourceID)
	}

	// Delete db_schema entries associated with databases on this instance
	q = qb.Q().Space(`
		DELETE FROM db_schema WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete db_schema for instance %s", resourceID)
	}

	// Delete databases associated with this instance
	q = qb.Q().Space(`
		DELETE FROM db WHERE instance = ?
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to delete databases for instance %s", resourceID)
	}

	// Finally, delete the instance itself (only if it's marked as deleted)
	q = qb.Q().Space(`
		DELETE FROM instance
		WHERE resource_id = ? AND deleted = TRUE
	`, resourceID)
	query, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to delete instance %s", resourceID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rowsAffected == 0 {
		return errors.Errorf("instance %s not found or not marked as deleted", resourceID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	// Clear the instance from cache
	s.instanceCache.Remove(getInstanceCacheKey(resourceID))

	return nil
}

func GetListInstanceFilter(filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Errorf("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	parseToLabelFilterSQL := func(resource, key string, value any) (*qb.Query, error) {
		switch v := value.(type) {
		case string:
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ?", resource, key), v), nil
		case []any:
			if len(v) == 0 {
				return nil, errors.Errorf("empty label filter")
			}

			labelValueList := make([]any, len(v))
			for i, raw := range v {
				str, ok := raw.(string)
				if !ok {
					return nil, errors.Errorf("label value must be string, got %T", raw)
				}
				labelValueList[i] = str
			}
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ANY(?)", resource, key), labelValueList), nil
		default:
			return nil, errors.Errorf("empty value %v for label filter", value)
		}
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		// Handle label filters like "labels.org_group"
		if varStr, ok := variable.(string); ok {
			if labelKey, ok := strings.CutPrefix(varStr, "labels."); ok {
				return parseToLabelFilterSQL("instance.metadata", labelKey, value)
			}
		}

		switch variable {
		case "name":
			return qb.Q().Space("instance.metadata->>'title' = ?", value.(string)), nil
		case "resource_id":
			return qb.Q().Space("instance.resource_id = ?", value.(string)), nil
		case "environment":
			environment, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("failed to parse value %v to string", value)
			}
			if environment != "" {
				environmentID, err := common.GetEnvironmentID(environment)
				if err != nil {
					return nil, errors.Errorf("invalid environment filter %q", value)
				}
				return qb.Q().Space("instance.environment = ?", environmentID), nil
			}
			return qb.Q().Space("instance.environment IS NULL"), nil
		case "state":
			v1State, ok := v1pb.State_value[value.(string)]
			if !ok {
				return nil, errors.Errorf("invalid state filter %q", value)
			}
			return qb.Q().Space("instance.deleted = ?", v1pb.State(v1State) == v1pb.State_DELETED), nil
		case "engine":
			v1Engine, ok := v1pb.Engine_value[value.(string)]
			if !ok {
				return nil, errors.Errorf("invalid engine filter %q", value)
			}
			engine := convertEngine(v1pb.Engine(v1Engine))
			return qb.Q().Space("instance.metadata->>'engine' = ?", engine), nil
		case "host":
			return qb.Q().Space(
				`instance.metadata -> 'dataSources' @> jsonb_build_array(jsonb_build_object('host', ?))`,
				value.(string)), nil
		case "port":
			return qb.Q().Space(
				`instance.metadata -> 'dataSources' @> jsonb_build_array(jsonb_build_object('port', ?))`,
				value.(string)), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			return qb.Q().Space(
				`EXISTS (SELECT 1 FROM db WHERE db.instance = instance.resource_id AND db.project = ?)`,
				projectID), nil
		default:
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	parseToEngineSQL := func(expr celast.Expr) (*qb.Query, error) {
		variable, value := getVariableAndValueFromExpr(expr)
		if variable != "engine" {
			return nil, errors.Errorf(`only "engine" support "engine in [xx]"/"!(engine in [xx])" operator`)
		}
		if value == nil {
			return nil, errors.Errorf(`empty value %v for "engine" operator`, value)
		}
		list, ok := value.([]any)
		if !ok {
			return nil, errors.Errorf(`expect list, got %T, hint: filter engine in ["xx"]`, value)
		}
		if len(list) == 0 {
			return nil, errors.Errorf(`empty value %v for "engine" operator`, value)
		}

		engineList := make([]any, len(list))
		for i, raw := range list {
			engine, ok := raw.(string)
			if !ok {
				return nil, errors.Errorf(`expect string, got %T for engine %v`, raw, raw)
			}
			v1Engine, ok := v1pb.Engine_value[engine]
			if !ok {
				return nil, errors.Errorf(`invalid engine filter %q`, engine)
			}
			storeEngine := convertEngine(v1pb.Engine(v1Engine))
			engineList[i] = storeEngine
		}
		return qb.Q().Space("instance.metadata->>'engine' = ANY(?)", engineList), nil
	}

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.Or("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.And("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				if strValue == "" {
					return nil, errors.Errorf(`empty value for %q`, variable)
				}

				switch variable {
				case "name":
					return qb.Q().Space("LOWER(instance.metadata->>'title') LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				case "resource_id":
					return qb.Q().Space("LOWER(instance.resource_id) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				case "host", "port":
					return qb.Q().Space(fmt.Sprintf("ds ->> '%s' LIKE ?", variable), "%"+strValue+"%"), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable == "engine" {
					return parseToEngineSQL(expr)
				} else if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return parseToLabelFilterSQL("instance.metadata", labelKey, value)
				}
				return nil, errors.Errorf("unsupport variable %q", variable)
			case celoperators.LogicalNot:
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`only support !(engine in ["{engine1}", "{engine2}"]) format`)
				}
				qq, err := getFilter(args[0])
				if err != nil {
					return nil, err
				}
				return qb.Q().Space("(NOT (?))", qq), nil
			default:
				return nil, errors.Errorf("unexpected function %v", functionName)
			}
		default:
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	q, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", q), nil
}

func convertEngine(engine v1pb.Engine) string {
	return storepb.Engine_name[int32(engine)]
}

func GetInstanceOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	orderByKeys := []*OrderByKey{}
	for _, orderByKey := range keys {
		switch orderByKey.Key {
		case "title":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "instance.metadata->>'title'",
				SortOrder: orderByKey.SortOrder,
			})
		case "environment":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "instance.environment",
				SortOrder: orderByKey.SortOrder,
			})
		default:
			return nil, errors.Errorf(`invalid order key "%v"`, orderByKey.Key)
		}
	}

	return orderByKeys, nil
}
