package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
)

var (
	_ api.ProjectMemberService = (*ProjectMemberService)(nil)
)

// ProjectMemberService represents a service for managing projectMember.
type ProjectMemberService struct {
	l  *bytebase.Logger
	db *DB
}

// NewProjectMemberService returns a new instance of ProjectMemberService.
func NewProjectMemberService(logger *bytebase.Logger, db *DB) *ProjectMemberService {
	return &ProjectMemberService{l: logger, db: db}
}

// CreateProjectMember creates a new projectMember.
func (s *ProjectMemberService) CreateProjectMember(ctx context.Context, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectMember, err := createProjectMember(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// FindProjectMemberList retrieves a list of projectMembers based on find.
func (s *ProjectMemberService) FindProjectMemberList(ctx context.Context, find *api.ProjectMemberFind) ([]*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectMemberList(ctx, tx, find)
	if err != nil {
		return []*api.ProjectMember{}, err
	}

	return list, nil
}

// FindProjectMember retrieves a single projectMember based on find.
// Returns ENOTFOUND if no matching record.
// Returns the first matching one and prints a warning if finding more than 1 matching records.
func (s *ProjectMemberService) FindProjectMember(ctx context.Context, find *api.ProjectMemberFind) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectMemberList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project member not found: %v", find)}
	} else if len(list) > 1 {
		s.l.Logf(bytebase.WARN, "found mulitple project members: %d, expect 1", len(list))
	}
	return list[0], nil
}

// PatchProjectMemberByID updates an existing projectMember by ID.
// Returns ENOTFOUND if projectMember does not exist.
func (s *ProjectMemberService) PatchProjectMemberByID(ctx context.Context, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectMember, err := patchProjectMember(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// DeleteProjectMemberByID deletes an existing projectMember by ID.
// Returns ENOTFOUND if projectMember does not exist.
func (s *ProjectMemberService) DeleteProjectMemberByID(ctx context.Context, delete *api.ProjectMemberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteProjectMember(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createProjectMember creates a new projectMember.
func createProjectMember(ctx context.Context, tx *Tx, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_member (
			workspace_id,
			creator_id,
			updater_id,
			project_id,
			`+"`role`,"+`
			principal_id
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`,
		create.WorkspaceId,
		create.CreatorId,
		create.CreatorId,
		create.ProjectId,
		create.Role,
		create.PrincipalId,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectMember api.ProjectMember
	projectMember.Project = &api.ResourceObject{}
	if err := row.Scan(
		&projectMember.ID,
		&projectMember.WorkspaceId,
		&projectMember.CreatorId,
		&projectMember.CreatedTs,
		&projectMember.UpdaterId,
		&projectMember.UpdatedTs,
		&projectMember.Project.ID,
		&projectMember.Role,
		&projectMember.PrincipalId,
	); err != nil {
		return nil, FormatError(err)
	}

	return &projectMember, nil
}

func findProjectMemberList(ctx context.Context, tx *Tx, find *api.ProjectMemberFind) (_ []*api.ProjectMember, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.WorkspaceId; v != nil {
		where, args = append(where, "workspace_id = ?"), append(args, *v)
	}
	if v := find.ProjectId; v != nil {
		where, args = append(where, "project_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
			workspace_id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			project_id,
		    role,
		    principal_id
		FROM project_member
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.ProjectMember, 0)
	for rows.Next() {
		var projectMember api.ProjectMember
		projectMember.Project = &api.ResourceObject{}
		if err := rows.Scan(
			&projectMember.ID,
			&projectMember.WorkspaceId,
			&projectMember.CreatorId,
			&projectMember.CreatedTs,
			&projectMember.UpdaterId,
			&projectMember.UpdatedTs,
			&projectMember.Project.ID,
			&projectMember.Role,
			&projectMember.PrincipalId,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &projectMember)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchProjectMember updates a projectMember by ID. Returns the new state of the projectMember after update.
func patchProjectMember(ctx context.Context, tx *Tx, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.Role; v != nil {
		set, args = append(set, "role = ?"), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE project_member
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, workspace_id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectMember api.ProjectMember
		projectMember.Project = &api.ResourceObject{}
		if err := row.Scan(
			&projectMember.ID,
			&projectMember.WorkspaceId,
			&projectMember.CreatorId,
			&projectMember.CreatedTs,
			&projectMember.UpdaterId,
			&projectMember.UpdatedTs,
			&projectMember.Project.ID,
			&projectMember.Role,
			&projectMember.PrincipalId,
		); err != nil {
			return nil, FormatError(err)
		}

		return &projectMember, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project member ID not found: %d", patch.ID)}
}

// deleteProjectMember permanently deletes a projectMember by ID.
func deleteProjectMember(ctx context.Context, tx *Tx, delete *api.ProjectMemberDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project member ID not found: %d", delete.ID)}
	}

	return nil
}
