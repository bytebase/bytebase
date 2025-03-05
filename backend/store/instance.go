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
func (s *Store) CreateInstanceV2(ctx context.Context, instanceCreate *InstanceMessage) (*InstanceMessage, error) {
	if err := validateDataSources(instanceCreate.Metadata); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	metadataBytes, err := protojson.Marshal(instanceCreate.Metadata)
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
		query := fmt.Sprintf(`
			UPDATE instance
			SET `+strings.Join(set, ", ")+`
			WHERE %s
			RETURNING
				resource_id,
				environment,
				deleted,
				metadata
		`, strings.Join(where, " AND "))
		var environment sql.NullString
		var metadata []byte
		if err := tx.QueryRowContext(ctx, query, args...).Scan(
			&instance.ResourceID,
			&environment,
			&instance.Deleted,
			&metadata,
		); err != nil {
			return nil, err
		}
		if environment.Valid {
			instance.EnvironmentID = environment.String
		}

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
			environment,
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

var countActivateInstanceQuery = "SELECT COUNT(1) FROM instance WHERE (metadata ? 'activation') AND (metadata->>'activation')::boolean = TRUE AND deleted = FALSE"

// GetActivatedInstanceCount gets the number of activated instances.
func (s *Store) GetActivatedInstanceCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.db.QueryRowContext(ctx, countActivateInstanceQuery).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
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
	switch instance.Metadata.GetEngine() {
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
