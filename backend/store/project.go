package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ProjectMessage is the message for project.
type ProjectMessage struct {
	ResourceID                 string
	Title                      string
	Webhooks                   []*ProjectWebhookMessage
	DataClassificationConfigID string
	Setting                    *storepb.Project
	Deleted                    bool
}

func (p *ProjectMessage) GetName() string {
	return fmt.Sprintf("projects/%s", p.ResourceID)
}

// FindProjectMessage is the message for finding projects.
type FindProjectMessage struct {
	ResourceID  *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	FilterQ     *qb.Query
	OrderByKeys []*OrderByKey
}

// UpdateProjectMessage is the message for updating a project.
type UpdateProjectMessage struct {
	ResourceID string

	Title                      *string
	DataClassificationConfigID *string
	Setting                    *storepb.Project
	Delete                     *bool
}

// GetProject gets project by resource ID.
func (s *Store) GetProject(ctx context.Context, find *FindProjectMessage) (*ProjectMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.projectCache.Get(*find.ResourceID); ok && s.enableCache {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	projects, err := s.ListProjects(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, nil
	}
	if len(projects) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d projects with filter %+v, expect 1", len(projects), find)}
	}
	project := projects[0]

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.storeProjectCache(project)
	return projects[0], nil
}

// ListProjects lists all projects.
func (s *Store) ListProjects(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := qb.Q().Space("SELECT resource_id, name, data_classification_config_id, setting, deleted FROM project WHERE TRUE")
	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}
	if !find.ShowDeleted {
		q.And("deleted = ?", false)
	}

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY project.resource_id")
	}
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	var projectMessages []*ProjectMessage
	rows, err := tx.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectMessage ProjectMessage
		var payload []byte
		if err := rows.Scan(
			&projectMessage.ResourceID,
			&projectMessage.Title,
			&projectMessage.DataClassificationConfigID,
			&payload,
			&projectMessage.Deleted,
		); err != nil {
			return nil, err
		}
		setting := &storepb.Project{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, setting); err != nil {
			return nil, err
		}
		projectMessage.Setting = setting
		projectMessages = append(projectMessages, &projectMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, project := range projectMessages {
		projectWebhooks, err := s.ListProjectWebhooks(ctx, &FindProjectWebhookMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, err
		}
		project.Webhooks = projectWebhooks
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, project := range projectMessages {
		s.storeProjectCache(project)
	}
	return projectMessages, nil
}

// CreateProject creates a project.
func (s *Store) CreateProject(ctx context.Context, create *ProjectMessage, creator *UserMessage) (*ProjectMessage, error) {
	if creator == nil {
		return nil, errors.Errorf("creator cannot be nil")
	}
	if create.Setting == nil {
		create.Setting = &storepb.Project{}
	}
	payload, err := protojson.Marshal(create.Setting)
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	project := &ProjectMessage{
		ResourceID:                 create.ResourceID,
		Title:                      create.Title,
		DataClassificationConfigID: create.DataClassificationConfigID,
		Setting:                    create.Setting,
	}
	q := qb.Q().Space("INSERT INTO project (resource_id, name, data_classification_config_id, setting)")
	q.Space("VALUES (?, ?, ?, ?)", create.ResourceID, create.Title, create.DataClassificationConfigID, payload)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return nil, err
	}

	policy := &storepb.IamPolicy{
		Bindings: []*storepb.Binding{
			{
				Role: common.FormatRole(common.ProjectOwner),
				Members: []string{
					common.FormatUserEmail(creator.Email),
				},
				Condition: &expr.Expr{},
			},
		},
	}
	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return nil, err
	}
	if _, err := s.CreatePolicy(ctx, &PolicyMessage{
		ResourceType:      storepb.Policy_PROJECT,
		Resource:          common.FormatProject(project.ResourceID),
		Payload:           string(policyPayload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.storeProjectCache(project)
	return project, nil
}

// UpdateProjects updates projects in a single transaction.
func (s *Store) UpdateProjects(ctx context.Context, patches ...*UpdateProjectMessage) ([]*ProjectMessage, error) {
	if len(patches) == 0 {
		return nil, nil
	}

	// Remove all projects from cache first
	for _, patch := range patches {
		s.removeProjectCache(patch.ResourceID)
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update all projects in the transaction
	for _, patch := range patches {
		if err := updateProjectImpl(ctx, tx, patch); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch and return all updated projects
	var updatedProjects []*ProjectMessage
	for _, patch := range patches {
		project, err := s.GetProject(ctx, &FindProjectMessage{ResourceID: &patch.ResourceID})
		if err != nil {
			return nil, err
		}
		updatedProjects = append(updatedProjects, project)
	}

	return updatedProjects, nil
}

func updateProjectImpl(ctx context.Context, txn *sql.Tx, patch *UpdateProjectMessage) error {
	set := qb.Q()

	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Delete; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if v := patch.DataClassificationConfigID; v != nil {
		set.Comma("data_classification_config_id = ?", *v)
	}
	if v := patch.Setting; v != nil {
		payload, err := protojson.Marshal(patch.Setting)
		if err != nil {
			return err
		}
		set.Comma("setting = ?", payload)
	}

	if set.Len() == 0 {
		return errors.New("no fields to update")
	}

	q := qb.Q().Space("UPDATE project SET ?", set).
		Space("WHERE resource_id = ?", patch.ResourceID)

	sql, args, err := q.ToSQL()
	if err != nil {
		return err
	}
	if _, err := txn.ExecContext(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}

func (s *Store) storeProjectCache(project *ProjectMessage) {
	s.projectCache.Add(project.ResourceID, project)
}

func (s *Store) removeProjectCache(resourceID string) {
	s.projectCache.Remove(resourceID)
}

// DeleteProject permanently purges a soft-deleted project and all related resources.
// This operation is irreversible and should only be used for:
// - Administrative cleanup of old soft-deleted projects
// - Test cleanup
// Following AIP-164/165, this only works on projects where deleted = TRUE.
func (s *Store) DeleteProject(ctx context.Context, resourceID string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Delete query_history entries that reference this project
	q := qb.Q().Space("DELETE FROM query_history WHERE project_id = ?", resourceID)
	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query_history delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete query_history for project %s", resourceID)
	}

	// Delete policy entries that reference this project
	q = qb.Q().Space("DELETE FROM policy")
	q.Space("WHERE (resource_type = ? AND resource = 'projects/' || ?)", storepb.Policy_PROJECT.String(), resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build policy delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete policies for project %s", resourceID)
	}

	// Delete worksheets associated with this project
	q = qb.Q().Space("UPDATE worksheet SET project = ? WHERE project = ?", common.DefaultProjectID, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build worksheet update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to update worksheets for project %s", resourceID)
	}

	// Delete issue_comment entries for issues in this project
	q = qb.Q().Space("DELETE FROM issue_comment")
	q.Space("WHERE issue_id IN (SELECT id FROM issue WHERE project = ?)", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build issue_comment delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete issue_comment for project %s", resourceID)
	}

	// Delete issues associated with this project
	q = qb.Q().Space("DELETE FROM issue WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build issue delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete issues for project %s", resourceID)
	}

	// Delete plan_check_run entries for plans in this project
	q = qb.Q().Space("DELETE FROM plan_check_run")
	q.Space("WHERE plan_id IN (SELECT id FROM plan WHERE project = ?)", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build plan_check_run delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plan_check_run for project %s", resourceID)
	}

	// Delete plans associated with this project
	q = qb.Q().Space("DELETE FROM plan WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build plan delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plans for project %s", resourceID)
	}

	// Delete task_run_log entries for tasks in plans of this project
	q = qb.Q().Space("DELETE FROM task_run_log")
	q.Space("WHERE task_run_id IN (")
	q.Space("SELECT tr.id FROM task_run tr")
	q.Space("JOIN task t ON tr.task_id = t.id")
	q.Space("JOIN plan p ON t.plan_id = p.id")
	q.Space("WHERE p.project = ?)", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task_run_log delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run_log for project %s", resourceID)
	}

	// Delete task_run entries for tasks in plans of this project
	q = qb.Q().Space("DELETE FROM task_run")
	q.Space("WHERE task_id IN (")
	q.Space("SELECT t.id FROM task t")
	q.Space("JOIN plan p ON t.plan_id = p.id")
	q.Space("WHERE p.project = ?)", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task_run delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run for project %s", resourceID)
	}

	// Delete tasks in plans of this project
	q = qb.Q().Space("DELETE FROM task")
	q.Space("WHERE plan_id IN (SELECT id FROM plan WHERE project = ?)", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete tasks for project %s", resourceID)
	}

	// Delete sheets associated with this project
	q = qb.Q().Space("DELETE FROM sheet WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build sheet delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete sheets for project %s", resourceID)
	}

	// Delete releases associated with this project
	q = qb.Q().Space("DELETE FROM release WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build release delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete releases for project %s", resourceID)
	}

	// Delete db_groups associated with this project
	q = qb.Q().Space("DELETE FROM db_group WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build db_group delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete db_groups for project %s", resourceID)
	}

	// Move databases to the default project instead of deleting them
	q = qb.Q().Space("UPDATE db SET project = ? WHERE project = ?", common.DefaultProjectID, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build db update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to move databases to default project for project %s", resourceID)
	}

	// Delete project webhooks
	q = qb.Q().Space("DELETE FROM project_webhook WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project_webhook delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete project_webhook for project %s", resourceID)
	}

	// Finally, delete the project itself (only if it's marked as deleted)
	q = qb.Q().Space("DELETE FROM project WHERE resource_id = ? AND deleted = TRUE", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project delete query")
	}
	result, err := tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to delete project %s", resourceID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rowsAffected == 0 {
		return errors.Errorf("project %s not found or not marked as deleted", resourceID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	// Clear the project from cache
	s.projectCache.Remove(resourceID)

	return nil
}

func GetListProjectFilter(filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Errorf("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	parseToLabelFilterSQL := func(resource, key string, value any) (*qb.Query, error) {
		switch v := value.(type) {
		case string:
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ?", resource, key), v), nil
		case []any:
			if len(v) == 0 {
				return nil, errors.Errorf("empty label filter")
			}

			labelValueList := make([]any, len(v))
			for i, raw := range v {
				str, ok := raw.(string)
				if !ok {
					return nil, errors.Errorf("label value must be string, got %T", raw)
				}
				labelValueList[i] = str
			}
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ANY(?)", resource, key), labelValueList), nil
		default:
			return nil, errors.Errorf("empty value %v for label filter", value)
		}
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "name":
			return qb.Q().Space("project.name = ?", value.(string)), nil
		case "resource_id":
			return qb.Q().Space("project.resource_id = ?", value.(string)), nil
		case "exclude_default":
			if excludeDefault, ok := value.(bool); excludeDefault && ok {
				return qb.Q().Space("project.resource_id != ?", common.DefaultProjectID), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "state":
			stateStr, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("state value must be string, got %T", value)
			}
			// Try with STATE_ prefix first (e.g., "STATE_ACTIVE", "STATE_DELETED")
			v1State, ok := v1pb.State_value[stateStr]
			if !ok {
				// If not found, try without STATE_ prefix (e.g., "ACTIVE", "DELETED")
				if v, exists := v1pb.State_value[strings.TrimPrefix(stateStr, "STATE_")]; exists {
					v1State = v
					ok = true
				}
			}
			if !ok {
				return nil, errors.Errorf("invalid state filter %q", value)
			}
			return qb.Q().Space("project.deleted = ?", v1pb.State(v1State) == v1pb.State_DELETED), nil
		default:
			varStr, ok := variable.(string)
			if !ok {
				return nil, errors.Errorf("unsupport variable %q", variable)
			}
			if labelKey, ok := strings.CutPrefix(varStr, "labels."); ok {
				return parseToLabelFilterSQL("project.setting", labelKey, value)
			}
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.Or("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.And("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}

				switch variable {
				case "name":
					return qb.Q().Space("LOWER(project.name) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				case "resource_id":
					return qb.Q().Space("LOWER(project.resource_id) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return parseToLabelFilterSQL("project.setting", labelKey, value)
				}
				return nil, errors.Errorf("unexpected %v operator for %v", functionName, variable)
			default:
				return nil, errors.Errorf("unexpected function %v", functionName)
			}
		default:
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	q, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", q), nil
}

func GetProjectOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}
	if len(keys) > 1 || keys[0].Key != "title" {
		return nil, errors.Errorf(`only support order by "title"`)
	}

	return []*OrderByKey{
		{
			Key:       "name",
			SortOrder: keys[0].SortOrder,
		},
	}, nil
}
