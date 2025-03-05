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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// InstanceMessage is the message for instance.
type InstanceMessage struct {
	ResourceID    string
	Title         string
	Engine        storepb.Engine
	ExternalLink  string
	Activation    bool
	EnvironmentID string
	Deleted       bool
	EngineVersion string
	Metadata      *storepb.Instance
}

// UpdateInstanceMessage is the message for updating an instance.
type UpdateInstanceMessage struct {
	ResourceID string

	Title         *string
	ExternalLink  *string
	Deleted       *bool
	EngineVersion *string
	Activation    *bool
	Metadata      *storepb.Instance
	EnvironmentID *string
}

// FindInstanceMessage is the message for finding instances.
type FindInstanceMessage struct {
	ResourceID  *string
	ResourceIDs *[]string
	ShowDeleted bool
}

// GetInstanceV2 gets an instance by the resource_id.
func (s *Store) GetInstanceV2(ctx context.Context, find *FindInstanceMessage) (*InstanceMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.instanceCache.Get(getInstanceCacheKey(*find.ResourceID)); ok {
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

	instances, err := listInstanceImplV2(ctx, tx, find)
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

// CreateInstanceV2 creates the instance.
func (s *Store) CreateInstanceV2(ctx context.Context, instanceCreate *InstanceMessage, maximumActivation int) (*InstanceMessage, error) {
	if err := validateDataSources(instanceCreate.Metadata); err != nil {
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

	metadataBytes, err := protojson.Marshal(instanceCreate.Metadata)
	if err != nil {
		return nil, err
	}

	var environment *string
	if instanceCreate.EnvironmentID != "" {
		environment = &instanceCreate.EnvironmentID
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO instance (
				resource_id,
				environment,
				name,
				engine,
				external_link,
				activation,
				metadata
			)
			SELECT $1, $2, $3, $4, $5, $6, $7
			%s
		`, where),
		instanceCreate.ResourceID,
		environment,
		instanceCreate.Title,
		instanceCreate.Engine.String(),
		instanceCreate.ExternalLink,
		instanceCreate.Activation,
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
		Title:         instanceCreate.Title,
		Engine:        instanceCreate.Engine,
		ExternalLink:  instanceCreate.ExternalLink,
		Activation:    instanceCreate.Activation,
		Metadata:      instanceCreate.Metadata,
	}
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	return instance, nil
}

// UpdateInstanceV2 updates an instance.
func (s *Store) UpdateInstanceV2(ctx context.Context, patch *UpdateInstanceMessage, maximumActivation int) (*InstanceMessage, error) {
	set, args, where := []string{}, []any{}, []string{}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EnvironmentID; v != nil {
		set, args = append(set, fmt.Sprintf("environment = $%d", len(args)+1)), append(args, *v)
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
	if v := patch.Deleted; v != nil {
		if patch.Activation != nil {
			return nil, errors.Errorf(`cannot set "activation" and "deleted" at the same time`)
		}
		if *patch.Deleted {
			set, args = append(set, fmt.Sprintf("activation = $%d", len(args)+1)), append(args, false)
		}
		set, args = append(set, fmt.Sprintf(`deleted = $%d`, len(args)+1)), append(args, *v)
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
	if len(set) > 0 {
		var engine string
		query := fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE %s
			RETURNING
				resource_id,
				environment,
				name,
				engine,
				engine_version,
				external_link,
				activation,
				deleted,
				metadata
		`, strings.Join(where, " AND "))
		var environment sql.NullString
		var metadata []byte
		if err := tx.QueryRowContext(ctx, query, args...).Scan(
			&instance.ResourceID,
			&environment,
			&instance.Title,
			&engine,
			&instance.EngineVersion,
			&instance.ExternalLink,
			&instance.Activation,
			&instance.Deleted,
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

		var instanceMetadata storepb.Instance
		if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &instanceMetadata); err != nil {
			return nil, err
		}
		instance.Metadata = &instanceMetadata
	} else {
		existedInstance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{
			ResourceID: &patch.ResourceID,
		})
		if err != nil {
			return nil, err
		}
		instance = existedInstance
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	return instance, nil
}

func listInstanceImplV2(ctx context.Context, tx *Tx, find *FindInstanceMessage) ([]*InstanceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceIDs; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = ANY($%d)", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("instance.deleted = $%d", len(args)+1)), append(args, false)
	}

	var instanceMessages []*InstanceMessage
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			resource_id,
			name,
			environment,
			engine,
			engine_version,
			external_link,
			activation,
			deleted,
			metadata
		FROM instance
		WHERE %s
		ORDER BY resource_id`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var instanceMessage InstanceMessage
		var environment sql.NullString
		var engine string
		var metadata []byte
		if err := rows.Scan(
			&instanceMessage.ResourceID,
			&instanceMessage.Title,
			&environment,
			&engine,
			&instanceMessage.EngineVersion,
			&instanceMessage.ExternalLink,
			&instanceMessage.Activation,
			&instanceMessage.Deleted,
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

		var instanceMetadata storepb.Instance
		if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &instanceMetadata); err != nil {
			return nil, err
		}
		instanceMessage.Metadata = &instanceMetadata
		instanceMessages = append(instanceMessages, &instanceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instanceMessages, nil
}

var countActivateInstanceQuery = "SELECT COUNT(*) FROM instance WHERE activation = TRUE AND deleted = FALSE"

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

func validateDataSources(metadata *storepb.Instance) error {
	dataSourceMap := map[string]bool{}
	adminCount := 0
	for _, dataSource := range metadata.GetDataSources() {
		if dataSourceMap[dataSource.GetId()] {
			return status.Errorf(codes.InvalidArgument, "duplicate data source ID %s", dataSource.GetId())
		}
		dataSourceMap[dataSource.GetId()] = true
		if dataSource.GetType() == storepb.DataSourceType_ADMIN {
			adminCount++
		}
	}
	if adminCount != 1 {
		return status.Errorf(codes.InvalidArgument, "require exactly one admin data source")
	}
	return nil
}

// IsObjectCaseSensitive returns true if the engine ignores database and table case sensitive.
func IsObjectCaseSensitive(instance *InstanceMessage) bool {
	switch instance.Engine {
	case storepb.Engine_TIDB:
		return false
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		return !(instance.Metadata != nil && instance.Metadata.MysqlLowerCaseTableNames != 0)
	case storepb.Engine_MSSQL:
		// In fact, SQL Server is possible to create a case-sensitive database and case-insensitive database on one instance.
		// https://www.webucator.com/article/how-to-check-case-sensitivity-in-sql-server/
		// But by default, SQL Server is case-insensitive.
		return false
	default:
		return true
	}
}
