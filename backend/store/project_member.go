package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

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

// TODO(zp): Do not expose the project member after migrating to V1 API.

// ProjectMemberMessage is the message to create a project member.
type ProjectMemberMessage struct {
	ID          int
	ProjectID   int
	PrincipalID int
}

// GetProjectMemberByProjectIDAndPrincipalIDAndRole gets a project member by project ID and principal ID.
func (s *Store) GetProjectMemberByProjectIDAndPrincipalIDAndRole(ctx context.Context, projectID int, principalID int, role api.Role) (*ProjectMemberMessage, error) {
	var projectMember ProjectMemberMessage
	query := `
	SELECT
		project_member.id,
		project_member.project_id,
		project_member.principal_id
	FROM project_member 
	WHERE project_member.project_id = $1 AND project_member.principal_id = $2 AND project_member.role = $3`
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, projectID, principalID, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&projectMember.ID, &projectMember.ProjectID, &projectMember.PrincipalID); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &projectMember, nil
}

// GetProjectMemberByID gets a project member by ID.
func (s *Store) GetProjectMemberByID(ctx context.Context, projectMemberID int) (*ProjectMemberMessage, error) {
	var projectMember ProjectMemberMessage
	query := `
	SELECT
		project_member.id,
		project_member.project_id,
		project_member.principal_id
	FROM project_member 
	WHERE project_member.id = $1`
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, projectMemberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&projectMember.ID, &projectMember.ProjectID, &projectMember.PrincipalID); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &projectMember, nil
}

// GetProjectPolicy gets the IAM policy of a project.
func (s *Store) GetProjectPolicy(ctx context.Context, find *GetProjectPolicyMessage) (*IAMPolicyMessage, error) {
	if find.ProjectID == nil && find.UID == nil {
		return nil, errors.Errorf("GetProjectPolicy must set either resource ID or UID")
	}
	if find.ProjectID != nil {
		if policy, ok := s.projectPolicyCache.Load(*find.ProjectID); ok {
			return policy.(*IAMPolicyMessage), nil
		}
	}
	if find.UID != nil {
		if policy, ok := s.projectIDPolicyCache.Load(*find.UID); ok {
			return policy.(*IAMPolicyMessage), nil
		}
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
		return nil, err
	}
	defer tx.Rollback()

	projectPolicy, err := s.getProjectPolicyImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.projectPolicyCache.Store(project.ResourceID, projectPolicy)
	s.projectIDPolicyCache.Store(project.UID, projectPolicy)
	return projectPolicy, nil
}

// SetProjectIAMPolicy sets the IAM policy of a project.
func (s *Store) SetProjectIAMPolicy(ctx context.Context, set *IAMPolicyMessage, creatorUID int, projectUID int) (*IAMPolicyMessage, error) {
	if set == nil {
		return nil, errors.Errorf("SetProjectPolicy must set IAMPolicyMessage")
	}
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &projectUID})
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := s.setProjectIAMPolicyImpl(ctx, tx, set, creatorUID, projectUID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.projectPolicyCache.Delete(project.ResourceID)
	s.projectIDPolicyCache.Delete(project.UID)
	return s.GetProjectPolicy(ctx, &GetProjectPolicyMessage{UID: &projectUID})
}

func (s *Store) getProjectPolicyImpl(ctx context.Context, tx *Tx, find *GetProjectPolicyMessage) (*IAMPolicyMessage, error) {
	where, args := []string{"TRUE"}, []any{}
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
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role api.Role
		member := &UserMessage{}
		if err := rows.Scan(
			&member.ID,
			&role,
		); err != nil {
			return nil, err
		}
		roleMap[role] = append(roleMap[role], member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

	if len(deletes.Bindings) > 0 {
		if err := s.deleteProjectIAMPolicyImpl(ctx, tx, projectUID, deletes); err != nil {
			return err
		}
	}

	if len(inserts.Bindings) > 0 {
		args := []any{}
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
			return err
		}
	}
	return nil
}

func (*Store) deleteProjectIAMPolicyImpl(ctx context.Context, tx *Tx, projectUID int, deletes *IAMPolicyMessage) error {
	if len(deletes.Bindings) == 0 {
		return nil
	}
	where, args := []string{}, []any{}
	args = append(args, projectUID)
	for _, binding := range deletes.Bindings {
		for _, member := range binding.Members {
			where = append(where, fmt.Sprintf("(project_member.principal_id = $%d AND project_member.role = $%d)", len(args)+1, len(args)+2))
			args = append(args, member.ID, binding.Role)
		}
	}
	query := fmt.Sprintf(`DELETE FROM project_member WHERE project_member.project_id = $1 AND (%s)`, strings.Join(where, " OR "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

func getIAMPolicyDiff(oldPolicy *IAMPolicyMessage, newPolicy *IAMPolicyMessage) (*IAMPolicyMessage, *IAMPolicyMessage) {
	oldUserIDToProjectRoleMap := make(map[int]map[api.Role]bool)
	oldUserIDToUserMap := make(map[int]*UserMessage)
	newUserIDToProjectRoleMap := make(map[int]map[api.Role]bool)
	newUserIDToUserMap := make(map[int]*UserMessage)

	for _, binding := range oldPolicy.Bindings {
		for _, member := range binding.Members {
			oldUserIDToUserMap[member.ID] = member

			if _, ok := oldUserIDToProjectRoleMap[member.ID]; !ok {
				oldUserIDToProjectRoleMap[member.ID] = make(map[api.Role]bool)
			}
			oldUserIDToProjectRoleMap[member.ID][binding.Role] = true
		}
	}
	for _, binding := range newPolicy.Bindings {
		for _, member := range binding.Members {
			newUserIDToUserMap[member.ID] = member

			if _, ok := newUserIDToProjectRoleMap[member.ID]; !ok {
				newUserIDToProjectRoleMap[member.ID] = make(map[api.Role]bool)
			}
			newUserIDToProjectRoleMap[member.ID][binding.Role] = true
		}
	}

	deletes := make(map[api.Role][]*UserMessage)
	inserts := make(map[api.Role][]*UserMessage)
	// Delete member that no longer exists.
	for oldUserID, oldRoleMap := range oldUserIDToProjectRoleMap {
		for oldRole := range oldRoleMap {
			if newRoleMap, ok := newUserIDToProjectRoleMap[oldUserID]; !ok || !newRoleMap[oldRole] {
				deletes[oldRole] = append(deletes[oldRole], oldUserIDToUserMap[oldUserID])
			}
		}
	}

	// Create member if not exist in old policy.
	for newUserID, newRoleMap := range newUserIDToProjectRoleMap {
		for newRole := range newRoleMap {
			if oldRoleMap, ok := oldUserIDToProjectRoleMap[newUserID]; !ok || !oldRoleMap[newRole] {
				inserts[newRole] = append(inserts[newRole], newUserIDToUserMap[newUserID])
			}
		}
	}

	var deleteBindings []*PolicyBinding
	for role, users := range deletes {
		deleteBindings = append(deleteBindings, &PolicyBinding{
			Role:    role,
			Members: users,
		})
	}

	var upsertBindings []*PolicyBinding
	for role, users := range inserts {
		upsertBindings = append(upsertBindings, &PolicyBinding{
			Role:    role,
			Members: users,
		})
	}

	return &IAMPolicyMessage{
			Bindings: deleteBindings,
		}, &IAMPolicyMessage{
			Bindings: upsertBindings,
		}
}
