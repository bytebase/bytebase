package store

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
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
