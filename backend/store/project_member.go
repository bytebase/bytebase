package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// IAMPolicyMessage is the IAM policy of a project.
type IAMPolicyMessage struct {
	Bindings []*PolicyBinding
}

// PolicyBinding is the IAM policy binding of a project.
type PolicyBinding struct {
	Role      api.Role
	Members   []*UserMessage
	Condition *expr.Expr
	// We keep the raw condition to be compatible with the json format string in the store.
	rawCondition string
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

// GetProjectUsingRole gets a project that uses the role.
func (s *Store) GetProjectUsingRole(ctx context.Context, role api.Role) (bool, string, error) {
	query := `
		SELECT project.resource_id
		FROM project_member, project
		WHERE project_member.role = $1 AND project_member.project_id = project.id
		LIMIT 1
	`
	var project string
	if err := s.db.db.QueryRowContext(ctx, query, role).Scan(&project); err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, project, nil
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

type roleConditionKey struct {
	role         api.Role
	rawCondition string
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

	roleMap := make(map[roleConditionKey][]*UserMessage)
	rows, err := tx.QueryContext(ctx, `
			SELECT
				project_member.principal_id,
				project_member.role,
				project_member.condition
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
		var condition string
		member := &UserMessage{}
		if err := rows.Scan(
			&member.ID,
			&role,
			&condition,
		); err != nil {
			return nil, err
		}
		key := roleConditionKey{role, condition}
		roleMap[key] = append(roleMap[key], member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var keys []roleConditionKey
	for role := range roleMap {
		keys = append(keys, role)
	}
	sort.Slice(keys, func(i, j int) bool {
		if string(keys[i].role) < string(keys[j].role) {
			return true
		}
		if string(keys[i].role) == string(keys[j].role) && keys[i].rawCondition < keys[j].rawCondition {
			return true
		}
		return false
	})
	projectPolicy := &IAMPolicyMessage{}
	for _, key := range keys {
		var condition expr.Expr
		if err := protojson.Unmarshal([]byte(key.rawCondition), &condition); err != nil {
			return nil, err
		}

		binding := &PolicyBinding{Role: key.role, Condition: &condition, rawCondition: key.rawCondition}
		for _, member := range roleMap[key] {
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
	// Convert condition to json string format.
	for _, binding := range set.Bindings {
		if binding.rawCondition == "" {
			if binding.Condition == nil {
				binding.Condition = &expr.Expr{}
			}
			bytes, err := protojson.Marshal(binding.Condition)
			if err != nil {
				return err
			}
			binding.rawCondition = string(bytes)
		}
	}

	oldPolicy, err := s.getProjectPolicyImpl(ctx, tx, &GetProjectPolicyMessage{
		UID: &projectUID,
	})
	if err != nil {
		return err
	}
	// Deletes and inserts don't have condition in *expr.Expr because we use rawCondition string for updates.
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
				placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5, len(args)+6))
				args = append(args,
					creatorUID,   // creator_id
					creatorUID,   // updater_id
					projectUID,   // project_id
					binding.Role, // role
					member.ID,    // principal_id
					binding.rawCondition,
				)
			}
		}
		query := fmt.Sprintf(`INSERT INTO project_member (
			creator_id,
			updater_id,
			project_id,
			role,
			principal_id,
			condition
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
			where = append(where, fmt.Sprintf("(project_member.principal_id = $%d AND project_member.role = $%d AND project_member.condition = $%d)", len(args)+1, len(args)+2, len(args)+3))
			args = append(args, member.ID, binding.Role, binding.rawCondition)
		}
	}
	query := fmt.Sprintf(`DELETE FROM project_member WHERE project_member.project_id = $1 AND (%s)`, strings.Join(where, " OR "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

func getIAMPolicyDiff(oldPolicy *IAMPolicyMessage, newPolicy *IAMPolicyMessage) (*IAMPolicyMessage, *IAMPolicyMessage) {
	oldUserIDToProjectRoleMap := make(map[int]map[roleConditionKey]bool)
	oldUserIDToUserMap := make(map[int]*UserMessage)
	newUserIDToProjectRoleMap := make(map[int]map[roleConditionKey]bool)
	newUserIDToUserMap := make(map[int]*UserMessage)

	for _, binding := range oldPolicy.Bindings {
		key := roleConditionKey{role: binding.Role, rawCondition: binding.rawCondition}
		for _, member := range binding.Members {
			oldUserIDToUserMap[member.ID] = member
			if _, ok := oldUserIDToProjectRoleMap[member.ID]; !ok {
				oldUserIDToProjectRoleMap[member.ID] = make(map[roleConditionKey]bool)
			}
			oldUserIDToProjectRoleMap[member.ID][key] = true
		}
	}
	for _, binding := range newPolicy.Bindings {
		key := roleConditionKey{role: binding.Role, rawCondition: binding.rawCondition}
		for _, member := range binding.Members {
			newUserIDToUserMap[member.ID] = member
			if _, ok := newUserIDToProjectRoleMap[member.ID]; !ok {
				newUserIDToProjectRoleMap[member.ID] = make(map[roleConditionKey]bool)
			}
			newUserIDToProjectRoleMap[member.ID][key] = true
		}
	}

	deletes := make(map[roleConditionKey][]*UserMessage)
	inserts := make(map[roleConditionKey][]*UserMessage)
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
	for rc, users := range deletes {
		deleteBindings = append(deleteBindings, &PolicyBinding{
			Role:    rc.role,
			Members: users,
			// We escape the condition because it's not useful for updates.
			rawCondition: rc.rawCondition,
		})
	}

	var upsertBindings []*PolicyBinding
	for rc, users := range inserts {
		upsertBindings = append(upsertBindings, &PolicyBinding{
			Role:    rc.role,
			Members: users,
			// We escape the condition because it's not useful for updates.
			rawCondition: rc.rawCondition,
		})
	}

	return &IAMPolicyMessage{
			Bindings: deleteBindings,
		}, &IAMPolicyMessage{
			Bindings: upsertBindings,
		}
}
