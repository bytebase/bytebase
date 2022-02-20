package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.ProjectMemberService = (*ProjectMemberService)(nil)
)

// ProjectMemberService represents a service for managing projectMember.
type ProjectMemberService struct {
	l  *zap.Logger
	db *DB
}

// NewProjectMemberService returns a new instance of ProjectMemberService.
func NewProjectMemberService(logger *zap.Logger, db *DB) *ProjectMemberService {
	return &ProjectMemberService{l: logger, db: db}
}

// CreateProjectMember creates a new projectMember.
func (s *ProjectMemberService) CreateProjectMember(ctx context.Context, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	projectMember, err := pgCreateProjectMember(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}
	if _, err := createProjectMember(ctx, tx.Tx, create); err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
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
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findProjectMemberList(ctx, tx.PTx, find)
	if err != nil {
		return []*api.ProjectMember{}, err
	}

	return list, nil
}

// FindProjectMember finds project members.
func (s *ProjectMemberService) FindProjectMember(ctx context.Context, find *api.ProjectMemberFind) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findProjectMemberList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d project members with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchProjectMember updates an existing projectMember by ID.
// Returns ENOTFOUND if projectMember does not exist.
func (s *ProjectMemberService) PatchProjectMember(ctx context.Context, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	projectMember, err := pgPatchProjectMember(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}
	if _, err := patchProjectMember(ctx, tx.Tx, patch); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// DeleteProjectMember deletes an existing projectMember by ID.
func (s *ProjectMemberService) DeleteProjectMember(ctx context.Context, delete *api.ProjectMemberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	if err := pgDeleteProjectMember(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}
	if err := deleteProjectMember(ctx, tx.Tx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
		return FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createProjectMember creates a new projectMember.
func createProjectMember(ctx context.Context, tx *sql.Tx, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_member (
			creator_id,
			updater_id,
			project_id,
			role,
			principal_id
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Role,
		create.PrincipalID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectMember api.ProjectMember
	if err := row.Scan(
		&projectMember.ID,
		&projectMember.CreatorID,
		&projectMember.CreatedTs,
		&projectMember.UpdaterID,
		&projectMember.UpdatedTs,
		&projectMember.ProjectID,
		&projectMember.Role,
		&projectMember.PrincipalID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &projectMember, nil
}

// pgCreateProjectMember creates a new projectMember.
func pgCreateProjectMember(ctx context.Context, tx *sql.Tx, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_member (
			creator_id,
			updater_id,
			project_id,
			role,
			principal_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Role,
		create.PrincipalID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectMember api.ProjectMember
	if err := row.Scan(
		&projectMember.ID,
		&projectMember.CreatorID,
		&projectMember.CreatedTs,
		&projectMember.UpdaterID,
		&projectMember.UpdatedTs,
		&projectMember.ProjectID,
		&projectMember.Role,
		&projectMember.PrincipalID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &projectMember, nil
}

func findProjectMemberList(ctx context.Context, tx *sql.Tx, find *api.ProjectMemberFind) (_ []*api.ProjectMember, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
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
		if err := rows.Scan(
			&projectMember.ID,
			&projectMember.CreatorID,
			&projectMember.CreatedTs,
			&projectMember.UpdaterID,
			&projectMember.UpdatedTs,
			&projectMember.ProjectID,
			&projectMember.Role,
			&projectMember.PrincipalID,
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
func patchProjectMember(ctx context.Context, tx *sql.Tx, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Role; v != nil {
		set, args = append(set, "role = ?"), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE project_member
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectMember api.ProjectMember
		if err := row.Scan(
			&projectMember.ID,
			&projectMember.CreatorID,
			&projectMember.CreatedTs,
			&projectMember.UpdaterID,
			&projectMember.UpdatedTs,
			&projectMember.ProjectID,
			&projectMember.Role,
			&projectMember.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		return &projectMember, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project member ID not found: %d", patch.ID)}
}

// pgPatchProjectMember updates a projectMember by ID. Returns the new state of the projectMember after update.
func pgPatchProjectMember(ctx context.Context, tx *sql.Tx, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE project_member
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectMember api.ProjectMember
		if err := row.Scan(
			&projectMember.ID,
			&projectMember.CreatorID,
			&projectMember.CreatedTs,
			&projectMember.UpdaterID,
			&projectMember.UpdatedTs,
			&projectMember.ProjectID,
			&projectMember.Role,
			&projectMember.PrincipalID,
		); err != nil {
			return nil, FormatError(err)
		}

		return &projectMember, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project member ID not found: %d", patch.ID)}
}

// deleteProjectMember permanently deletes a projectMember by ID.
func deleteProjectMember(ctx context.Context, tx *sql.Tx, delete *api.ProjectMemberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = ?`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}

// pgDeleteProjectMember permanently deletes a projectMember by ID.
func pgDeleteProjectMember(ctx context.Context, tx *sql.Tx, delete *api.ProjectMemberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
