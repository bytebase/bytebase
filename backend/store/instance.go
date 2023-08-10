package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetInstanceByID gets an instance of Instance.
func (s *Store) GetInstanceByID(ctx context.Context, id int) (*api.Instance, error) {
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{UID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Instance with ID %d", id)
	}
	if instance == nil {
		return nil, nil
	}
	composedInstance, err := s.composeInstance(ctx, instance)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Instance with instance[%+v]", instance)
	}
	return composedInstance, nil
}

// private function.
func (s *Store) composeInstance(ctx context.Context, instance *InstanceMessage) (*api.Instance, error) {
	composedInstance := &api.Instance{
		ID:            instance.UID,
		ResourceID:    instance.ResourceID,
		RowStatus:     api.Normal,
		Name:          instance.Title,
		Engine:        instance.Engine,
		EngineVersion: instance.EngineVersion,
		ExternalLink:  instance.ExternalLink,
	}
	if instance.Deleted {
		composedInstance.RowStatus = api.Archived
	}

	environment, err := s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, err
	}
	composedInstance.EnvironmentID = environment.UID
	composedEnvironment, err := s.GetEnvironmentByID(ctx, environment.UID)
	if err != nil {
		return nil, err
	}
	composedInstance.Environment = composedEnvironment

	for _, ds := range instance.DataSources {
		composedInstance.DataSourceList = append(composedInstance.DataSourceList, &api.DataSource{
			ID:         ds.UID,
			InstanceID: instance.UID,
			Name:       ds.ID,
			Type:       ds.Type,
			Username:   ds.Username,
			Host:       ds.Host,
			Port:       ds.Port,
			Options: api.DataSourceOptions{
				SRV:                    ds.SRV,
				AuthenticationDatabase: ds.AuthenticationDatabase,
				SID:                    ds.SID,
				ServiceName:            ds.ServiceName,
				SSHHost:                ds.SSHHost,
				SSHPort:                ds.SSHPort,
				SSHUser:                ds.SSHUser,
			},
			Database: ds.Database,
		})
		if ds.Type == api.Admin {
			composedInstance.Host = ds.Host
			composedInstance.Port = ds.Port
		}
	}

	return composedInstance, nil
}

// InstanceMessage is the message for instance.
type InstanceMessage struct {
	ResourceID   string
	Title        string
	Engine       db.Type
	ExternalLink string
	DataSources  []*DataSourceMessage
	Activation   bool
	Options      *storepb.InstanceOptions
	// Output only.
	EnvironmentID string
	UID           int
	Deleted       bool
	EngineVersion string
	Metadata      *storepb.InstanceMetadata
}

// UpdateInstanceMessage is the message for updating an instance.
type UpdateInstanceMessage struct {
	Title         *string
	ExternalLink  *string
	Delete        *bool
	DataSources   *[]*DataSourceMessage
	EngineVersion *string
	Activation    *bool
	Options       *storepb.InstanceOptions
	Metadata      *storepb.InstanceMetadata

	// Output only.
	UpdaterID     int
	EnvironmentID string
	ResourceID    string
}

// FindInstanceMessage is the message for finding instances.
type FindInstanceMessage struct {
	UID           *int
	EnvironmentID *string
	ResourceID    *string
	ShowDeleted   bool
}

// GetInstanceV2 gets an instance by the resource_id.
func (s *Store) GetInstanceV2(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.ResourceID != nil {
		if instance, ok := s.instanceCache.Load(getInstanceCacheKey(*find.ResourceID)); ok {
			return instance.(*InstanceMessage), nil
		}
	}
	if find.UID != nil {
		if instance, ok := s.instanceIDCache.Load(*find.UID); ok {
			return instance.(*InstanceMessage), nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list instances with find instance message %+v", find)
	}
	if len(instances) == 0 {
		return nil, nil
	}
	if len(instances) > 1 {
		return nil, errors.Errorf("find %d instances with find instance message %+v, expected 1", len(instances), find)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	instance := instances[0]
	s.instanceCache.Store(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// ListInstancesV2 lists all instance.
func (s *Store) ListInstancesV2(ctx context.Context, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
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
		s.instanceCache.Store(getInstanceCacheKey(instance.ResourceID), instance)
		s.instanceIDCache.Store(instance.UID, instance)
	}
	return instances, nil
}

// CreateInstanceV2 creates the instance.
func (s *Store) CreateInstanceV2(ctx context.Context, instanceCreate *InstanceMessage, creatorID, maximumActivation int) (*InstanceMessage, error) {
	if err := validateDataSourceList(instanceCreate.DataSources); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO(d): use the same query for environment.
	environment, err := s.getEnvironmentImplV2(ctx, tx, &FindEnvironmentMessage{
		ResourceID: &instanceCreate.EnvironmentID,
	})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %s not found", instanceCreate.EnvironmentID)
	}

	where := ""
	if instanceCreate.Activation {
		where = fmt.Sprintf("WHERE (%s) < %d", countActivateInstanceQuery, maximumActivation)
	}

	optionBytes, err := protojson.Marshal(instanceCreate.Options)
	if err != nil {
		return nil, err
	}

	metadataBytes, err := protojson.Marshal(instanceCreate.Metadata)
	if err != nil {
		return nil, err
	}

	var instanceID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			INSERT INTO instance (
				resource_id,
				creator_id,
				updater_id,
				environment_id,
				name,
				engine,
				external_link,
				activation,
				options,
				metadata
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
			%s
			RETURNING id
		`, where),
		instanceCreate.ResourceID,
		creatorID,
		creatorID,
		environment.UID,
		instanceCreate.Title,
		instanceCreate.Engine,
		instanceCreate.ExternalLink,
		instanceCreate.Activation,
		optionBytes,
		metadataBytes,
	).Scan(&instanceID); err != nil {
		return nil, err
	}

	for _, ds := range instanceCreate.DataSources {
		if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceID, creatorID, ds); err != nil {
			return nil, err
		}
	}

	dataSources, err := s.listDataSourceV2(ctx, tx, instanceCreate.ResourceID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	instance := &InstanceMessage{
		EnvironmentID: instanceCreate.EnvironmentID,
		ResourceID:    instanceCreate.ResourceID,
		UID:           instanceID,
		Title:         instanceCreate.Title,
		Engine:        instanceCreate.Engine,
		ExternalLink:  instanceCreate.ExternalLink,
		DataSources:   dataSources,
		Activation:    instanceCreate.Activation,
		Options:       instanceCreate.Options,
		Metadata:      instanceCreate.Metadata,
	}
	s.instanceCache.Store(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage, maximumActivation int) (*InstanceMessage, error) {
	if patch.DataSources != nil {
		if err := validateDataSourceList(*patch.DataSources); err != nil {
			return nil, err
		}
	}

	set, args, where := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", patch.UpdaterID)}, []string{}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, fmt.Sprintf("external_link = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EngineVersion; v != nil {
		set, args = append(set, fmt.Sprintf("engine_version = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Activation; v != nil {
		set, args = append(set, fmt.Sprintf("activation = $%d", len(args)+1)), append(args, *v)
		if *v {
			where = append(where, fmt.Sprintf("(%s) < %d", countActivateInstanceQuery, maximumActivation))
		}
	}
	if v := patch.Delete; v != nil {
		if patch.Activation != nil {
			return nil, errors.Errorf(`cannot set "activation" and "row_status" at the same time`)
		}
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
			set, args = append(set, fmt.Sprintf("activation = $%d", len(args)+1)), append(args, false)
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	if v := patch.Options; v != nil {
		set, args = append(set, fmt.Sprintf("options = $%d", len(args)+1)), append(args, v)
	}
	if v := patch.Metadata; v != nil {
		set, args = append(set, fmt.Sprintf("metadata = $%d", len(args)+1)), append(args, v)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO(d): use the same query for environment.
	environment, err := s.getEnvironmentImplV2(ctx, tx, &FindEnvironmentMessage{
		ResourceID: &patch.EnvironmentID,
	})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %s not found", patch.EnvironmentID)
	}

	args = append(args, patch.ResourceID, environment.UID)
	where = append(where, fmt.Sprintf("resource_id = $%d", len(args)-1), fmt.Sprintf("environment_id = $%d", len(args)))

	instance := &InstanceMessage{
		EnvironmentID: patch.EnvironmentID,
	}
	query := fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE %s
			RETURNING
				id,
				resource_id,
				name,
				engine,
				engine_version,
				external_link,
				activation,
				row_status,
				options,
				metadata
		`, strings.Join(where, " AND "))
	var rowStatus string
	var options, metadata []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&instance.UID,
		&instance.ResourceID,
		&instance.Title,
		&instance.Engine,
		&instance.EngineVersion,
		&instance.ExternalLink,
		&instance.Activation,
		&rowStatus,
		&options,
		&metadata,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if patch.DataSources != nil {
		if err := s.clearDataSourceImpl(ctx, tx, instance.UID); err != nil {
			return nil, err
		}

		for _, ds := range *patch.DataSources {
			if err := s.addDataSourceToInstanceImplV2(ctx, tx, instance.UID, patch.UpdaterID, ds); err != nil {
				return nil, err
			}
		}
	}
	instance.Deleted = convertRowStatusToDeleted(rowStatus)
	dataSourceList, err := s.listDataSourceV2(ctx, tx, patch.ResourceID)
	if err != nil {
		return nil, err
	}
	instance.DataSources = dataSourceList
	var instanceOptions storepb.InstanceOptions
	if err := protojson.Unmarshal(options, &instanceOptions); err != nil {
		return nil, err
	}
	instance.Options = &instanceOptions

	var instanceMetadata storepb.InstanceMetadata
	if err := protojson.Unmarshal(metadata, &instanceMetadata); err != nil {
		return nil, err
	}
	instance.Metadata = &instanceMetadata

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

func (s *Store) listInstanceImplV2(ctx context.Context, tx *Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.EnvironmentID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("instance.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var instanceMessages []*InstanceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			environment.resource_id as environment_id,
			instance.id AS instance_uid,
			instance.resource_id AS resource_id,
			instance.name AS name,
			engine,
			engine_version,
			external_link,
			activation,
			instance.row_status AS row_status,
			instance.options AS options,
			instance.metadata AS metadata
		FROM instance
		LEFT JOIN environment ON environment.id = instance.environment_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var rowStatus string
		var options, metadata []byte
		if err := rows.Scan(
			&instanceMessage.EnvironmentID,
			&instanceMessage.UID,
			&instanceMessage.ResourceID,
			&instanceMessage.Title,
			&instanceMessage.Engine,
			&instanceMessage.EngineVersion,
			&instanceMessage.ExternalLink,
			&instanceMessage.Activation,
			&rowStatus,
			&options,
			&metadata,
		); err != nil {
			return nil, err
		}
		instanceMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		var instanceOptions storepb.InstanceOptions
		if err := protojson.Unmarshal(options, &instanceOptions); err != nil {
			return nil, err
		}
		instanceMessage.Options = &instanceOptions
		var instanceMetadata storepb.InstanceMetadata
		if err := protojson.Unmarshal(metadata, &instanceMetadata); err != nil {
			return nil, err
		}
		instanceMessage.Metadata = &instanceMetadata
		instanceMessages = append(instanceMessages, &instanceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, instanceMessage := range instanceMessages {
		dataSourceList, err := s.listDataSourceV2(ctx, tx, instanceMessage.ResourceID)
		if err != nil {
			return nil, err
		}
		instanceMessage.DataSources = dataSourceList
	}

	return instanceMessages, nil
}

// FindInstanceWithDatabaseBackupEnabled finds instances with at least one database who enables backup policy.
func (s *Store) FindInstanceWithDatabaseBackupEnabled(ctx context.Context) ([]*InstanceMessage, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT DISTINCT
			instance.id
		FROM instance
		JOIN db ON db.instance_id = instance.id
		JOIN backup_setting AS bs ON db.id = bs.database_id
		WHERE bs.enabled = true AND instance.row_status = $1
	`, api.Normal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var instanceUIDs []int
	for rows.Next() {
		var instanceUID int
		if err := rows.Scan(
			&instanceUID,
		); err != nil {
			return nil, err
		}
		instanceUIDs = append(instanceUIDs, instanceUID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var instances []*InstanceMessage
	for _, instanceUID := range instanceUIDs {
		instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{UID: &instanceUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %v", instanceUID)
		}
		if instance == nil {
			continue
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

var countActivateInstanceQuery = "SELECT COUNT(*) FROM instance WHERE activation = TRUE AND row_status = 'NORMAL'"

// CheckActivationLimit checks if activation instance count reaches the limit.
func (s *Store) CheckActivationLimit(ctx context.Context, maximumActivation int) error {
	count := 0
	if err := s.db.db.QueryRowContext(ctx, countActivateInstanceQuery).Scan(&count); err != nil {
		return err
	}

	if count >= maximumActivation {
		return common.Errorf(common.Invalid, "activation instance count reaches the limitation")
	}
	return nil
}

func validateDataSourceList(dataSources []*DataSourceMessage) error {
	dataSourceMap := map[api.DataSourceType]bool{}
	for _, dataSource := range dataSources {
		if dataSourceMap[dataSource.Type] {
			return status.Errorf(codes.InvalidArgument, "duplicate data source type %s", dataSource.Type)
		}
		dataSourceMap[dataSource.Type] = true
	}
	if !dataSourceMap[api.Admin] {
		return status.Errorf(codes.InvalidArgument, "missing required data source type %s", api.Admin)
	}
	return nil
}

// IgnoreDatabaseAndTableCaseSensitive returns true if the engine ignores database and table case sensitive.
func IgnoreDatabaseAndTableCaseSensitive(instance *InstanceMessage) bool {
	switch instance.Engine {
	case db.TiDB:
		return true
	case db.MySQL, db.MariaDB:
		return instance.Metadata != nil && instance.Metadata.MysqlLowerCaseTableNames != 0
	case db.MSSQL:
		// In fact, SQL Server is possible to create a case-sensitive database and case-insensitive database on one instance.
		// https://www.webucator.com/article/how-to-check-case-sensitivity-in-sql-server/
		// But by default, SQL Server is case-insensitive.
		return true
	default:
		return false
	}
}
