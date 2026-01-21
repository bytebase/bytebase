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
	ID int
	// Email must be lower case, format: {name}@{project-id}.workload.bytebase.com or {name}@workload.bytebase.com
	Email         string
	Name          string
	MemberDeleted bool
	// Project is the owning project. NULL for workspace-level workload identities.
	Project *string
	// Config is the workload identity configuration.
	Config *storepb.WorkloadIdentityConfig
}

// FindWorkloadIdentityMessage is the message for finding workload identities.
type FindWorkloadIdentityMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	// Project filters by owning project. Use empty string for workspace-level workload identities.
	Project *string
}

// CreateWorkloadIdentityMessage is the message for creating a workload identity.
type CreateWorkloadIdentityMessage struct {
	// Email must be lower case.
	Email string
	Name  string
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

// GetWorkloadIdentityByEmail gets a workload identity by email.
func (s *Store) GetWorkloadIdentityByEmail(ctx context.Context, email string) (*WorkloadIdentityMessage, error) {
	wis, err := s.ListWorkloadIdentities(ctx, &FindWorkloadIdentityMessage{Email: &email, ShowDeleted: true})
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
	where := qb.Q().Space("type = ?", storepb.PrincipalType_WORKLOAD_IDENTITY.String())

	if v := find.ID; v != nil {
		where.And("id = ?", *v)
	}
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

	q := qb.Q().Space(`
		SELECT
			id,
			deleted,
			email,
			name,
			project,
			profile
		FROM principal
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

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wis []*WorkloadIdentityMessage
	for rows.Next() {
		var wi WorkloadIdentityMessage
		var project sql.NullString
		var profileBytes []byte
		if err := rows.Scan(
			&wi.ID,
			&wi.MemberDeleted,
			&wi.Email,
			&wi.Name,
			&project,
			&profileBytes,
		); err != nil {
			return nil, err
		}
		if project.Valid {
			wi.Project = &project.String
		}
		// Parse profile to extract workload identity config
		var profile storepb.UserProfile
		if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
			return nil, err
		}
		wi.Config = profile.WorkloadIdentityConfig
		wis = append(wis, &wi)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return wis, nil
}

// CreateWorkloadIdentity creates a workload identity.
func (s *Store) CreateWorkloadIdentity(ctx context.Context, create *CreateWorkloadIdentityMessage) (*WorkloadIdentityMessage, error) {
	email := strings.ToLower(create.Email)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	profile := &storepb.UserProfile{
		WorkloadIdentityConfig: create.Config,
	}
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO principal (
			email,
			name,
			type,
			password_hash,
			phone,
			profile,
			project
		)
		VALUES (?, ?, ?, '', '', ?, ?)
		RETURNING id
	`, email, create.Name, storepb.PrincipalType_WORKLOAD_IDENTITY.String(), profileBytes, create.Project)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var wiID int
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan(&wiID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &WorkloadIdentityMessage{
		ID:      wiID,
		Email:   email,
		Name:    create.Name,
		Project: create.Project,
		Config:  create.Config,
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
	// Config update requires updating the profile JSON
	updateConfig := patch.Config != nil
	if updateConfig {
		profile := &storepb.UserProfile{
			WorkloadIdentityConfig: patch.Config,
		}
		profileBytes, err := protojson.Marshal(profile)
		if err != nil {
			return nil, err
		}
		set.Comma("profile = ?", profileBytes)
	}

	if set.Len() == 0 {
		return wi, nil
	}

	sqlStr, args, err := qb.Q().Space(`UPDATE principal SET ? WHERE id = ?
		RETURNING id, deleted, email, name, project, profile`,
		set, wi.ID).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var updated WorkloadIdentityMessage
	var project sql.NullString
	var profileBytes []byte
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan(
		&updated.ID,
		&updated.MemberDeleted,
		&updated.Email,
		&updated.Name,
		&project,
		&profileBytes,
	); err != nil {
		return nil, err
	}
	if project.Valid {
		updated.Project = &project.String
	}
	// Parse profile to extract workload identity config
	var profile storepb.UserProfile
	if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
		return nil, err
	}
	updated.Config = profile.WorkloadIdentityConfig

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Also update the unified cache if this WI is in there
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
