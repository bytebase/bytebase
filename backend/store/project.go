package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// GetProjectByID gets an instance of Project.
func (s *Store) GetProjectByID(ctx context.Context, id int) (*api.Project, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Project with ID %d", id)
	}
	if project == nil {
		return nil, nil
	}
	composedProject, err := s.composeProject(project)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Project with projectRaw[%+v]", project)
	}
	return composedProject, nil
}

func (*Store) composeProject(project *ProjectMessage) (*api.Project, error) {
	composedProject := &api.Project{
		ID:               project.UID,
		ResourceID:       project.ResourceID,
		RowStatus:        api.Normal,
		Name:             project.Title,
		Key:              project.Key,
		WorkflowType:     project.Workflow,
		Visibility:       project.Visibility,
		TenantMode:       project.TenantMode,
		DBNameTemplate:   project.DBNameTemplate,
		SchemaChangeType: project.SchemaChangeType,
	}
	if project.Deleted {
		composedProject.RowStatus = api.Archived
	}
	return composedProject, nil
}

// ProjectMessage is the message for project.
type ProjectMessage struct {
	ResourceID                 string
	Title                      string
	Key                        string
	Workflow                   api.ProjectWorkflowType
	Visibility                 api.ProjectVisibility
	TenantMode                 api.ProjectTenantMode
	DBNameTemplate             string
	SchemaChangeType           api.ProjectSchemaChangeType
	Webhooks                   []*ProjectWebhookMessage
	DataClassificationConfigID string
	// The following fields are output only and not used for create().
	UID     int
	Deleted bool
}

// FindProjectMessage is the message for finding projects.
type FindProjectMessage struct {
	// We should only set either UID or ResourceID.
	// Deprecate UID later once we fully migrate to ResourceID.
	UID         *int
	ResourceID  *string
	ShowDeleted bool
}

// UpdateProjectMessage is the message for updating a project.
type UpdateProjectMessage struct {
	UpdaterID  int
	ResourceID string

	Title                      *string
	Key                        *string
	TenantMode                 *api.ProjectTenantMode
	DBNameTemplate             *string
	Workflow                   *api.ProjectWorkflowType
	SchemaChangeType           *api.ProjectSchemaChangeType
	DataClassificationConfigID *string
	Delete                     *bool
}

// GetProjectV2 gets project by resource ID.
func (s *Store) GetProjectV2(ctx context.Context, find *FindProjectMessage) (*ProjectMessage, error) {
	if find.ResourceID != nil {
		if project, ok := s.projectCache.Load(*find.ResourceID); ok {
			return project.(*ProjectMessage), nil
		}
	}
	if find.UID != nil {
		if project, ok := s.projectIDCache.Load(*find.UID); ok {
			return project.(*ProjectMessage), nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	// TODO(d): consider moving these defaults to somewhere else.
	if create.Workflow == "" {
		create.Workflow = api.UIWorkflow
	}
	if create.Visibility == "" {
		create.Visibility = api.Public
	}
	if create.SchemaChangeType == "" {
		create.SchemaChangeType = api.ProjectSchemaChangeTypeDDL
	}
	user, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	project := &ProjectMessage{
		ResourceID:       create.ResourceID,
		Title:            create.Title,
		Key:              create.Key,
		Workflow:         create.Workflow,
		Visibility:       create.Visibility,
		TenantMode:       create.TenantMode,
		DBNameTemplate:   create.DBNameTemplate,
		SchemaChangeType: create.SchemaChangeType,
	}
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO project (
				creator_id,
				updater_id,
				resource_id,
				name,
				key,
				workflow_type,
				visibility,
				tenant_mode,
				db_name_template,
				schema_change_type
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`,
		creatorID,
		creatorID,
		create.ResourceID,
		create.Title,
		create.Key,
		create.Workflow,
		create.Visibility,
		create.TenantMode,
		create.DBNameTemplate,
		create.SchemaChangeType,
	).Scan(
		&project.UID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery("failed to create project")
		}
		return nil, err
	}

	policy := &IAMPolicyMessage{
		Bindings: []*PolicyBinding{
			{
				Role: api.Owner,
				Members: []*UserMessage{
					user,
				},
			},
		},
	}
	if err := s.setProjectIAMPolicyImpl(ctx, tx, policy, creatorID, project.UID); err != nil {
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	project, err := s.updateProjectImplV2(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.storeProjectCache(project)
	return project, nil
}

// WARNING: calling updateProjectImplV2 from other store library has to invalidate the cache.
func (s *Store) updateProjectImplV2(ctx context.Context, tx *Tx, patch *UpdateProjectMessage) (*ProjectMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Key; v != nil {
		set, args = append(set, fmt.Sprintf("key = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	if v := patch.TenantMode; v != nil {
		set, args = append(set, fmt.Sprintf("tenant_mode = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DBNameTemplate; v != nil {
		set, args = append(set, fmt.Sprintf("db_name_template = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Workflow; v != nil {
		set, args = append(set, fmt.Sprintf("workflow_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaChangeType; v != nil {
		set, args = append(set, fmt.Sprintf("schema_change_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DataClassificationConfigID; v != nil {
		set, args = append(set, fmt.Sprintf("data_classification_config_id = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ResourceID)

	project := &ProjectMessage{}
	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE project
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d
		RETURNING
			id,
			resource_id,
			name,
			key,
			workflow_type,
			visibility,
			tenant_mode,
			db_name_template,
			schema_change_type,
			data_classification_config_id,
			row_status
	`, len(args)),
		args...,
	).Scan(
		&project.UID,
		&project.ResourceID,
		&project.Title,
		&project.Key,
		&project.Workflow,
		&project.Visibility,
		&project.TenantMode,
		&project.DBNameTemplate,
		&project.SchemaChangeType,
		&project.DataClassificationConfigID,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %s", patch.ResourceID)}
		}
		return nil, err
	}
	projectWebhooks, err := s.findProjectWebhookImplV2(ctx, tx, &FindProjectWebhookMessage{ProjectID: &project.UID})
	if err != nil {
		return nil, err
	}
	project.Webhooks = projectWebhooks
	project.Deleted = convertRowStatusToDeleted(rowStatus)
	return project, nil
}

func (s *Store) listProjectImplV2(ctx context.Context, tx *Tx, find *FindProjectMessage) ([]*ProjectMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var projectMessages []*ProjectMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			name,
			key,
			workflow_type,
			visibility,
			tenant_mode,
			db_name_template,
			schema_change_type,
			data_classification_config_id,
			row_status
		FROM project
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectMessage ProjectMessage
		var rowStatus string
		if err := rows.Scan(
			&projectMessage.UID,
			&projectMessage.ResourceID,
			&projectMessage.Title,
			&projectMessage.Key,
			&projectMessage.Workflow,
			&projectMessage.Visibility,
			&projectMessage.TenantMode,
			&projectMessage.DBNameTemplate,
			&projectMessage.SchemaChangeType,
			&projectMessage.DataClassificationConfigID,
			&rowStatus,
		); err != nil {
			return nil, err
		}
		projectMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		projectMessages = append(projectMessages, &projectMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, project := range projectMessages {
		projectWebhooks, err := s.findProjectWebhookImplV2(ctx, tx, &FindProjectWebhookMessage{ProjectID: &project.UID})
		if err != nil {
			return nil, err
		}
		project.Webhooks = projectWebhooks
	}

	return projectMessages, nil
}

func (s *Store) storeProjectCache(project *ProjectMessage) {
	s.projectCache.Store(project.ResourceID, project)
	s.projectIDCache.Store(project.UID, project)
}

func (s *Store) removeProjectCache(resourceID string) {
	if project, ok := s.projectCache.Load(resourceID); ok {
		s.projectIDCache.Delete(project.(*ProjectMessage).UID)
	}
	s.projectCache.Delete(resourceID)
}
