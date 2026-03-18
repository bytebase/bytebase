package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// ServiceAccountMessage is the message for a service account.
type ServiceAccountMessage struct {
	// Email must be lower case, format: {name}@{project-id}.service.bytebase.com or {name}@service.bytebase.com
	Email          string
	Name           string
	Workspace      string
	ServiceKeyHash string
	MemberDeleted  bool
	// Project is the owning project. NULL for workspace-level service accounts.
	Project *string
}

// FindServiceAccountMessage is the message for finding service accounts.
type FindServiceAccountMessage struct {
	Workspace   string
	Email       *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	// Project filters by owning project. Use empty string for workspace-level service accounts.
	Project *string
	// FilterQ is the CEL filter query.
	FilterQ *qb.Query
}

// CreateServiceAccountMessage is the message for creating a service account.
type CreateServiceAccountMessage struct {
	// Email must be lower case.
	Email          string
	Name           string
	Workspace      string
	ServiceKeyHash string
	// Project is the owning project. NULL for workspace-level service accounts.
	Project *string
}

// UpdateServiceAccountMessage is the message to update a service account.
type UpdateServiceAccountMessage struct {
	Name           *string
	ServiceKeyHash *string
	Delete         *bool
}

// GetServiceAccountByEmail gets a service account by email without workspace filter.
// For use by login flow, runners, and GetAccountByEmail (cross-workspace).
func (s *Store) GetServiceAccountByEmail(ctx context.Context, email string) (*ServiceAccountMessage, error) {
	q := qb.Q().Space(`
		SELECT deleted, email, name, workspace, service_key_hash, project
		FROM service_account
		WHERE email = ?
	`, strings.ToLower(email))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var sa ServiceAccountMessage
	var project sql.NullString
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&sa.MemberDeleted, &sa.Email, &sa.Name, &sa.Workspace, &sa.ServiceKeyHash, &project,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if project.Valid {
		sa.Project = &project.String
	}
	return &sa, nil
}

// GetServiceAccount gets a service account by workspace and email.
// For use by API layer (workspace-scoped).
func (s *Store) GetServiceAccount(ctx context.Context, workspace string, email string) (*ServiceAccountMessage, error) {
	sas, err := s.ListServiceAccounts(ctx, &FindServiceAccountMessage{Workspace: workspace, Email: &email, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	if len(sas) == 0 {
		return nil, nil
	}
	return sas[0], nil
}

// ListServiceAccounts lists service accounts.
func (s *Store) ListServiceAccounts(ctx context.Context, find *FindServiceAccountMessage) ([]*ServiceAccountMessage, error) {
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
			service_key_hash,
			project
		FROM service_account
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

	var sas []*ServiceAccountMessage
	for rows.Next() {
		var sa ServiceAccountMessage
		var project sql.NullString
		if err := rows.Scan(
			&sa.MemberDeleted,
			&sa.Email,
			&sa.Name,
			&sa.Workspace,
			&sa.ServiceKeyHash,
			&project,
		); err != nil {
			return nil, err
		}
		if project.Valid {
			sa.Project = &project.String
		}
		sas = append(sas, &sa)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	return sas, nil
}

// CreateServiceAccount creates a service account.
func (s *Store) CreateServiceAccount(ctx context.Context, create *CreateServiceAccountMessage) (*ServiceAccountMessage, error) {
	email := strings.ToLower(create.Email)

	q := qb.Q().Space(`
		INSERT INTO service_account (
			email,
			name,
			workspace,
			service_key_hash,
			project
		)
		VALUES (?, ?, ?, ?, ?)
	`, email, create.Name, create.Workspace, create.ServiceKeyHash, create.Project)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, sqlStr, args...); err != nil {
		return nil, err
	}

	return &ServiceAccountMessage{
		Email:          email,
		Name:           create.Name,
		Workspace:      create.Workspace,
		ServiceKeyHash: create.ServiceKeyHash,
		Project:        create.Project,
	}, nil
}

// UpdateServiceAccount updates a service account.
func (s *Store) UpdateServiceAccount(ctx context.Context, sa *ServiceAccountMessage, patch *UpdateServiceAccountMessage) (*ServiceAccountMessage, error) {
	set := qb.Q()
	if v := patch.Delete; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if v := patch.Name; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.ServiceKeyHash; v != nil {
		set.Comma("service_key_hash = ?", *v)
	}

	if set.Len() == 0 {
		return sa, nil
	}

	sqlStr, args, err := qb.Q().Space(`UPDATE service_account SET ? WHERE email = ? AND workspace = ?
		RETURNING deleted, email, name, workspace, service_key_hash, project`,
		set, sa.Email, sa.Workspace).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var updated ServiceAccountMessage
	var project sql.NullString
	if err := s.GetDB().QueryRowContext(ctx, sqlStr, args...).Scan(
		&updated.MemberDeleted,
		&updated.Email,
		&updated.Name,
		&updated.Workspace,
		&updated.ServiceKeyHash,
		&project,
	); err != nil {
		return nil, err
	}
	if project.Valid {
		updated.Project = &project.String
	}

	// Also update the unified cache if this SA is in there
	s.userEmailCache.Remove(sa.Email)

	return &updated, nil
}

// DeleteServiceAccount soft-deletes a service account.
func (s *Store) DeleteServiceAccount(ctx context.Context, sa *ServiceAccountMessage) error {
	deleted := true
	_, err := s.UpdateServiceAccount(ctx, sa, &UpdateServiceAccountMessage{Delete: &deleted})
	return err
}

// UndeleteServiceAccount restores a soft-deleted service account.
func (s *Store) UndeleteServiceAccount(ctx context.Context, sa *ServiceAccountMessage) (*ServiceAccountMessage, error) {
	deleted := false
	return s.UpdateServiceAccount(ctx, sa, &UpdateServiceAccountMessage{Delete: &deleted})
}
