package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// WorkloadIdentityMessage is the message for a workload identity.
type WorkloadIdentityMessage struct {
	// Email must be lower case, format: {name}@{project-id}.workload.bytebase.com or {name}@workload.bytebase.com
	Email         string
	Name          string
	Workspace     string
	MemberDeleted bool
	// Project is the owning project. NULL for workspace-level workload identities.
	Project *string
	// Config is the workload identity configuration.
	Config *storepb.WorkloadIdentityConfig
}

// FindWorkloadIdentityMessage is the message for finding workload identities.
type FindWorkloadIdentityMessage struct {
	Workspace   string
	Email       *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	// Project filters by owning project. Use empty string for workspace-level workload identities.
	Project *string
	// FilterQ is the CEL filter query.
	FilterQ *qb.Query
}

// CreateWorkloadIdentityMessage is the message for creating a workload identity.
type CreateWorkloadIdentityMessage struct {
	// Email must be lower case.
	Email     string
	Name      string
	Workspace string
	// Project is the owning project. NULL for workspace-level workload identities.
	Project *string
	// Config is the workload identity configuration.
	Config *storepb.WorkloadIdentityConfig
}

// UpdateWorkloadIdentityMessage is the message to update a workload identity.
type UpdateWorkloadIdentityMessage struct {
	Name   *string
	Delete *bool
	Config *storepb.WorkloadIdentityConfig
}

// GetWorkloadIdentityByEmail gets a workload identity by email without workspace filter.
// For use by login flow, runners, and GetAccountByEmail (cross-workspace).
func (s *Store) GetWorkloadIdentityByEmail(ctx context.Context, email string) (*WorkloadIdentityMessage, error) {
	q := qb.Q().Space(`
		SELECT deleted, email, name, workspace, project, config
		FROM workload_identity
		WHERE email = ?
	`, strings.ToLower(email))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var wi WorkloadIdentityMessage
	var project sql.NullString
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&wi.MemberDeleted, &wi.Email, &wi.Name, &wi.Workspace, &project, &configBytes,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if project.Valid {
		wi.Project = &project.String
	}
	var config storepb.WorkloadIdentityConfig
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}
	wi.Config = &config
	return &wi, nil
}

// GetWorkloadIdentity gets a workload identity by workspace and email.
// For use by API layer (workspace-scoped).
func (s *Store) GetWorkloadIdentity(ctx context.Context, workspace string, email string) (*WorkloadIdentityMessage, error) {
	wis, err := s.ListWorkloadIdentities(ctx, &FindWorkloadIdentityMessage{Workspace: workspace, Email: &email, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(wis) == 0 {
		return nil, nil
	}
	return wis[0], nil
}

// ListWorkloadIdentities lists workload identities.
func (s *Store) ListWorkloadIdentities(ctx context.Context, find *FindWorkloadIdentityMessage) ([]*WorkloadIdentityMessage, error) {
	where := qb.Q().Space("workspace = ?", find.Workspace)

	if v := find.Email; v != nil {
		where.And("email = ?", strings.ToLower(*v))
	}
	if !find.ShowDeleted {
		where.And("deleted = ?", false)
	}
	if v := find.Project; v != nil {
		if *v == "" {
			where.And("project IS NULL")
		} else {
			where.And("project = ?", *v)
		}
	}
	if v := find.FilterQ; v != nil {
		where.And("?", v)
	}

	q := qb.Q().Space(`
		SELECT
			deleted,
			email,
			name,
			workspace,
			project,
			config
		FROM workload_identity
		WHERE ?
		ORDER BY created_at ASC
	`, where)

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wis []*WorkloadIdentityMessage
	for rows.Next() {
		var wi WorkloadIdentityMessage
		var project sql.NullString
		var configBytes []byte
		if err := rows.Scan(
			&wi.MemberDeleted,
			&wi.Email,
			&wi.Name,
			&wi.Workspace,
			&project,
			&configBytes,
		); err != nil {
			return nil, err
		}
		if project.Valid {
			wi.Project = &project.String
		}
		var config storepb.WorkloadIdentityConfig
		if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, &config); err != nil {
			return nil, err
		}
		wi.Config = &config
		wis = append(wis, &wi)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	return wis, nil
}

// CreateWorkloadIdentity creates a workload identity.
func (s *Store) CreateWorkloadIdentity(ctx context.Context, create *CreateWorkloadIdentityMessage) (*WorkloadIdentityMessage, error) {
	email := strings.ToLower(create.Email)

	configBytes, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO workload_identity (
			email,
			name,
			workspace,
			project,
			config
		)
		VALUES (?, ?, ?, ?, ?)
	`, email, create.Name, create.Workspace, create.Project, configBytes)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, sqlStr, args...); err != nil {
		return nil, err
	}

	return &WorkloadIdentityMessage{
		Email:     email,
		Name:      create.Name,
		Workspace: create.Workspace,
		Project:   create.Project,
		Config:    create.Config,
	}, nil
}

// UpdateWorkloadIdentity updates a workload identity.
func (s *Store) UpdateWorkloadIdentity(ctx context.Context, wi *WorkloadIdentityMessage, patch *UpdateWorkloadIdentityMessage) (*WorkloadIdentityMessage, error) {
	set := qb.Q()
	if v := patch.Delete; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if v := patch.Name; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Config; v != nil {
		configBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set.Comma("config = ?", configBytes)
	}

	if set.Len() == 0 {
		return wi, nil
	}

	sqlStr, args, err := qb.Q().Space(`UPDATE workload_identity SET ? WHERE email = ? AND workspace = ?
		RETURNING deleted, email, name, workspace, project, config`,
		set, wi.Email, wi.Workspace).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var updated WorkloadIdentityMessage
	var project sql.NullString
	var configBytes []byte
	if err := s.GetDB().QueryRowContext(ctx, sqlStr, args...).Scan(
		&updated.MemberDeleted,
		&updated.Email,
		&updated.Name,
		&updated.Workspace,
		&project,
		&configBytes,
	); err != nil {
		return nil, err
	}
	if project.Valid {
		updated.Project = &project.String
	}
	var config storepb.WorkloadIdentityConfig
	if err := common.ProtojsonUnmarshaler.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}
	updated.Config = &config

	s.userEmailCache.Remove(wi.Email)

	return &updated, nil
}

// DeleteWorkloadIdentity soft-deletes a workload identity.
func (s *Store) DeleteWorkloadIdentity(ctx context.Context, wi *WorkloadIdentityMessage) error {
	deleted := true
	_, err := s.UpdateWorkloadIdentity(ctx, wi, &UpdateWorkloadIdentityMessage{Delete: &deleted})
	return err
}

// UndeleteWorkloadIdentity restores a soft-deleted workload identity.
func (s *Store) UndeleteWorkloadIdentity(ctx context.Context, wi *WorkloadIdentityMessage) (*WorkloadIdentityMessage, error) {
	deleted := false
	return s.UpdateWorkloadIdentity(ctx, wi, &UpdateWorkloadIdentityMessage{Delete: &deleted})
}
