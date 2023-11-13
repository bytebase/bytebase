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

func (m *IAMPolicyMessage) String() string {
	var bindings []string
	for _, binding := range m.Bindings {
		var members []string
		for _, member := range binding.Members {
			members = append(members, member.Email)
		}
		if binding.Condition == nil {
			binding.Condition = &expr.Expr{}
		}
		bindings = append(bindings, fmt.Sprintf("[%s] condition %s: %s", binding.Role, binding.Condition, strings.Join(members, ", ")))
	}
	return fmt.Sprintf("policy:\n%s\n\n", strings.Join(bindings, "\n"))
}

func formatCondition(condition *expr.Expr) (string, error) {
	if condition == nil {
		return "{}", nil
	}
	bytes, err := protojson.Marshal(condition)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (m *IAMPolicyMessage) sort() error {
	sort.Slice(m.Bindings, func(i, j int) bool {
		if m.Bindings[i].Role < m.Bindings[j].Role {
			return true
		}
		str1, err := formatCondition(m.Bindings[i].Condition)
		if err != nil {
			return false
		}
		str2, err := formatCondition(m.Bindings[j].Condition)
		if err != nil {
			return false
		}
		if m.Bindings[i].Role == m.Bindings[j].Role && str1 < str2 {
			return true
		}
		return false
	})
	for _, binding := range m.Bindings {
		sort.Slice(binding.Members, func(i, j int) bool {
			return binding.Members[i].ID < binding.Members[j].ID
		})
	}
	return nil
}

// PolicyBinding is the IAM policy binding of a project.
type PolicyBinding struct {
	Role      api.Role
	Members   []*UserMessage
	Condition *expr.Expr
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
		if v, ok := s.projectPolicyCache.Get(*find.ProjectID); ok {
			return v, nil
		}
	}
	if find.UID != nil {
		if v, ok := s.projectIDPolicyCache.Get(*find.UID); ok {
			return v, nil
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

	s.projectPolicyCache.Add(project.ResourceID, projectPolicy)
	s.projectIDPolicyCache.Add(project.UID, projectPolicy)
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

	s.projectPolicyCache.Remove(project.ResourceID)
	s.projectIDPolicyCache.Remove(project.UID)
	return s.GetProjectPolicy(ctx, &GetProjectPolicyMessage{UID: &projectUID})
}

type roleConditionMapKey struct {
	role      api.Role
	condition string
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

	roleMap := map[roleConditionMapKey][]int{}
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
		var rawCondition string
		var userUID int
		if err := rows.Scan(
			&userUID,
			&role,
			&rawCondition,
		); err != nil {
			return nil, err
		}
		key := roleConditionMapKey{role: role, condition: rawCondition}
		roleMap[key] = append(roleMap[key], userUID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	projectPolicy := &IAMPolicyMessage{}
	for key, userUIDs := range roleMap {
		var condition expr.Expr
		if err := protojson.Unmarshal([]byte(key.condition), &condition); err != nil {
			return nil, err
		}
		binding := &PolicyBinding{Role: key.role, Condition: &condition}
		for _, userUID := range userUIDs {
			user, err := s.GetUserByID(ctx, userUID)
			if err != nil {
				return nil, err
			}
			binding.Members = append(binding.Members, user)
		}
		projectPolicy.Bindings = append(projectPolicy.Bindings, binding)
	}
	if err := projectPolicy.sort(); err != nil {
		return nil, err
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
	// Deletes and inserts don't have condition in *expr.Expr because we use rawCondition string for updates.
	deletes, inserts, err := GetIAMPolicyDiff(oldPolicy, set)
	if err != nil {
		return err
	}
	if len(deletes.Bindings) > 0 {
		if err := s.deleteProjectIAMPolicyImpl(ctx, tx, projectUID, deletes); err != nil {
			return err
		}
	}

	if len(inserts.Bindings) > 0 {
		args := []any{}
		var placeholders []string
		for _, binding := range inserts.Bindings {
			rawCondition, err := formatCondition(binding.Condition)
			if err != nil {
				return err
			}

			for _, member := range binding.Members {
				placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5, len(args)+6))
				args = append(args,
					creatorUID,   // creator_id
					creatorUID,   // updater_id
					projectUID,   // project_id
					binding.Role, // role
					member.ID,    // principal_id
					rawCondition,
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
		rawCondition, err := formatCondition(binding.Condition)
		if err != nil {
			return err
		}
		for _, member := range binding.Members {
			where = append(where, fmt.Sprintf("(project_member.principal_id = $%d AND project_member.role = $%d AND project_member.condition = $%d)", len(args)+1, len(args)+2, len(args)+3))
			args = append(args, member.ID, binding.Role, rawCondition)
		}
	}
	query := fmt.Sprintf(`DELETE FROM project_member WHERE project_member.project_id = $1 AND (%s)`, strings.Join(where, " OR "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// GetIAMPolicyDiff returns the diff between old and new policy (remove and add).
// TODO(d): make minimal diff.
func GetIAMPolicyDiff(oldPolicy *IAMPolicyMessage, newPolicy *IAMPolicyMessage) (*IAMPolicyMessage, *IAMPolicyMessage, error) {
	oldMap, newMap := make(map[roleConditionMapKey][]*UserMessage), make(map[roleConditionMapKey][]*UserMessage)

	for _, binding := range oldPolicy.Bindings {
		str, err := formatCondition(binding.Condition)
		if err != nil {
			return nil, nil, err
		}
		key := roleConditionMapKey{role: binding.Role, condition: str}
		if oldMap[key] != nil {
			oldMap[key] = filterUser(append(oldMap[key], binding.Members...))
		} else {
			oldMap[key] = binding.Members
		}
	}
	for _, binding := range newPolicy.Bindings {
		str, err := formatCondition(binding.Condition)
		if err != nil {
			return nil, nil, err
		}
		key := roleConditionMapKey{role: binding.Role, condition: str}
		if newMap[key] != nil {
			newMap[key] = filterUser(append(newMap[key], binding.Members...))
		} else {
			newMap[key] = binding.Members
		}
	}

	remove, add := &IAMPolicyMessage{}, &IAMPolicyMessage{}
	// Delete member that no longer exists.
	for key, oldMembers := range oldMap {
		newMembers, ok := newMap[key]
		var condition expr.Expr
		if err := protojson.Unmarshal([]byte(key.condition), &condition); err != nil {
			return nil, nil, err
		}
		if !ok {
			remove.Bindings = append(remove.Bindings, &PolicyBinding{
				Role:      key.role,
				Condition: &condition,
				Members:   oldMembers,
			})
		} else {
			// Reconcile members.
			oldUserMap, newUserMap := make(map[int]*UserMessage), make(map[int]*UserMessage)
			for _, oldMember := range oldMembers {
				oldUserMap[oldMember.ID] = oldMember
			}
			for _, newMember := range newMembers {
				newUserMap[newMember.ID] = newMember
			}
			var removeMembers, addMembers []*UserMessage
			for oldID, oldMember := range oldUserMap {
				if _, ok := newUserMap[oldID]; !ok {
					removeMembers = append(removeMembers, oldMember)
				}
			}
			for newID, newMember := range newUserMap {
				if _, ok := oldUserMap[newID]; !ok {
					addMembers = append(addMembers, newMember)
				}
			}
			if len(removeMembers) > 0 {
				remove.Bindings = append(remove.Bindings, &PolicyBinding{
					Role:      key.role,
					Condition: &condition,
					Members:   removeMembers,
				})
			}
			if len(addMembers) > 0 {
				add.Bindings = append(add.Bindings, &PolicyBinding{
					Role:      key.role,
					Condition: &condition,
					Members:   addMembers,
				})
			}
		}
	}

	// Create member if not exist in old policy.
	for key, members := range newMap {
		if _, ok := oldMap[key]; !ok {
			var condition expr.Expr
			if err := protojson.Unmarshal([]byte(key.condition), &condition); err != nil {
				return nil, nil, err
			}
			add.Bindings = append(add.Bindings, &PolicyBinding{
				Role:      key.role,
				Condition: &condition,
				Members:   members,
			})
		}
	}
	if err := remove.sort(); err != nil {
		return nil, nil, err
	}
	if err := add.sort(); err != nil {
		return nil, nil, err
	}

	return remove, add, nil
}

func filterUser(list []*UserMessage) []*UserMessage {
	result, f := []*UserMessage{}, make(map[int]bool, len(list))
	for _, user := range list {
		if f[user.ID] {
			continue
		}
		result, f[user.ID] = append(result, user), true
	}
	return result
}
