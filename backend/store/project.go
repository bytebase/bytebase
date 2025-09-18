package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	Filter      *ListResourceFilter
}

// UpdateProjectMessage is the message for updating a project.
type UpdateProjectMessage struct {
	ResourceID string

	Title                      *string
	DataClassificationConfigID *string
	Setting                    *storepb.Project
	Delete                     *bool
}

// GetProjectV2 gets project by resource ID.
func (s *Store) GetProjectV2(ctx context.Context, find *FindProjectMessage) (*ProjectMessage, error) {
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

	projects, err := s.listProjectImplV2(ctx, tx, find)
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

// ListProjectV2 lists all projects.
func (s *Store) ListProjectV2(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	projects, err := s.listProjectImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, project := range projects {
		s.storeProjectCache(project)
	}
	return projects, nil
}

// CreateProjectV2 creates a project.
func (s *Store) CreateProjectV2(ctx context.Context, create *ProjectMessage, creatorID int) (*ProjectMessage, error) {
	user, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
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
	if _, err := tx.ExecContext(ctx, `
			INSERT INTO project (
				resource_id,
				name,
				data_classification_config_id,
				setting
			)
			VALUES ($1, $2, $3, $4)
		`,
		create.ResourceID,
		create.Title,
		create.DataClassificationConfigID,
		payload,
	); err != nil {
		return nil, err
	}

	policy := &storepb.IamPolicy{
		Bindings: []*storepb.Binding{
			{
				Role: common.FormatRole(common.ProjectOwner),
				Members: []string{
					common.FormatUserUID(user.ID),
				},
				Condition: &expr.Expr{},
			},
		},
	}
	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return nil, err
	}
	if _, err := s.CreatePolicyV2(ctx, &PolicyMessage{
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

// UpdateProjectV2 updates a project.
func (s *Store) UpdateProjectV2(ctx context.Context, patch *UpdateProjectMessage) (*ProjectMessage, error) {
	s.removeProjectCache(patch.ResourceID)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := updateProjectImplV2(ctx, tx, patch); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &patch.ResourceID})
}

// BatchUpdateProjectsV2 updates multiple projects in a single transaction.
func (s *Store) BatchUpdateProjectsV2(ctx context.Context, patches []*UpdateProjectMessage) ([]*ProjectMessage, error) {
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
		if err := updateProjectImplV2(ctx, tx, patch); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch and return all updated projects
	var updatedProjects []*ProjectMessage
	for _, patch := range patches {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &patch.ResourceID})
		if err != nil {
			return nil, err
		}
		updatedProjects = append(updatedProjects, project)
	}

	return updatedProjects, nil
}

func updateProjectImplV2(ctx context.Context, txn *sql.Tx, patch *UpdateProjectMessage) error {
	set, args := []string{}, []any{}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Delete; v != nil {
		set, args = append(set, fmt.Sprintf("deleted = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DataClassificationConfigID; v != nil {
		set, args = append(set, fmt.Sprintf("data_classification_config_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Setting; v != nil {
		payload, err := protojson.Marshal(patch.Setting)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("setting = $%d", len(args)+1)), append(args, payload)
	}

	args = append(args, patch.ResourceID)
	if _, err := txn.ExecContext(ctx, fmt.Sprintf(`
		UPDATE project
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d`, len(args)),
		args...,
	); err != nil {
		return err
	}
	return nil
}

func (s *Store) listProjectImplV2(ctx context.Context, txn *sql.Tx, find *FindProjectMessage) ([]*ProjectMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("deleted = $%d", len(args)+1)), append(args, false)
	}

	query := fmt.Sprintf(`
		SELECT
			resource_id,
			name,
			data_classification_config_id,
			setting,
			deleted
		FROM project
		WHERE %s
		ORDER BY project.resource_id`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	var projectMessages []*ProjectMessage
	rows, err := txn.QueryContext(ctx, query, args...)
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
		projectWebhooks, err := s.findProjectWebhookImplV2(ctx, txn, &FindProjectWebhookMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, err
		}
		project.Webhooks = projectWebhooks
	}

	return projectMessages, nil
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
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM query_history
		WHERE project_id = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete query_history for project %s", resourceID)
	}

	// Delete policy entries that reference this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM policy
		WHERE (resource_type = $1 AND resource = 'projects/' || $2)
	`, storepb.Policy_PROJECT.String(), resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete policies for project %s", resourceID)
	}

	// Delete worksheets associated with this project
	if _, err := tx.ExecContext(ctx, `
		UPDATE worksheet
		SET project = $1
		WHERE project = $2
	`, common.DefaultProjectID, resourceID); err != nil {
		return errors.Wrapf(err, "failed to update worksheets for project %s", resourceID)
	}

	// Delete issue_comment entries for issues in this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM issue_comment
		WHERE issue_id IN (
			SELECT id FROM issue WHERE project = $1
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete issue_comment for project %s", resourceID)
	}

	// Delete issues associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM issue WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete issues for project %s", resourceID)
	}

	// Delete plan_check_run entries for plans in this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM plan_check_run
		WHERE plan_id IN (
			SELECT id FROM plan WHERE project = $1
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete plan_check_run for project %s", resourceID)
	}

	// Delete plans associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM plan WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete plans for project %s", resourceID)
	}

	// Delete task_run_log entries for tasks in pipelines of this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM task_run_log
		WHERE task_run_id IN (
			SELECT tr.id FROM task_run tr
			JOIN task t ON tr.task_id = t.id
			JOIN pipeline p ON t.pipeline_id = p.id
			WHERE p.project = $1
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete task_run_log for project %s", resourceID)
	}

	// Delete task_run entries for tasks in pipelines of this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM task_run
		WHERE task_id IN (
			SELECT t.id FROM task t
			JOIN pipeline p ON t.pipeline_id = p.id
			WHERE p.project = $1
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete task_run for project %s", resourceID)
	}

	// Delete tasks in pipelines of this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM task
		WHERE pipeline_id IN (
			SELECT id FROM pipeline WHERE project = $1
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete tasks for project %s", resourceID)
	}

	// Delete pipelines associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM pipeline WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete pipelines for project %s", resourceID)
	}

	// Delete sheets associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM sheet WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete sheets for project %s", resourceID)
	}

	// Delete releases associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM release WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete releases for project %s", resourceID)
	}

	// Delete changelists associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM changelist WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete changelists for project %s", resourceID)
	}

	// Delete db_groups associated with this project
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM db_group WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete db_groups for project %s", resourceID)
	}

	// Move databases to the default project instead of deleting them
	if _, err := tx.ExecContext(ctx, `
		UPDATE db
		SET project = $1
		WHERE project = $2
	`, common.DefaultProjectID, resourceID); err != nil {
		return errors.Wrapf(err, "failed to move databases to default project for project %s", resourceID)
	}

	// Delete project webhooks
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM project_webhook WHERE project = $1
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete project_webhook for project %s", resourceID)
	}

	// Finally, delete the project itself (only if it's marked as deleted)
	result, err := tx.ExecContext(ctx, `
		DELETE FROM project
		WHERE resource_id = $1 AND deleted = TRUE
	`, resourceID)
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
