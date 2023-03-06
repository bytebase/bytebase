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
	composedProject, err := s.composeProject(ctx, project)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Project with projectRaw[%+v]", project)
	}
	return composedProject, nil
}

// FindProject finds a list of Project instances.
func (s *Store) FindProject(ctx context.Context, find *api.ProjectFind) ([]*api.Project, error) {
	v2Find := &FindProjectMessage{ShowDeleted: true}
	projects, err := s.ListProjectV2(ctx, v2Find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Project list with ProjectFind[%+v]", v2Find)
	}
	var composedProjects []*api.Project
	for _, project := range projects {
		composedProject, err := s.composeProject(ctx, project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Project with projectRaw[%+v]", project)
		}
		if find.RowStatus != nil && composedProject.RowStatus != *find.RowStatus {
			continue
		}
		composedProjects = append(composedProjects, composedProject)
	}
	return composedProjects, nil
}

// PatchProject patches an instance of Project.
func (s *Store) PatchProject(ctx context.Context, patch *api.ProjectPatch) (*api.Project, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &patch.ID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project %d", patch.ID)
	}
	v2Update := &UpdateProjectMessage{
		UpdaterID:        patch.UpdaterID,
		ResourceID:       project.ResourceID,
		Title:            patch.Name,
		Key:              patch.Key,
		DBNameTemplate:   patch.DBNameTemplate,
		LGTMCheckSetting: patch.LGTMCheckSetting,
	}
	if patch.TenantMode != nil {
		m := api.ProjectTenantMode(*patch.TenantMode)
		v2Update.TenantMode = &m
	}

	if patch.WorkflowType != nil {
		v := api.ProjectWorkflowType(*patch.WorkflowType)
		v2Update.Workflow = &v
	}
	if patch.SchemaChangeType != nil {
		v := api.ProjectSchemaChangeType(*patch.SchemaChangeType)
		v2Update.SchemaChangeType = &v
	}
	if patch.RowStatus != nil {
		deleted := *patch.RowStatus == string(api.Archived)
		v2Update.Delete = &deleted
	}
	project, err = s.UpdateProjectV2(ctx, v2Update)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Project with ProjectPatch %#v", patch)
	}
	composedProject, err := s.composeProject(ctx, project)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Project with projectRaw[%+v]", project)
	}
	return composedProject, nil
}

func (s *Store) composeProject(ctx context.Context, project *ProjectMessage) (*api.Project, error) {
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
		LGTMCheckSetting: project.LGTMCheckSetting,
	}
	if project.Deleted {
		composedProject.RowStatus = api.Archived
	}

	policy, err := s.GetProjectPolicy(ctx, &GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, err
	}
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			principal, err := s.GetPrincipalByID(ctx, member.ID)
			if err != nil {
				return nil, err
			}
			composedProject.ProjectMemberList = append(composedProject.ProjectMemberList, &api.ProjectMember{
				ID:        member.ID,
				ProjectID: project.UID,
				Role:      string(binding.Role),
				Principal: principal,
			})
		}
	}
	return composedProject, nil
}

// ProjectMessage is the mssage for project.
type ProjectMessage struct {
	ResourceID       string
	Title            string
	Key              string
	Workflow         api.ProjectWorkflowType
	Visibility       api.ProjectVisibility
	TenantMode       api.ProjectTenantMode
	DBNameTemplate   string
	SchemaChangeType api.ProjectSchemaChangeType
	LGTMCheckSetting api.LGTMCheckSetting
	Webhooks         []*ProjectWebhookMessage
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

	Title            *string
	Key              *string
	TenantMode       *api.ProjectTenantMode
	DBNameTemplate   *string
	Workflow         *api.ProjectWorkflowType
	SchemaChangeType *api.ProjectSchemaChangeType
	LGTMCheckSetting *api.LGTMCheckSetting
	Delete           *bool
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
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	s.projectCache.Store(project.ResourceID, project)
	s.projectIDCache.Store(project.UID, project)
	return projects[0], nil
}

// ListProjectV2 lists all projects.
func (s *Store) ListProjectV2(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projects, err := s.listProjectImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, project := range projects {
		s.projectCache.Store(project.ResourceID, project)
		s.projectIDCache.Store(project.UID, project)
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
	if create.LGTMCheckSetting.Value == "" {
		create.LGTMCheckSetting = api.GetDefaultLGTMCheckSetting()
	}

	user, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
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
		LGTMCheckSetting: create.LGTMCheckSetting,
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
				schema_change_type,
				lgtm_check
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
		create.LGTMCheckSetting,
	).Scan(
		&project.UID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery("failed to create project")
		}
		return nil, FormatError(err)
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
		return nil, FormatError(err)
	}

	s.projectCache.Store(project.ResourceID, project)
	s.projectIDCache.Store(project.UID, project)
	return project, nil
}

// UpdateProjectV2 updates a project.
func (s *Store) UpdateProjectV2(ctx context.Context, patch *UpdateProjectMessage) (*ProjectMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	project, err := s.updateProjectImplV2(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.projectCache.Store(project.ResourceID, project)
	s.projectIDCache.Store(project.UID, project)
	return project, nil
}

// WARNING: calling updateProjectImplV2 from other store library has to invalidate the cache.
func (s *Store) updateProjectImplV2(ctx context.Context, tx *Tx, patch *UpdateProjectMessage) (*ProjectMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}
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
	if v := patch.LGTMCheckSetting; v != nil {
		set, args = append(set, fmt.Sprintf("lgtm_check = $%d", len(args)+1)), append(args, *v)
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
			lgtm_check,
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
		&project.LGTMCheckSetting,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project ID not found: %s", patch.ResourceID)}
		}
		return nil, FormatError(err)
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
	where, args := []string{"TRUE"}, []interface{}{}
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
			lgtm_check,
			row_status
		FROM project
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
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
			&projectMessage.LGTMCheckSetting,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		projectWebhooks, err := s.findProjectWebhookImplV2(ctx, tx, &FindProjectWebhookMessage{ProjectID: &projectMessage.UID})
		if err != nil {
			return nil, err
		}
		projectMessage.Webhooks = projectWebhooks
		projectMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		projectMessages = append(projectMessages, &projectMessage)
	}

	return projectMessages, nil
}
