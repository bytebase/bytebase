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
	defer tx.PTx.Rollback()

	projectMember, err := createProjectMember(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
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
	defer tx.PTx.Rollback()

	projectMember, err := patchProjectMember(ctx, tx.PTx, patch)
	if err != nil {
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
	defer tx.PTx.Rollback()

	if err := deleteProjectMember(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// SetProjectMember set the project member with provided project member list
func (s *ProjectMemberService) SetProjectMember(ctx context.Context, set *api.ProjectMemberSet) (createdMember, deletedMember []*api.ProjectMember, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	findProjectMember := &api.ProjectMemberFind{ProjectID: &set.ID}
	existingProjectMemberList, err := findProjectMemberList(ctx, tx.PTx, findProjectMember)
	if err != nil {
		return nil, nil, FormatError(err)
	}

	oldMemberMap := make(map[int]*api.ProjectMember)
	for _, existingMember := range existingProjectMemberList {
		oldMemberMap[existingMember.PrincipalID] = existingMember
	}

	newMemberMap := make(map[int]bool)
	for _, createMember := range set.List {
		newMemberMap[createMember.PrincipalID] = true
	}

	createdMemberList := make([]*api.ProjectMember, 0)
	deletedMemberList := make([]*api.ProjectMember, 0)
	for _, memberCreate := range set.List {
		// if the member exists (NOTICE: a member with the same principal ID but different role provider will be considered as two member)
		//  we will try to update its field
		if memberBefore, ok := oldMemberMap[memberCreate.PrincipalID]; ok && memberBefore.RoleProvider == memberCreate.RoleProvider {
			// if we update a member, we will the member in both createdMemberList and deletedMemberList
			updatedMember, err := patchProjectMember(ctx, tx.PTx, &api.ProjectMemberPatch{
				ID:           memberBefore.ID,
				UpdaterID:    memberCreate.CreatorID,
				Role:         (*string)(&memberCreate.Role),
				RoleProvider: (*string)(&memberCreate.RoleProvider),
				Payload:      &memberCreate.Payload,
			})
			if err != nil {
				return nil, nil, FormatError(err)
			}
			// we append the updated member to createdMemberList, old member to the deletedMemberList
			createdMemberList = append(createdMemberList, updatedMember)
			deletedMemberList = append(deletedMemberList, memberBefore)
		} else {
			createdMember, err := createProjectMember(ctx, tx.PTx, memberCreate)
			if err != nil {
				return nil, nil, FormatError(err)
			}
			createdMemberList = append(createdMemberList, createdMember)
		}
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, nil, FormatError(err)
	}

	return createdMemberList, deletedMemberList, nil
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
			principal_id,
			role_provider,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, role_provider, payload
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Role,
		create.PrincipalID,
		create.RoleProvider,
		create.Payload,
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
		&projectMember.RoleProvider,
		&projectMember.Payload,
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
			principal_id,
			role_provider,
			payload
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
			&projectMember.RoleProvider,
			&projectMember.Payload,
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
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, api.Role(*v))
	}
	if v := patch.RoleProvider; v != nil {
		set, args = append(set, fmt.Sprintf("role_provider = $%d", len(args)+1)), append(args, api.Role(*v))
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE project_member
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id, role_provider, payload
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
			&projectMember.RoleProvider,
			&projectMember.Payload,
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
