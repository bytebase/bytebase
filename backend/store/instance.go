package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
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

// FindInstance finds a list of Instance instances.
func (s *Store) FindInstance(ctx context.Context, find *api.InstanceFind) ([]*api.Instance, error) {
	v2Find := &FindInstanceMessage{ShowDeleted: true}
	instances, err := s.ListInstancesV2(ctx, v2Find)
	if err != nil {
		return nil, err
	}

	var composedInstances []*api.Instance
	for _, instance := range instances {
		composedInstance, err := s.composeInstance(ctx, instance)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Instance with instance[%+v]", instance)
		}
		if find.RowStatus != nil && composedInstance.RowStatus != *find.RowStatus {
			continue
		}
		composedInstances = append(composedInstances, composedInstance)
	}
	return composedInstances, nil
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
			DatabaseID: ds.DatabaseID,
			Name:       ds.Title,
			Type:       ds.Type,
			Username:   ds.Username,
			Host:       ds.Host,
			Port:       ds.Port,
			Options:    api.DataSourceOptions{SRV: ds.SRV, AuthenticationDatabase: ds.AuthenticationDatabase, SID: ds.SID, ServiceName: ds.ServiceName},
			Database:   ds.Database,
		})
		if ds.Type == api.Admin {
			composedInstance.Host = ds.Host
			composedInstance.Port = ds.Port
		}
	}

	return composedInstance, nil
}

// InstanceMessage is the mssage for instance.
type InstanceMessage struct {
	ResourceID   string
	Title        string
	Engine       db.Type
	ExternalLink string
	DataSources  []*DataSourceMessage
	// Output only.
	EnvironmentID string
	UID           int
	Deleted       bool
	EngineVersion string
}

// UpdateInstanceMessage is the mssage for updating an instance.
type UpdateInstanceMessage struct {
	Title         *string
	ExternalLink  *string
	Delete        *bool
	DataSources   *[]*DataSourceMessage
	EngineVersion *string

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
	if find.EnvironmentID != nil && find.ResourceID != nil {
		if instance, ok := s.instanceCache.Load(getInstanceCacheKey(*find.EnvironmentID, *find.ResourceID)); ok {
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
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	instance := instances[0]
	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// ListInstancesV2 lists all instance.
func (s *Store) ListInstancesV2(ctx context.Context, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	instances, err := s.listInstanceImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, instance := range instances {
		s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
		s.instanceIDCache.Store(instance.UID, instance)
	}
	return instances, nil
}

// CreateInstanceV2 creates the instance.
func (s *Store) CreateInstanceV2(ctx context.Context, environmentID string, instanceCreate *InstanceMessage, creatorID int) (*InstanceMessage, error) {
	if err := validateDataSourceList(instanceCreate.DataSources); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	// TODO(d): use the same query for environment.
	environment, err := s.getEnvironmentImplV2(ctx, tx, &FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %s not found", environmentID)
	}

	var instanceID int
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO instance (
				resource_id,
				creator_id,
				updater_id,
				environment_id,
				name,
				engine,
				external_link
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`,
		instanceCreate.ResourceID,
		creatorID,
		creatorID,
		environment.UID,
		instanceCreate.Title,
		instanceCreate.Engine,
		instanceCreate.ExternalLink,
	).Scan(&instanceID); err != nil {
		return nil, FormatError(err)
	}

	allDatabaseUID, err := s.createDatabaseDefaultImpl(ctx, tx, instanceID, &DatabaseMessage{DatabaseName: api.AllDatabaseName})
	if err != nil {
		return nil, err
	}

	for _, ds := range instanceCreate.DataSources {
		if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceID, allDatabaseUID, creatorID, ds); err != nil {
			return nil, err
		}
	}

	dataSources, err := s.listDataSourceV2(ctx, tx, instanceCreate.ResourceID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	instance := &InstanceMessage{
		EnvironmentID: environmentID,
		ResourceID:    instanceCreate.ResourceID,
		UID:           instanceID,
		Title:         instanceCreate.Title,
		Engine:        instanceCreate.Engine,
		ExternalLink:  instanceCreate.ExternalLink,
		DataSources:   dataSources,
	}
	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage) (*InstanceMessage, error) {
	if patch.DataSources != nil {
		if err := validateDataSourceList(*patch.DataSources); err != nil {
			return nil, err
		}
	}

	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExternalLink; v != nil {
		set, args = append(set, fmt.Sprintf("external_link = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EngineVersion; v != nil {
		set, args = append(set, fmt.Sprintf("engine_version = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
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

	instance := &InstanceMessage{
		EnvironmentID: patch.EnvironmentID,
	}
	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE resource_id = $%d AND environment_id = $%d
			RETURNING
				id,
				resource_id,
				name,
				engine,
				external_link,
				row_status
		`, len(args)-1, len(args)),
		args...,
	).Scan(
		&instance.UID,
		&instance.ResourceID,
		&instance.Title,
		&instance.Engine,
		&instance.ExternalLink,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}

	if patch.DataSources != nil {
		allDatabaseName := api.AllDatabaseName
		allDatabase, err := s.getDatabaseImplV2(ctx, tx, &FindDatabaseMessage{
			InstanceID:         &instance.ResourceID,
			DatabaseName:       &allDatabaseName,
			IncludeAllDatabase: true,
		})
		if err != nil {
			return nil, err
		}

		if err := s.clearDataSourceImpl(ctx, tx, instance.UID, allDatabase.UID); err != nil {
			return nil, err
		}

		for _, ds := range *patch.DataSources {
			if err := s.addDataSourceToInstanceImplV2(ctx, tx, instance.UID, allDatabase.UID, patch.UpdaterID, ds); err != nil {
				return nil, err
			}
		}
	}
	instance.Deleted = convertRowStatusToDeleted(rowStatus)
	dataSourceList, err := s.listDataSourceV2(ctx, tx, patch.ResourceID)
	if err != nil {
		return nil, FormatError(err)
	}
	instance.DataSources = dataSourceList

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.instanceCache.Store(getInstanceCacheKey(instance.EnvironmentID, instance.ResourceID), instance)
	s.instanceIDCache.Store(instance.UID, instance)
	return instance, nil
}

func (s *Store) listInstanceImplV2(ctx context.Context, tx *Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
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
			instance.row_status AS row_status
		FROM instance
		LEFT JOIN environment ON environment.id = instance.environment_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var rowStatus string
		if err := rows.Scan(
			&instanceMessage.EnvironmentID,
			&instanceMessage.UID,
			&instanceMessage.ResourceID,
			&instanceMessage.Title,
			&instanceMessage.Engine,
			&instanceMessage.EngineVersion,
			&instanceMessage.ExternalLink,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		instanceMessages = append(instanceMessages, &instanceMessage)
	}

	for _, instanceMessage := range instanceMessages {
		dataSourceList, err := s.listDataSourceV2(ctx, tx, instanceMessage.ResourceID)
		if err != nil {
			return nil, FormatError(err)
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
		return nil, FormatError(err)
	}
	defer rows.Close()
	var instanceUIDs []int
	for rows.Next() {
		var instanceUID int
		if err := rows.Scan(
			&instanceUID,
		); err != nil {
			return nil, FormatError(err)
		}
		instanceUIDs = append(instanceUIDs, instanceUID)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
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
