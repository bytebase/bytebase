package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ServiceAccountMessage is the message for a service account.
type ServiceAccountMessage struct {
	ID int
	// Email must be lower case, format: {name}@{project-id}.service.bytebase.com or {name}@service.bytebase.com
	Email         string
	Name          string
	MemberDeleted bool
	// ProjectID is the owning project. NULL for workspace-level service accounts.
	ProjectID *string
}

// FindServiceAccountMessage is the message for finding service accounts.
type FindServiceAccountMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	// ProjectID filters by owning project. Use empty string for workspace-level service accounts.
	ProjectID *string
}

// CreateServiceAccountMessage is the message for creating a service account.
type CreateServiceAccountMessage struct {
	// Email must be lower case.
	Email string
	Name  string
	// ProjectID is the owning project. NULL for workspace-level service accounts.
	ProjectID *string
}

// UpdateServiceAccountMessage is the message to update a service account.
type UpdateServiceAccountMessage struct {
	Name   *string
	Delete *bool
}

// GetServiceAccountByEmail gets a service account by email.
func (s *Store) GetServiceAccountByEmail(ctx context.Context, email string) (*ServiceAccountMessage, error) {
	sas, err := s.ListServiceAccounts(ctx, &FindServiceAccountMessage{Email: &email, ShowDeleted: true})
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
	where := qb.Q().Space("type = ?", storepb.PrincipalType_SERVICE_ACCOUNT.String())

	if v := find.ID; v != nil {
		where.And("id = ?", *v)
	}
	if v := find.Email; v != nil {
		where.And("email = ?", strings.ToLower(*v))
	}
	if !find.ShowDeleted {
		where.And("deleted = ?", false)
	}
	if v := find.ProjectID; v != nil {
		if *v == "" {
			where.And("project_id IS NULL")
		} else {
			where.And("project_id = ?", *v)
		}
	}

	q := qb.Q().Space(`
		SELECT
			id,
			deleted,
			email,
			name,
			project_id
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

	var sas []*ServiceAccountMessage
	for rows.Next() {
		var sa ServiceAccountMessage
		var projectID sql.NullString
		if err := rows.Scan(
			&sa.ID,
			&sa.MemberDeleted,
			&sa.Email,
			&sa.Name,
			&projectID,
		); err != nil {
			return nil, err
		}
		if projectID.Valid {
			sa.ProjectID = &projectID.String
		}
		sas = append(sas, &sa)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return sas, nil
}

// CreateServiceAccount creates a service account.
func (s *Store) CreateServiceAccount(ctx context.Context, create *CreateServiceAccountMessage) (*ServiceAccountMessage, error) {
	email := strings.ToLower(create.Email)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	profileBytes, err := protojson.Marshal(&storepb.UserProfile{})
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
			project_id
		)
		VALUES (?, ?, ?, '', '', ?, ?)
		RETURNING id
	`, email, create.Name, storepb.PrincipalType_SERVICE_ACCOUNT.String(), profileBytes, create.ProjectID)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var saID int
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan(&saID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &ServiceAccountMessage{
		ID:        saID,
		Email:     email,
		Name:      create.Name,
		ProjectID: create.ProjectID,
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

	if set.Len() == 0 {
		return sa, nil
	}

	sqlStr, args, err := qb.Q().Space(`UPDATE principal SET ? WHERE id = ?
		RETURNING id, deleted, email, name, project_id`,
		set, sa.ID).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var updated ServiceAccountMessage
	var projectID sql.NullString
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan(
		&updated.ID,
		&updated.MemberDeleted,
		&updated.Email,
		&updated.Name,
		&projectID,
	); err != nil {
		return nil, err
	}
	if projectID.Valid {
		updated.ProjectID = &projectID.String
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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
