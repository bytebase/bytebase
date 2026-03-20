package store

// runner_queries.go provides cross-workspace query methods for background runners.
// These methods intentionally omit workspace filtering because runners process
// tasks across all workspaces. They should NOT be used by API handlers.
//
// After loading an entity, use its .Workspace field for subsequent workspace-scoped queries.

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// GetProjectByResourceID gets a project by its globally unique resource ID without workspace filter.
// For use by runners that resolve workspace from the loaded entity.
func (s *Store) GetProjectByResourceID(ctx context.Context, resourceID string) (*ProjectMessage, error) {
	if v, ok := s.projectCache.Get(resourceID); ok && s.enableCache {
		return v, nil
	}

	q := qb.Q().Space("SELECT resource_id, workspace, name, setting, deleted FROM project WHERE resource_id = ?", resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	var project ProjectMessage
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&project.ResourceID,
		&project.Workspace,
		&project.Title,
		&payload,
		&project.Deleted,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	setting := &storepb.Project{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, setting); err != nil {
		return nil, err
	}
	project.Setting = setting
	s.storeProjectCache(&project)
	return &project, nil
}

// GetInstanceByResourceID gets an instance by its globally unique resource ID without workspace filter.
// For use by runners that resolve workspace from the loaded entity.
func (s *Store) GetInstanceByResourceID(ctx context.Context, resourceID string) (*InstanceMessage, error) {
	if v, ok := s.instanceCache.Get(getInstanceCacheKey(resourceID)); ok && s.enableCache {
		return v, nil
	}

	q := qb.Q().Space(`
		SELECT
			instance.resource_id,
			instance.workspace,
			instance.environment,
			instance.deleted,
			instance.metadata
		FROM instance
		WHERE instance.resource_id = ?
	`, resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var instance InstanceMessage
	var environment sql.NullString
	var metadata []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&instance.ResourceID,
		&instance.Workspace,
		&environment,
		&instance.Deleted,
		&metadata,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if environment.Valid {
		instance.EnvironmentID = &environment.String
	}
	instanceMetadata := &storepb.Instance{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, instanceMetadata); err != nil {
		return nil, err
	}
	instance.Metadata = instanceMetadata

	if err := s.deobfuscateInstances(ctx, []*InstanceMessage{&instance}); err != nil {
		return nil, err
	}
	s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), &instance)
	return &instance, nil
}

// ListAllInstances lists instances across all workspaces without workspace filter.
// For use by runners (e.g., schema sync) that need to process all instances.
func (s *Store) ListAllInstances(ctx context.Context, showDeleted bool) ([]*InstanceMessage, error) {
	where := qb.Q().Space("TRUE")
	if !showDeleted {
		where.And("instance.deleted = ?", false)
	}

	q := qb.Q().Space(`
		SELECT
			instance.resource_id,
			instance.workspace,
			instance.environment,
			instance.deleted,
			instance.metadata
		FROM instance
		WHERE ?
		ORDER BY resource_id ASC
	`, where)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var instances []*InstanceMessage
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var instance InstanceMessage
		var environment sql.NullString
		var metadata []byte
		if err := rows.Scan(
			&instance.ResourceID,
			&instance.Workspace,
			&environment,
			&instance.Deleted,
			&metadata,
		); err != nil {
			return nil, err
		}
		if environment.Valid {
			instance.EnvironmentID = &environment.String
		}
		instanceMetadata := &storepb.Instance{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, instanceMetadata); err != nil {
			return nil, err
		}
		instance.Metadata = instanceMetadata
		instances = append(instances, &instance)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Deobfuscate per-workspace (group by workspace to avoid redundant secret lookups).
	byWorkspace := make(map[string][]*InstanceMessage)
	for _, inst := range instances {
		byWorkspace[inst.Workspace] = append(byWorkspace[inst.Workspace], inst)
	}
	for _, wsInstances := range byWorkspace {
		if err := s.deobfuscateInstances(ctx, wsInstances); err != nil {
			return nil, err
		}
	}

	for _, instance := range instances {
		s.instanceCache.Add(getInstanceCacheKey(instance.ResourceID), instance)
	}
	return instances, nil
}

// DeleteExpiredExportArchivesAll deletes expired export archives across all workspaces.
// For use by the cleaner runner.
func (s *Store) DeleteExpiredExportArchivesAll(ctx context.Context, retentionPeriod time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-retentionPeriod)
	q := qb.Q().Space("DELETE FROM export_archive WHERE created_at < ?", cutoffTime)
	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}
	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
