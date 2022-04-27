package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// projectRaw is the store model for a Project.
// Fields have exactly the same meanings as Project.
type projectRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name           string
	Key            string
	WorkflowType   api.ProjectWorkflowType
	Visibility     api.ProjectVisibility
	TenantMode     api.ProjectTenantMode
	DBNameTemplate string
	RoleProvider   api.ProjectRoleProvider
}

// ToProject creates an instance of Project based on the projectRaw.
// This is intended to be called when we need to compose a Project relationship.
func (raw *projectRaw) ToProject() *api.Project {
	return &api.Project{
		ID: raw.ID,

		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		Name:           raw.Name,
		Key:            raw.Key,
		WorkflowType:   raw.WorkflowType,
		Visibility:     raw.Visibility,
		TenantMode:     raw.TenantMode,
		DBNameTemplate: raw.DBNameTemplate,
		RoleProvider:   raw.RoleProvider,
	}
}

// GetProjectByID gets an instance of Project
func (s *Store) GetProjectByID(ctx context.Context, id int) (*api.Project, error) {
	find := &api.ProjectFind{ID: &id}
	projectRaw, err := s.getProjectRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Project with ID[%d], error[%w]", id, err)
	}
	if projectRaw == nil {
		return nil, nil
	}
	project, err := s.composeProject(ctx, projectRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Project with projectRaw[%+v], error[%w]", projectRaw, err)
	}
	return project, nil
}

// FindProject finds a list of Project instances
func (s *Store) FindProject(ctx context.Context, find *api.ProjectFind) ([]*api.Project, error) {
	projectRawList, err := s.findProjectRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Project list, error[%w]", err)
	}
	var projectList []*api.Project
	for _, raw := range projectRawList {
		project, err := s.composeProject(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Project with projectRaw[%+v], error[%w]", raw, err)
		}
		projectList = append(projectList, project)
	}
	return projectList, nil
}

// CreateProject creates an instance of Project
func (s *Store) CreateProject(ctx context.Context, create *api.ProjectCreate) (*api.Project, error) {
	projectRaw, err := s.createProjectRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Project with ProjectCreate[%+v], error[%w]", create, err)
	}
	project, err := s.composeProject(ctx, projectRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Project with projectRaw[%+v], error[%w]", projectRaw, err)
	}
	return project, nil
}

// PatchProject patches an instance of Project
func (s *Store) PatchProject(ctx context.Context, patch *api.ProjectPatch) (*api.Project, error) {
	projectRaw, err := s.patchProjectRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch Project with ProjectPatch[%+v], error[%w]", patch, err)
	}
	project, err := s.composeProject(ctx, projectRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Project with projectRaw[%+v], error[%w]", projectRaw, err)
	}
	return project, nil
}

//
// private functions
//

func (s *Store) composeProject(ctx context.Context, raw *projectRaw) (*api.Project, error) {
	project := raw.ToProject()

	creator, err := s.GetPrincipalByID(ctx, project.CreatorID)
	if err != nil {
		return nil, err
	}
	project.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, project.UpdaterID)
	if err != nil {
		return nil, err
	}
	project.Updater = updater

	projectMemberList, err := s.FindProjectMember(ctx, &api.ProjectMemberFind{ProjectID: &project.ID})
	if err != nil {
		return nil, err
	}
	project.ProjectMemberList = projectMemberList

	return project, nil
}

// createProjectRaw creates a new project.
func (s *Store) createProjectRaw(ctx context.Context, create *api.ProjectCreate) (*projectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectRaw, err := createProjectImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.ProjectCache, projectRaw.ID, projectRaw); err != nil {
		return nil, err
	}

	return projectRaw, nil
}

// findProjectRaw retrieves a list of projects based on find.
func (s *Store) findProjectRaw(ctx context.Context, find *api.ProjectFind) ([]*projectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, project := range list {
			if err := s.cache.UpsertCache(api.ProjectCache, project.ID, project); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// getProjectRaw retrieves a single project based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getProjectRaw(ctx context.Context, find *api.ProjectFind) (*projectRaw, error) {
	if find.ID != nil {
		project := &projectRaw{}
		has, err := s.cache.FindCache(api.ProjectCache, *find.ID, project)
		if err != nil {
			return nil, err
		}
		if has {
			return project, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d projects with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.ProjectCache, list[0].ID, list[0]); err != nil {
		return nil, err
	}
	return list[0], nil
}

// patchProjectRaw updates an existing project by ID.
// Returns ENOTFOUND if project does not exist.
func (s *Store) patchProjectRaw(ctx context.Context, patch *api.ProjectPatch) (*projectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	project, err := patchProjectImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.ProjectCache, project.ID, project); err != nil {
		return nil, err
	}

	return project, nil
}

// patchProjectRawTx updates an existing project by ID.
// Returns ENOTFOUND if project does not exist.
func (s *Store) patchProjectRawTx(ctx context.Context, tx *sql.Tx, patch *api.ProjectPatch) (*projectRaw, error) {
	project, err := patchProjectImpl(ctx, tx, patch)

	if err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.ProjectCache, project.ID, project); err != nil {
		return nil, err
	}

	return project, nil
}

// createProjectImpl creates a new project.
func createProjectImpl(ctx context.Context, tx *sql.Tx, create *api.ProjectCreate) (*projectRaw, error) {
	// Insert row into database.
	if create.RoleProvider == "" {
		create.RoleProvider = api.ProjectRoleProviderBytebase
	}
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project (
			creator_id,
			updater_id,
			name,
			key,
			workflow_type,
			visibility,
			tenant_mode,
			db_name_template,
			role_provider
		)
		VALUES ($1, $2, $3, $4, 'UI', 'PUBLIC', $5, $6, $7)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, role_provider
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		strings.ToUpper(create.Key),
		create.TenantMode,
		create.DBNameTemplate,
		create.RoleProvider,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var project projectRaw
	if err := row.Scan(
		&project.ID,
		&project.RowStatus,
		&project.CreatorID,
		&project.CreatedTs,
		&project.UpdaterID,
		&project.UpdatedTs,
		&project.Name,
		&project.Key,
		&project.WorkflowType,
		&project.Visibility,
		&project.TenantMode,
		&project.DBNameTemplate,
		&project.RoleProvider,
	); err != nil {
		return nil, FormatError(err)
	}

	return &project, nil
}

func findProjectImpl(ctx context.Context, tx *sql.Tx, find *api.ProjectFind) ([]*projectRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("id IN (SELECT project_id FROM project_project WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			key,
			workflow_type,
			visibility,
			tenant_mode,
			db_name_template,
			role_provider
		FROM project
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into projectRawList.
	var projectRawList []*projectRaw
	for rows.Next() {
		var project projectRaw
		if err := rows.Scan(
			&project.ID,
			&project.RowStatus,
			&project.CreatorID,
			&project.CreatedTs,
			&project.UpdaterID,
			&project.UpdatedTs,
			&project.Name,
			&project.Key,
			&project.WorkflowType,
			&project.Visibility,
			&project.TenantMode,
			&project.DBNameTemplate,
			&project.RoleProvider,
		); err != nil {
			return nil, FormatError(err)
		}

		projectRawList = append(projectRawList, &project)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return projectRawList, nil
}

// patchProjectImpl updates a project by ID. Returns the new state of the project after update.
func patchProjectImpl(ctx context.Context, tx *sql.Tx, patch *api.ProjectPatch) (*projectRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Key; v != nil {
		set, args = append(set, fmt.Sprintf("key = $%d", len(args)+1)), append(args, strings.ToUpper(*v))
	}
	if v := patch.WorkflowType; v != nil {
		set, args = append(set, fmt.Sprintf("workflow_type = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.RoleProvider; v != nil {
		set, args = append(set, fmt.Sprintf("role_provider = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE project
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, key, workflow_type, visibility, tenant_mode, db_name_template, role_provider
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var project projectRaw
		if err := row.Scan(
			&project.ID,
			&project.RowStatus,
			&project.CreatorID,
			&project.CreatedTs,
			&project.UpdaterID,
			&project.UpdatedTs,
			&project.Name,
			&project.Key,
			&project.WorkflowType,
			&project.Visibility,
			&project.TenantMode,
			&project.DBNameTemplate,
			&project.RoleProvider,
		); err != nil {
			return nil, FormatError(err)
		}

		return &project, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project ID not found: %d", patch.ID)}
}
