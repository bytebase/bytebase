package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// InstanceMessage is the message for instance.
type InstanceMessage struct {
	ResourceID   string
	Title        string
	Engine       storepb.Engine
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
	// OptionsUpsert upserts the top-level messages of the instance options.
	OptionsUpsert       *storepb.InstanceOptions
	Metadata            *storepb.InstanceMetadata
	UpdateEnvironmentID bool
	EnvironmentID       string

	// Output only.
	UpdaterID  int
	ResourceID string
}

// FindInstanceMessage is the message for finding instances.
type FindInstanceMessage struct {
	UID         *int
	ResourceID  *string
	ResourceIDs *[]string
	ShowDeleted bool
	ProjectUID  *int
}

// GetInstanceV2 gets an instance by the resource_id.
func (s *Store) GetInstanceV2(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.instanceCache.Get(getInstanceCacheKey(*find.ResourceID)); ok {
			return v, nil
		}
	}
	if find.UID != nil {
		if v, ok := s.instanceIDCache.Get(*find.UID); ok {
			return v, nil
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
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Add(instance.UID, instance)
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
		s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
		s.instanceIDCache.Add(instance.UID, instance)
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
	var environment *string
	if instanceCreate.EnvironmentID != "" {
		environment = &instanceCreate.EnvironmentID
	}
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			INSERT INTO instance (
				resource_id,
				creator_id,
				updater_id,
				environment,
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
		environment,
		instanceCreate.Title,
		instanceCreate.Engine.String(),
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
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Add(instance.UID, instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage, maximumActivation int) (*InstanceMessage, error) {
	if patch.DataSources != nil {
		if err := validateDataSourceList(*patch.DataSources); err != nil {
			return nil, err
		}
	}

	set, args, where := []string{"updater_id = $1", "updated_ts = $2"}, []any{patch.UpdaterID, time.Now().Unix()}, []string{}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if patch.UpdateEnvironmentID {
		var environment *string
		if patch.EnvironmentID != "" {
			environment = &patch.EnvironmentID
		}
		set, args = append(set, fmt.Sprintf("environment = $%d", len(args)+1)), append(args, environment)
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
	if v := patch.OptionsUpsert; v != nil {
		options, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("options = $%d", len(args)+1)), append(args, options)
	}
	if v := patch.Metadata; v != nil {
		metadata, err := protojson.Marshal(v)
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

	instance := &InstanceMessage{}
	var engine string
	query := fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE %s
			RETURNING
				id,
				resource_id,
				environment,
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
	var environment sql.NullString
	var options, metadata []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&instance.UID,
		&instance.ResourceID,
		&environment,
		&instance.Title,
		&engine,
		&instance.EngineVersion,
		&instance.ExternalLink,
		&instance.Activation,
		&rowStatus,
		&options,
		&metadata,
	); err != nil {
		return nil, err
	}
	if environment.Valid {
		instance.EnvironmentID = environment.String
	}
	engineTypeValue, ok := storepb.Engine_value[engine]
	if !ok {
		return nil, errors.Errorf("invalid engine %s", engine)
	}
	instance.Engine = storepb.Engine(engineTypeValue)

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
	if err := common.ProtojsonUnmarshaler.Unmarshal(options, &instanceOptions); err != nil {
		return nil, err
	}
	instance.Options = &instanceOptions

	var instanceMetadata storepb.InstanceMetadata
	if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &instanceMetadata); err != nil {
		return nil, err
	}
	instance.Metadata = &instanceMetadata

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	s.instanceIDCache.Add(instance.UID, instance)
	return instance, nil
}

func (s *Store) listInstanceImplV2(ctx context.Context, tx *Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceIDs; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectUID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.id IN (SELECT DISTINCT instance_id FROM db WHERE project_id = $%d)", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("instance.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var instanceMessages []*InstanceMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			name,
			environment,
			engine,
			engine_version,
			external_link,
			activation,
			row_status,
			options,
			metadata
		FROM instance
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var environment sql.NullString
		var engine, rowStatus string
		var options, metadata []byte
		if err := rows.Scan(
			&instanceMessage.UID,
			&instanceMessage.ResourceID,
			&instanceMessage.Title,
			&environment,
			&engine,
			&instanceMessage.EngineVersion,
			&instanceMessage.ExternalLink,
			&instanceMessage.Activation,
			&rowStatus,
			&options,
			&metadata,
		); err != nil {
			return nil, err
		}
		if environment.Valid {
			instanceMessage.EnvironmentID = environment.String
		}
		engineTypeValue, ok := storepb.Engine_value[engine]
		if !ok {
			return nil, errors.Errorf("invalid engine %s", engine)
		}
		instanceMessage.Engine = storepb.Engine(engineTypeValue)

		instanceMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		var instanceOptions storepb.InstanceOptions
		if err := common.ProtojsonUnmarshaler.Unmarshal(options, &instanceOptions); err != nil {
			return nil, err
		}
		instanceMessage.Options = &instanceOptions
		var instanceMetadata storepb.InstanceMetadata
		if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &instanceMetadata); err != nil {
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

var countActivateInstanceQuery = "SELECT COUNT(*) FROM instance WHERE activation = TRUE AND row_status = 'NORMAL'"

// CheckActivationLimit checks if activation instance count reaches the limit.
func (s *Store) CheckActivationLimit(ctx context.Context, maximumActivation int) error {
	count := 0
	if err := s.db.db.QueryRowContext(ctx, countActivateInstanceQuery).Scan(&count); err != nil {
		return err
	}

	if count >= maximumActivation {
		return common.Errorf(common.Invalid, "activation instance count has reached the limit (%v)", count)
	}
	return nil
}

func validateDataSourceList(dataSources []*DataSourceMessage) error {
	dataSourceMap := map[string]bool{}
	adminCount := 0
	for _, dataSource := range dataSources {
		if dataSourceMap[dataSource.ID] {
			return status.Errorf(codes.InvalidArgument, "duplicate data source ID %s", dataSource.ID)
		}
		dataSourceMap[dataSource.ID] = true
		if dataSource.Type == api.Admin {
			adminCount++
		}
	}
	if adminCount != 1 {
		return status.Errorf(codes.InvalidArgument, "require exactly one admin data source")
	}
	return nil
}

// IgnoreDatabaseAndTableCaseSensitive returns true if the engine ignores database and table case sensitive.
func IgnoreDatabaseAndTableCaseSensitive(instance *InstanceMessage) bool {
	switch instance.Engine {
	case storepb.Engine_TIDB:
		return true
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		return instance.Metadata != nil && instance.Metadata.MysqlLowerCaseTableNames != 0
	case storepb.Engine_MSSQL:
		// In fact, SQL Server is possible to create a case-sensitive database and case-insensitive database on one instance.
		// https://www.webucator.com/article/how-to-check-case-sensitivity-in-sql-server/
		// But by default, SQL Server is case-insensitive.
		return true
	default:
		return false
	}
}
