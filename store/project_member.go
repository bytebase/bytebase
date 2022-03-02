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

// getBatchUpdatePrincipalIDList return the principal ID for each operation (this function may be a litter overhead, but it is easy to be tested)
func getBatchUpdatePrincipalIDList(ctx context.Context, oldPrincipalIDList []int, newPrincipalIDList []int) (createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList []int, err error) {
	oldPrincipalIDSet := make(map[int]bool)
	for _, id := range oldPrincipalIDList {
		oldPrincipalIDSet[id] = true
	}

	newPrincipalIDSet := make(map[int]bool)
	for _, id := range newPrincipalIDList {
		newPrincipalIDSet[id] = true
	}

	createPrincipalIDList = make([]int, 0)
	patchPrincipalIDList = make([]int, 0)
	for _, newID := range newPrincipalIDList {
		// if the ID exists, we will try to update it (NOTICE: a member with the same principal ID but different role provider will be considered as separate member)
		if _, ok := oldPrincipalIDSet[newID]; ok {
			patchPrincipalIDList = append(patchPrincipalIDList, newID)
		} else {
			createPrincipalIDList = append(createPrincipalIDList, newID)
		}
	}

	deletePrincipalIDList = make([]int, 0)
	for _, oldID := range oldPrincipalIDList {
		// if the ID dose exist on the create list we will update it (done above)
		if _, ok := newPrincipalIDSet[oldID]; ok {
			continue
		}
		deletePrincipalIDList = append(deletePrincipalIDList, oldID)
	}

	return createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList, nil
}

// BatchUpdateProjectMember update the project member with provided project member list
func (s *ProjectMemberService) BatchUpdateProjectMember(ctx context.Context, batchUpdate *api.ProjectMemberBatchUpdate) (createdMemberList, deletedMemberList []*api.ProjectMember, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	txRead, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, FormatError(err)
	}
	findProjectMember := &api.ProjectMemberFind{
		ProjectID:    &batchUpdate.ID,
		RoleProvider: &batchUpdate.RoleProvider,
	}
	oldProjectMemberList, err := findProjectMemberList(ctx, txRead.PTx, findProjectMember)
	if err != nil {
		return nil, nil, err
	}

	oldMemberMap := make(map[int]*api.ProjectMember)
	oldPrincipalIDList := make([]int, 0, len(oldProjectMemberList))
	for _, oldMember := range oldProjectMemberList {
		oldMemberMap[oldMember.PrincipalID] = oldMember
		oldPrincipalIDList = append(oldPrincipalIDList, oldMember.PrincipalID)
	}
	newMemberMap := make(map[int]*api.ProjectMemberCreate)
	newPrincipalIDList := make([]int, 0, len(batchUpdate.List))
	for _, newMember := range batchUpdate.List {
		newMemberMap[newMember.PrincipalID] = newMember
		newPrincipalIDList = append(newPrincipalIDList, newMember.PrincipalID)
	}

	createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList, err := getBatchUpdatePrincipalIDList(ctx, oldPrincipalIDList, newPrincipalIDList)
	if err != nil {
		return nil, nil, FormatError(err)
	}

	createdMemberList = make([]*api.ProjectMember, 0)
	deletedMemberList = make([]*api.ProjectMember, 0)
	for _, id := range createPrincipalIDList {
		memberCreate := newMemberMap[id]
		createdMember, err := createProjectMember(ctx, tx.PTx, memberCreate)
		if err != nil {
			return nil, nil, FormatError(err)
		}
		createdMemberList = append(createdMemberList, createdMember)
	}

	for _, id := range patchPrincipalIDList {
		oldMember := oldMemberMap[id]
		newMember := newMemberMap[id]
		memberPatch := &api.ProjectMemberPatch{
			ID:           oldMember.ID,
			UpdaterID:    batchUpdate.UpdaterID,
			Role:         (*string)(&newMember.Role),
			RoleProvider: (*string)(&newMember.RoleProvider),
			Payload:      &newMember.Payload,
		}
		patchedMember, err := patchProjectMember(ctx, tx.PTx, memberPatch)
		if err != nil {
			return nil, nil, FormatError(err)
		}
		createdMemberList = append(createdMemberList, patchedMember)
		deletedMemberList = append(deletedMemberList, oldMember)
	}

	for _, id := range deletePrincipalIDList {
		deletedMember := oldMemberMap[id]
		memberDelete := &api.ProjectMemberDelete{
			ID:        deletedMember.ID,
			DeleterID: batchUpdate.UpdaterID,
		}
		err := deleteProjectMember(ctx, tx.PTx, memberDelete)
		if err != nil {
			return nil, nil, FormatError(err)
		}
		deletedMemberList = append(deletedMemberList, deletedMember)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, nil, FormatError(err)
	}

	return createdMemberList, deletedMemberList, nil
}

// createProjectMember creates a new projectMember.
func createProjectMember(ctx context.Context, tx *sql.Tx, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	// Insert row into database.
	if create.Payload == "" {
		create.Payload = "{}"
	}
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
	if v := find.RoleProvider; v != nil {
		where, args = append(where, fmt.Sprintf("role_provider = $%d", len(args)+1)), append(args, *v)
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
		payload := "{}"
		if *v == "" {
			payload = *v
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, api.Role(payload))
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
