package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.RepositoryService = (*RepositoryService)(nil)
)

// RepositoryService represents a service for managing repository.
type RepositoryService struct {
	l  *zap.Logger
	db *DB

	projectService api.ProjectService
}

// NewRepositoryService returns a new instance of RepositoryService.
func NewRepositoryService(logger *zap.Logger, db *DB, projectService api.ProjectService) *RepositoryService {
	return &RepositoryService{l: logger, db: db, projectService: projectService}
}

// CreateRepository creates a new repository.
func (s *RepositoryService) CreateRepository(ctx context.Context, create *api.RepositoryCreate) (*api.Repository, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	repository, err := s.createRepository(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return repository, nil
}

// FindRepositoryList retrieves a list of repositorys based on find.
func (s *RepositoryService) FindRepositoryList(ctx context.Context, find *api.RepositoryFind) ([]*api.Repository, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findRepositoryList(ctx, tx, find)
	if err != nil {
		return []*api.Repository{}, err
	}

	return list, nil
}

// FindRepository retrieves a single repository based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *RepositoryService) FindRepository(ctx context.Context, find *api.RepositoryFind) (*api.Repository, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findRepositoryList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d repositories with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchRepository updates an existing repository by ID.
// Returns ENOTFOUND if repository does not exist.
func (s *RepositoryService) PatchRepository(ctx context.Context, patch *api.RepositoryPatch) (*api.Repository, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	repository, err := patchRepository(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return repository, nil
}

// DeleteRepository deletes an existing repository by ID.
func (s *RepositoryService) DeleteRepository(ctx context.Context, delete *api.RepositoryDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = s.deleteRepository(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createRepository creates a new repository.
func (s *RepositoryService) createRepository(ctx context.Context, tx *Tx, create *api.RepositoryCreate) (*api.Repository, error) {
	// Updates the project workflow_type to "VCS"
	workflowType := api.VCSWorkflow
	projectPatch := api.ProjectPatch{
		ID:           create.ProjectID,
		UpdaterID:    create.CreatorID,
		WorkflowType: &workflowType,
	}
	if _, err := s.projectService.PatchProjectTx(ctx, tx.Tx, &projectPatch); err != nil {
		return nil, err
	}

	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO repository (
		    creator_id,
		    updater_id,
			vcs_id,
			project_id,
			name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			file_path_template,
			schema_path_template,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token,
			access_token,
			expires_ts,
			refresh_token
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
	`,
		create.CreatorID,
		create.CreatorID,
		create.VCSID,
		create.ProjectID,
		create.Name,
		create.FullPath,
		create.WebURL,
		create.BranchFilter,
		create.BaseDirectory,
		create.FilePathTemplate,
		create.SchemaPathTemplate,
		create.ExternalID,
		create.ExternalWebhookID,
		create.WebhookURLHost,
		create.WebhookEndpointID,
		create.WebhookSecretToken,
		create.AccessToken,
		create.ExpiresTs,
		create.RefreshToken,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var repository api.Repository
	if err := row.Scan(
		&repository.ID,
		&repository.CreatorID,
		&repository.CreatedTs,
		&repository.UpdaterID,
		&repository.UpdatedTs,
		&repository.VCSID,
		&repository.ProjectID,
		&repository.Name,
		&repository.FullPath,
		&repository.WebURL,
		&repository.BranchFilter,
		&repository.BaseDirectory,
		&repository.FilePathTemplate,
		&repository.SchemaPathTemplate,
		&repository.ExternalID,
		&repository.ExternalWebhookID,
		&repository.WebhookURLHost,
		&repository.WebhookEndpointID,
		&repository.WebhookSecretToken,
		&repository.AccessToken,
		&repository.ExpiresTs,
		&repository.RefreshToken,
	); err != nil {
		return nil, FormatError(err)
	}

	return &repository, nil
}

func findRepositoryList(ctx context.Context, tx *Tx, find *api.RepositoryFind) (_ []*api.Repository, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.VCSID; v != nil {
		where, args = append(where, "vcs_id = ?"), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, "project_id = ?"), append(args, *v)
	}
	if v := find.WebhookEndpointID; v != nil {
		where, args = append(where, "webhook_endpoint_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			vcs_id,
			project_id,
			name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			file_path_template,
			schema_path_template,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token,
			access_token,
			expires_ts,
			refresh_token
		FROM repository
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Repository, 0)
	for rows.Next() {
		var repository api.Repository
		if err := rows.Scan(
			&repository.ID,
			&repository.CreatorID,
			&repository.CreatedTs,
			&repository.UpdaterID,
			&repository.UpdatedTs,
			&repository.VCSID,
			&repository.ProjectID,
			&repository.Name,
			&repository.FullPath,
			&repository.WebURL,
			&repository.BranchFilter,
			&repository.BaseDirectory,
			&repository.FilePathTemplate,
			&repository.SchemaPathTemplate,
			&repository.ExternalID,
			&repository.ExternalWebhookID,
			&repository.WebhookURLHost,
			&repository.WebhookEndpointID,
			&repository.WebhookSecretToken,
			&repository.AccessToken,
			&repository.ExpiresTs,
			&repository.RefreshToken,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &repository)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchRepository updates a repository by ID. Returns the new state of the repository after update.
func patchRepository(ctx context.Context, tx *Tx, patch *api.RepositoryPatch) (*api.Repository, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.BranchFilter; v != nil {
		set, args = append(set, "branch_filter = ?"), append(args, *v)
	}
	if v := patch.BaseDirectory; v != nil {
		set, args = append(set, "base_directory = ?"), append(args, *v)
	}
	if v := patch.FilePathTemplate; v != nil {
		set, args = append(set, "file_path_template = ?"), append(args, *v)
	}
	if v := patch.SchemaPathTemplate; v != nil {
		set, args = append(set, "schema_path_template = ?"), append(args, *v)
	}
	if v := patch.AccessToken; v != nil {
		set, args = append(set, "access_token = ?"), append(args, *v)
	}
	if v := patch.ExpiresTs; v != nil {
		set, args = append(set, "expires_ts = ?"), append(args, *v)
	}
	if v := patch.RefreshToken; v != nil {
		set, args = append(set, "refresh_token = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE repository
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var repository api.Repository
		if err := row.Scan(
			&repository.ID,
			&repository.CreatorID,
			&repository.CreatedTs,
			&repository.UpdaterID,
			&repository.UpdatedTs,
			&repository.VCSID,
			&repository.ProjectID,
			&repository.Name,
			&repository.FullPath,
			&repository.WebURL,
			&repository.BranchFilter,
			&repository.BaseDirectory,
			&repository.FilePathTemplate,
			&repository.SchemaPathTemplate,
			&repository.ExternalID,
			&repository.ExternalWebhookID,
			&repository.WebhookURLHost,
			&repository.WebhookEndpointID,
			&repository.WebhookSecretToken,
			&repository.AccessToken,
			&repository.ExpiresTs,
			&repository.RefreshToken,
		); err != nil {
			return nil, FormatError(err)
		}

		return &repository, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("repository ID not found: %d", patch.ID)}
}

// deleteRepository permanently deletes a repository by ID.
func (s *RepositoryService) deleteRepository(ctx context.Context, tx *Tx, delete *api.RepositoryDelete) error {
	// Updates the project workflow_type to "UI"
	workflowType := api.UIWorkflow
	projectPatch := api.ProjectPatch{
		ID:           delete.ProjectID,
		UpdaterID:    delete.DeleterID,
		WorkflowType: &workflowType,
	}
	if _, err := s.projectService.PatchProjectTx(ctx, tx.Tx, &projectPatch); err != nil {
		return err
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM repository WHERE project_id = ?`, delete.ProjectID); err != nil {
		return FormatError(err)
	}
	return nil
}
