package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

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
	Role        string
	PrincipalID int
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
		Role:        raw.Role,
		PrincipalID: raw.PrincipalID,
	}
}

// CreateProjectMember creates an instance of ProjectMember.
func (s *Store) CreateProjectMember(ctx context.Context, create *api.ProjectMemberCreate) (*api.ProjectMember, error) {
	projectMemberRaw, err := s.createProjectMemberRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ProjectMember with ProjectMemberCreate[%+v]", create)
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ProjectMember with projectMemberRaw[%+v]", projectMemberRaw)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(projectMemberCacheNamespace, create.ProjectID)

	return projectMember, nil
}

// FindProjectMember finds a list of ProjectMember instances.
func (s *Store) FindProjectMember(ctx context.Context, find *api.ProjectMemberFind) ([]*api.ProjectMember, error) {
	findCopy := *find
	findCopy.ProjectID = nil
	isListProjectMember := find.ProjectID != nil && findCopy == api.ProjectMemberFind{}
	var cacheList []*api.ProjectMember
	has, err := s.cache.FindCache(projectMemberCacheNamespace, *find.ProjectID, &cacheList)
	if err != nil {
		return nil, err
	}
	if has && isListProjectMember {
		return cacheList, nil
	}

	projectMemberRawList, err := s.findProjectMemberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find ProjectMember list with ProjectMemberFind[%+v]", find)
	}
	var projectMemberList []*api.ProjectMember
	for _, raw := range projectMemberRawList {
		projectMember, err := s.composeProjectMember(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose ProjectMember with projectMemberRaw[%+v]", raw)
		}
		projectMemberList = append(projectMemberList, projectMember)
	}
	if isListProjectMember {
		if err := s.cache.UpsertCache(projectMemberCacheNamespace, *find.ProjectID, projectMemberList); err != nil {
			return nil, err
		}
	}
	return projectMemberList, nil
}

// GetProjectMember gets an instance of ProjectMember.
func (s *Store) GetProjectMember(ctx context.Context, find *api.ProjectMemberFind) (*api.ProjectMember, error) {
	projectMemberRaw, err := s.getProjectMemberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ProjectMember with projectMemberFind %+v", find)
	}
	if projectMemberRaw == nil {
		return nil, nil
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ProjectMember with projectMemberRaw %+v", projectMemberRaw)
	}
	return projectMember, nil
}

// GetProjectMemberByID gets an instance of ProjectMember by ID.
func (s *Store) GetProjectMemberByID(ctx context.Context, id int) (*api.ProjectMember, error) {
	find := &api.ProjectMemberFind{ID: &id}
	projectMember, err := s.GetProjectMember(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ProjectMember with ID %d", id)
	}
	return projectMember, nil
}

// PatchProjectMember patches an instance of ProjectMember.
func (s *Store) PatchProjectMember(ctx context.Context, patch *api.ProjectMemberPatch) (*api.ProjectMember, error) {
	projectMemberRaw, err := s.patchProjectMemberRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch ProjectMember with ProjectMemberPatch[%+v]", patch)
	}
	projectMember, err := s.composeProjectMember(ctx, projectMemberRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ProjectMember with projectMemberRaw[%+v]", projectMemberRaw)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(projectMemberCacheNamespace, projectMemberRaw.ProjectID)
	return projectMember, nil
}

// DeleteProjectMember deletes an existing projectMember by ID.
func (s *Store) DeleteProjectMember(ctx context.Context, delete *api.ProjectMemberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	if err := s.deleteProjectMemberImpl(ctx, tx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}
	// Invalidate the cache.
	s.cache.DeleteCache(projectMemberCacheNamespace, delete.ProjectID)

	return nil
}

//
// private functions
//

// composeProjectMember composes an instance of ProjectMember by projectMemberRaw.
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
	defer tx.Rollback()

	projectMember, err := createProjectMemberImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// findProjectMemberRaw retrieves a list of projectMembers based on find.
func (s *Store) findProjectMemberRaw(ctx context.Context, find *api.ProjectMemberFind) ([]*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectMemberImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getProjectMemberRaw finds project members.
func (s *Store) getProjectMemberRaw(ctx context.Context, find *api.ProjectMemberFind) (*projectMemberRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectMemberImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d project members with filter %+v, expect 1", len(list), find)}
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
	defer tx.Rollback()

	projectMember, err := patchProjectMemberImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectMember, nil
}

// createProjectMemberImpl creates a new projectMember.
func createProjectMemberImpl(ctx context.Context, tx *Tx, create *api.ProjectMemberCreate) (*projectMemberRaw, error) {
	// Insert row into database.
	query := `
		INSERT INTO project_member (
			creator_id,
			updater_id,
			project_id,
			role,
			principal_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`
	var projectMemberRaw projectMemberRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Role,
		create.PrincipalID,
	).Scan(
		&projectMemberRaw.ID,
		&projectMemberRaw.CreatorID,
		&projectMemberRaw.CreatedTs,
		&projectMemberRaw.UpdaterID,
		&projectMemberRaw.UpdatedTs,
		&projectMemberRaw.ProjectID,
		&projectMemberRaw.Role,
		&projectMemberRaw.PrincipalID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &projectMemberRaw, nil
}

func findProjectMemberImpl(ctx context.Context, tx *Tx, find *api.ProjectMemberFind) ([]*projectMemberRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("principal_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Role; v != nil {
		where, args = append(where, fmt.Sprintf("role = $%d", len(args)+1)), append(args, *v)
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
func patchProjectMemberImpl(ctx context.Context, tx *Tx, patch *api.ProjectMemberPatch) (*projectMemberRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Role; v != nil {
		set, args = append(set, fmt.Sprintf("role = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ID)

	var projectMemberRaw projectMemberRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE project_member
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, role, principal_id
	`, len(args)),
		args...,
	).Scan(
		&projectMemberRaw.ID,
		&projectMemberRaw.CreatorID,
		&projectMemberRaw.CreatedTs,
		&projectMemberRaw.UpdaterID,
		&projectMemberRaw.UpdatedTs,
		&projectMemberRaw.ProjectID,
		&projectMemberRaw.Role,
		&projectMemberRaw.PrincipalID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project member ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &projectMemberRaw, nil
}

// deleteProjectMemberImpl permanently deletes a projectMember by ID.
func (*Store) deleteProjectMemberImpl(ctx context.Context, tx *Tx, delete *api.ProjectMemberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_member WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}

// IAMPolicyMessage is the IAM policy of a project.
type IAMPolicyMessage struct {
	Bindings []*PolicyBinding
}

// PolicyBinding is the IAM policy binding of a project.
type PolicyBinding struct {
	Role    api.Role
	Members []*UserMessage
}

// GetProjectPolicyMessage is the message to get project policy.
type GetProjectPolicyMessage struct {
	ProjectID *string
	UID       *int
}

// GetProjectPolicy gets the IAM policy of a project.
func (s *Store) GetProjectPolicy(ctx context.Context, find *GetProjectPolicyMessage) (*IAMPolicyMessage, error) {
	if find.ProjectID == nil && find.UID == nil {
		return nil, errors.Errorf("GetProjectPolicy must set either resource ID or UID")
	}
	projectFind := &FindProjectMessage{}
	if v := find.ProjectID; v != nil {
		projectFind.ResourceID = v
	}
	if v := find.UID; v != nil {
		projectFind.UID = v
	}
	project, err := s.GetProjectV2(ctx, projectFind)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.Errorf("cannot find project with projectFind %v", projectFind)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectPolicy, err := s.getProjectPolicyImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectPolicy, nil
}

// SetProjectIAMPolicy sets the IAM policy of a project.
func (s *Store) SetProjectIAMPolicy(ctx context.Context, set *IAMPolicyMessage, creatorUID int, projectUID int) error {
	if set == nil {
		return errors.Errorf("SetProjectPolicy must set IAMPolicyMessage")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	if err := s.setProjectIAMPolicyImpl(ctx, tx, set, creatorUID, projectUID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) getProjectPolicyImpl(ctx context.Context, tx *Tx, find *GetProjectPolicyMessage) (*IAMPolicyMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("project_member.row_status = $%d", len(args)+1)), append(args, api.Normal)
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("project.id = $%d", len(args)+1)), append(args, *v)
	}

	roleMap := make(map[api.Role][]*UserMessage)
	rows, err := tx.QueryContext(ctx, `
			SELECT
				project_member.principal_id,
				project_member.role
			FROM project_member
			LEFT JOIN project ON project_member.project_id = project.id
			WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var role api.Role
		member := &UserMessage{}
		if err := rows.Scan(
			&member.ID,
			&role,
		); err != nil {
			return nil, FormatError(err)
		}
		roleMap[role] = append(roleMap[role], member)
	}

	var roles []api.Role
	for role := range roleMap {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool {
		return string(roles[i]) < string(roles[j])
	})
	projectPolicy := &IAMPolicyMessage{}
	for _, role := range roles {
		binding := &PolicyBinding{Role: role}
		for _, member := range roleMap[role] {
			user, err := s.GetUserByID(ctx, member.ID)
			if err != nil {
				return nil, err
			}
			binding.Members = append(binding.Members, user)
		}
		projectPolicy.Bindings = append(projectPolicy.Bindings, binding)
	}
	return projectPolicy, nil
}

func (s *Store) setProjectIAMPolicyImpl(ctx context.Context, tx *Tx, set *IAMPolicyMessage, creatorUID int, projectUID int) error {
	if set == nil {
		return errors.Errorf("SetProjectPolicy must set IAMPolicyMessage")
	}
	oldPolicy, err := s.getProjectPolicyImpl(ctx, tx, &GetProjectPolicyMessage{
		UID: &projectUID,
	})
	if err != nil {
		return err
	}
	deletes, inserts := getIAMPolicyDiff(oldPolicy, set)

	if len(deletes) > 0 {
		if err := s.deleteProjectIAMPolicyImpl(ctx, tx, projectUID, deletes); err != nil {
			return err
		}
	}

	if len(inserts.Bindings) > 0 {
		args := []interface{}{}
		var placeholders []string
		for _, binding := range inserts.Bindings {
			for _, member := range binding.Members {
				placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5))
				args = append(args,
					creatorUID,   // creator_id
					creatorUID,   // updater_id
					projectUID,   // project_id
					binding.Role, // role
					member.ID,    // principal_id
				)
			}
		}
		query := fmt.Sprintf(`INSERT INTO project_member (
			creator_id,
			updater_id,
			project_id,
			role,
			principal_id
		) VALUES %s`, strings.Join(placeholders, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return FormatError(err)
		}
	}
	return nil
}

func (*Store) deleteProjectIAMPolicyImpl(ctx context.Context, tx *Tx, projectUID int, memberIDs []int) error {
	if len(memberIDs) == 0 {
		return nil
	}
	query := ""
	where, deletePlaceholders, args := []string{}, []string{}, []interface{}{}
	where, args = append(where, fmt.Sprintf("(project_member.project_id = $%d)", len(args)+1)), append(args, projectUID)
	for _, id := range memberIDs {
		deletePlaceholders = append(deletePlaceholders, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, id)
	}
	where = append(where, fmt.Sprintf("(project_member.principal_id IN (%s))", strings.Join(deletePlaceholders, ", ")))
	query = `DELETE FROM project_member WHERE ` + strings.Join(where, " AND ")
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return FormatError(err)
	}
	return nil
}

func getIAMPolicyDiff(oldPolicy *IAMPolicyMessage, newPolicy *IAMPolicyMessage) ([]int, *IAMPolicyMessage) {
	oldUserIDToProjectRoleMap := make(map[int]api.Role)
	newUserIDToProjectRoleMap := make(map[int]api.Role)
	newUserIDToUserMap := make(map[int]*UserMessage)

	for _, binding := range oldPolicy.Bindings {
		for _, member := range binding.Members {
			oldUserIDToProjectRoleMap[member.ID] = binding.Role
		}
	}
	for _, binding := range newPolicy.Bindings {
		for _, member := range binding.Members {
			newUserIDToProjectRoleMap[member.ID] = binding.Role
			newUserIDToUserMap[member.ID] = member
		}
	}

	var deletes []int
	inserts := make(map[api.Role][]*UserMessage)
	// Delete member that no longer exist or role doesn't match.
	for oldUserID, oldRole := range oldUserIDToProjectRoleMap {
		if newRole, ok := newUserIDToProjectRoleMap[oldUserID]; !ok || oldRole != newRole {
			deletes = append(deletes, oldUserID)
		}
	}

	// Create member if not exist in old policy or project role doesn't match.
	for newUserID, newRole := range newUserIDToProjectRoleMap {
		if oldRole, ok := oldUserIDToProjectRoleMap[newUserID]; !ok || newRole != oldRole {
			inserts[newRole] = append(inserts[newRole], newUserIDToUserMap[newUserID])
		}
	}

	var upsertBindings []*PolicyBinding
	for role, users := range inserts {
		upsertBindings = append(upsertBindings, &PolicyBinding{
			Role:    role,
			Members: users,
		})
	}
	return deletes, &IAMPolicyMessage{
		Bindings: upsertBindings,
	}
}
