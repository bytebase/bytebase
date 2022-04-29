package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// projectMemberRaw is the store model for an ProjectMember.
// Fields have exactly the same meanings as ProjectMember.
type projectMemberRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID int

	// Domain specific fields
	Role         string
	PrincipalID  int
	RoleProvider api.ProjectRoleProvider
	Payload      string
}

// toProjectMember creates an instance of ProjectMember based on the projectMemberRaw.
// This is intended to be called when we need to compose an ProjectMember relationship.
func (raw *projectMemberRaw) toProjectMember() *api.ProjectMember {
	return &api.ProjectMember{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID: raw.PrincipalID,

		// Domain specific fields
		Role:         raw.Role,
		PrincipalID:  raw.PrincipalID,
		RoleProvider: raw.RoleProvider,
		Payload:      raw.Payload,
	}
}

// CreateProjectMember creates an instance of ProjectMember
func (s *Store) CreateProjectMember(ctx context.Context, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	projectMemberRaw, err := s.createProjectMemberRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create ProjectMember with ProjectMemberCreate[%+v], error[%w]", create, err)
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectMember with projectMemberRaw[%+v], error[%w]", projectMemberRaw, err)
	}
	return projectMember, nil
}

// FindProjectMember finds a list of ProjectMember instances
func (s *Store) FindProjectMember(ctx context.Context, find *api.ProjectMemberFind) ([]*api.ProjectMember, error) {
	projectMemberRawList, err := s.findProjectMemberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find ProjectMember list with ProjectMemberFind[%+v], error[%w]", find, err)
	}
	var projectMemberList []*api.ProjectMember
	for _, raw := range projectMemberRawList {
		projectMember, err := s.composeProjectMember(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose ProjectMember with projectMemberRaw[%+v], error[%w]", raw, err)
		}
		projectMemberList = append(projectMemberList, projectMember)
	}
	return projectMemberList, nil
}

// GetProjectMemberByID gets an instance of ProjectMember
func (s *Store) GetProjectMemberByID(ctx context.Context, id int) (*api.ProjectMember, error) {
	find := &api.ProjectMemberFind{ID: &id}
	projectMemberRaw, err := s.getProjectMemberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get ProjectMember with ID[%d], error[%w]", id, err)
	}
	if projectMemberRaw == nil {
		return nil, nil
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectMember with projectMemberRaw[%+v], error[%w]", projectMemberRaw, err)
	}
	return projectMember, nil
}

// PatchProjectMember patches an instance of ProjectMember
func (s *Store) PatchProjectMember(ctx context.Context, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	projectMemberRaw, err := s.patchProjectMemberRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch ProjectMember with ProjectMemberPatch[%+v], error[%w]", patch, err)
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectMember with projectMemberRaw[%+v], error[%w]", projectMemberRaw, err)
	}
	return projectMember, nil
}

// DeleteProjectMember deletes an existing projectMember by ID.
func (s *Store) DeleteProjectMember(ctx context.Context, delete *api.ProjectMemberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteProjectMemberImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// BatchUpdateProjectMember update the project member with provided project member list
func (s *Store) BatchUpdateProjectMember(ctx context.Context, batchUpdate *api.ProjectMemberBatchUpdate) ([]*api.ProjectMember, []*api.ProjectMember, error) {
	createdMemberRawList, deletedMemberRawList, err := s.batchUpdateProjectMemberRaw(ctx, batchUpdate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to batch update projectMemberRaw with ProjectMemberBatchUpdate[%+v], error[%w]", batchUpdate, err)
	}
	var createdMemberList, deletedMemberList []*api.ProjectMember
	for _, raw := range createdMemberRawList {
		createdMember, err := s.composeProjectMember(ctx, raw)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compose created ProjectMember with projectMemberRaw[%+v], error[%w]", raw, err)
		}
		createdMemberList = append(createdMemberList, createdMember)
	}
	for _, raw := range deletedMemberRawList {
		deletedMember, err := s.composeProjectMember(ctx, raw)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compose deleted ProjectMember with projectMemberRaw[%+v], error[%w]", raw, err)
		}
		deletedMemberList = append(deletedMemberList, deletedMember)
	}
	return createdMemberList, deletedMemberList, nil
}

//
// private functions
//

// composeProjectMember composes an instance of ProjectMember by projectMemberRaw
func (s *Store) composeProjectMember(ctx context.Context, raw *projectMemberRaw) (*api.ProjectMember, error) {
	projectMember := raw.toProjectMember()

	creator, err := s.GetPrincipalByID(ctx, projectMember.CreatorID)
	if err != nil {
		return nil, err
	}
	projectMember.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, projectMember.UpdaterID)
	if err != nil {
		return nil, err
	}
	projectMember.Updater = updater

	principal, err := s.GetPrincipalByID(ctx, projectMember.PrincipalID)
	if err != nil {
		return nil, err
	}
	projectMember.Principal = principal

	return projectMember, nil
}

// createProjectMemberRaw creates a new projectMember.
func (s *Store) createProjectMemberRaw(ctx context.Context, create *api.ProjectMemberCreate) (*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectMember, err := createProjectMemberImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// findProjectMemberRaw retrieves a list of projectMembers based on find.
func (s *Store) findProjectMemberRaw(ctx context.Context, find *api.ProjectMemberFind) ([]*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectMemberImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getProjectMemberRaw finds project members.
func (s *Store) getProjectMemberRaw(ctx context.Context, find *api.ProjectMemberFind) (*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectMemberImpl(ctx, tx.PTx, find)
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

// patchProjectMemberRaw updates an existing projectMember by ID.
// Returns ENOTFOUND if projectMember does not exist.
func (s *Store) patchProjectMemberRaw(ctx context.Context, patch *api.ProjectMemberPatch) (*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectMember, err := patchProjectMemberImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// getBatchUpdatePrincipalIDList return the principal ID for each operation (this function may be a litter overhead, but it is easy to be tested)
func getBatchUpdatePrincipalIDList(oldPrincipalIDList []int, newPrincipalIDList []int) (createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList []int) {
	oldPrincipalIDSet := make(map[int]bool)
	for _, id := range oldPrincipalIDList {
		oldPrincipalIDSet[id] = true
	}

	newPrincipalIDSet := make(map[int]bool)
	for _, id := range newPrincipalIDList {
		newPrincipalIDSet[id] = true
	}

	for _, newID := range newPrincipalIDList {
		// if the ID exists, we will try to update it (NOTICE: a member with the same principal ID but different role provider will be considered as separate member)
		if _, ok := oldPrincipalIDSet[newID]; ok {
			patchPrincipalIDList = append(patchPrincipalIDList, newID)
		} else {
			createPrincipalIDList = append(createPrincipalIDList, newID)
		}
	}

	for _, oldID := range oldPrincipalIDList {
		// if the old ID also exists on the new id list, then it has already been added to the patch list above.
		if _, ok := newPrincipalIDSet[oldID]; ok {
			continue
		}
		deletePrincipalIDList = append(deletePrincipalIDList, oldID)
	}

	return createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList
}

// batchUpdateProjectMemberRaw update the project member with provided project member list
func (s *Store) batchUpdateProjectMemberRaw(ctx context.Context, batchUpdate *api.ProjectMemberBatchUpdate) ([]*projectMemberRaw, []*projectMemberRaw, error) {
	var createdMemberRawList []*projectMemberRaw
	var deletedMemberRawList []*projectMemberRaw
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
	oldProjectMemberRawList, err := findProjectMemberImpl(ctx, txRead.PTx, findProjectMember)
	if err != nil {
		return nil, nil, err
	}

	oldMemberRawMap := make(map[int]*projectMemberRaw)
	var oldPrincipalIDList []int
	for _, oldMemberRaw := range oldProjectMemberRawList {
		oldMemberRawMap[oldMemberRaw.PrincipalID] = oldMemberRaw
		oldPrincipalIDList = append(oldPrincipalIDList, oldMemberRaw.PrincipalID)
	}
	memberCreateMap := make(map[int]*api.ProjectMemberCreate)
	var newPrincipalIDList []int
	for _, newMember := range batchUpdate.List {
		memberCreateMap[newMember.PrincipalID] = newMember
		newPrincipalIDList = append(newPrincipalIDList, newMember.PrincipalID)
	}

	createPrincipalIDList, patchPrincipalIDList, deletePrincipalIDList := getBatchUpdatePrincipalIDList(oldPrincipalIDList, newPrincipalIDList)

	for _, id := range createPrincipalIDList {
		memberCreate := memberCreateMap[id]
		createdMember, err := createProjectMemberImpl(ctx, tx.PTx, memberCreate)
		if err != nil {
			return nil, nil, FormatError(err)
		}
		createdMemberRawList = append(createdMemberRawList, createdMember)
	}

	for _, id := range patchPrincipalIDList {
		oldMemberRaw := oldMemberRawMap[id]
		newMemberCreate := memberCreateMap[id]
		memberPatch := &api.ProjectMemberPatch{
			ID:           oldMemberRaw.ID,
			UpdaterID:    batchUpdate.UpdaterID,
			Role:         (*string)(&newMemberCreate.Role),
			RoleProvider: (*string)(&newMemberCreate.RoleProvider),
			Payload:      &newMemberCreate.Payload,
		}
		patchedMemberRaw, err := patchProjectMemberImpl(ctx, tx.PTx, memberPatch)
		if err != nil {
			return nil, nil, FormatError(err)
		}
		createdMemberRawList = append(createdMemberRawList, patchedMemberRaw)
		deletedMemberRawList = append(deletedMemberRawList, oldMemberRaw)
	}

	for _, id := range deletePrincipalIDList {
		deletedMember := oldMemberRawMap[id]
		memberDelete := &api.ProjectMemberDelete{
			ID:        deletedMember.ID,
			DeleterID: batchUpdate.UpdaterID,
		}
		if err := deleteProjectMemberImpl(ctx, tx.PTx, memberDelete); err != nil {
			return nil, nil, FormatError(err)
		}
		deletedMemberRawList = append(deletedMemberRawList, deletedMember)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, nil, FormatError(err)
	}

	return createdMemberRawList, deletedMemberRawList, nil
}

// createProjectMemberImpl creates a new projectMember.
func createProjectMemberImpl(ctx context.Context, tx *sql.Tx, create *api.ProjectMemberCreate) (*projectMemberRaw, error) {
	// Insert row into database.
	if create.Payload == "" {
		create.Payload = "{}"
	}
	if create.RoleProvider == "" {
		create.RoleProvider = api.ProjectRoleProviderBytebase
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
	var projectMemberRaw projectMemberRaw
	if err := row.Scan(
		&projectMemberRaw.ID,
		&projectMemberRaw.CreatorID,
		&projectMemberRaw.CreatedTs,
		&projectMemberRaw.UpdaterID,
		&projectMemberRaw.UpdatedTs,
		&projectMemberRaw.ProjectID,
		&projectMemberRaw.Role,
		&projectMemberRaw.PrincipalID,
		&projectMemberRaw.RoleProvider,
		&projectMemberRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &projectMemberRaw, nil
}

func findProjectMemberImpl(ctx context.Context, tx *sql.Tx, find *api.ProjectMemberFind) ([]*projectMemberRaw, error) {
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

	// Iterate over result set and deserialize rows into projectMemberRawList.
	var projectMemberRawList []*projectMemberRaw
	for rows.Next() {
		var projectMemberRaw projectMemberRaw
		if err := rows.Scan(
			&projectMemberRaw.ID,
			&projectMemberRaw.CreatorID,
			&projectMemberRaw.CreatedTs,
			&projectMemberRaw.UpdaterID,
			&projectMemberRaw.UpdatedTs,
			&projectMemberRaw.ProjectID,
			&projectMemberRaw.Role,
			&projectMemberRaw.PrincipalID,
			&projectMemberRaw.RoleProvider,
			&projectMemberRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		projectMemberRawList = append(projectMemberRawList, &projectMemberRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return projectMemberRawList, nil
}

// patchProjectMemberImpl updates a projectMember by ID. Returns the new state of the projectMember after update.
func patchProjectMemberImpl(ctx context.Context, tx *sql.Tx, patch *api.ProjectMemberPatch) (*projectMemberRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.RoleProvider; v != nil {
		set, args = append(set, fmt.Sprintf("role_provider = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		payload := "{}"
		if *v == "" {
			payload = *v
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, payload)
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
		var projectMemberRaw projectMemberRaw
		if err := row.Scan(
			&projectMemberRaw.ID,
			&projectMemberRaw.CreatorID,
			&projectMemberRaw.CreatedTs,
			&projectMemberRaw.UpdaterID,
			&projectMemberRaw.UpdatedTs,
			&projectMemberRaw.ProjectID,
			&projectMemberRaw.Role,
			&projectMemberRaw.PrincipalID,
			&projectMemberRaw.RoleProvider,
			&projectMemberRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &projectMemberRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project member ID not found: %d", patch.ID)}
}

// deleteProjectMemberImpl permanently deletes a projectMember by ID.
func deleteProjectMemberImpl(ctx context.Context, tx *sql.Tx, delete *api.ProjectMemberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
