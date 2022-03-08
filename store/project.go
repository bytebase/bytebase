package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.ProjectService = (*ProjectService)(nil)
)

// ProjectService represents a service for managing project.
type ProjectService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewProjectService returns a new project of ProjectService.
func NewProjectService(logger *zap.Logger, db *DB, cache api.CacheService) *ProjectService {
	return &ProjectService{l: logger, db: db, cache: cache}
}

// CreateProject creates a new project.
func (s *ProjectService) CreateProject(ctx context.Context, create *api.ProjectCreate) (*api.ProjectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectRaw, err := createProject(ctx, tx.PTx, create)
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

// FindProjectList retrieves a list of projects based on find.
func (s *ProjectService) FindProjectList(ctx context.Context, find *api.ProjectFind) ([]*api.ProjectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectList(ctx, tx.PTx, find)
	if err != nil {
		return []*api.ProjectRaw{}, err
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

// FindProject retrieves a single project based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ProjectService) FindProject(ctx context.Context, find *api.ProjectFind) (*api.ProjectRaw, error) {
	if find.ID != nil {
		project := &api.ProjectRaw{}
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

	list, err := findProjectList(ctx, tx.PTx, find)
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

// PatchProject updates an existing project by ID.
// Returns ENOTFOUND if project does not exist.
func (s *ProjectService) PatchProject(ctx context.Context, patch *api.ProjectPatch) (*api.ProjectRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	project, err := patchProject(ctx, tx.PTx, patch)
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

// PatchProjectTx updates an existing project by ID.
// Returns ENOTFOUND if project does not exist.
func (s *ProjectService) PatchProjectTx(ctx context.Context, tx *sql.Tx, patch *api.ProjectPatch) (*api.ProjectRaw, error) {
	project, err := patchProject(ctx, tx, patch)

	if err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.ProjectCache, project.ID, project); err != nil {
		return nil, err
	}

	return project, nil
}

// createProject creates a new project.
func createProject(ctx context.Context, tx *sql.Tx, create *api.ProjectCreate) (*api.ProjectRaw, error) {
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
	var project api.ProjectRaw
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

func findProjectList(ctx context.Context, tx *sql.Tx, find *api.ProjectFind) ([]*api.ProjectRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("id IN (SELECT project_id FROM project_member WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.ProjectRaw, 0)
	for rows.Next() {
		var project api.ProjectRaw
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

		list = append(list, &project)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchProject updates a project by ID. Returns the new state of the project after update.
func patchProject(ctx context.Context, tx *sql.Tx, patch *api.ProjectPatch) (*api.ProjectRaw, error) {
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
		var project api.ProjectRaw
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
