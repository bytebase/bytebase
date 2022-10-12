package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// repositoryRaw is the store model for a Repository.
// Fields have exactly the same meanings as Repository.
type repositoryRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	VCSID     int
	ProjectID int

	// Domain specific fields
	Name               string
	FullPath           string
	WebURL             string
	BranchFilter       string
	BaseDirectory      string
	FilePathTemplate   string
	SchemaPathTemplate string
	SheetPathTemplate  string
	EnableSQLReviewCI  bool
	ExternalID         string
	ExternalWebhookID  string
	WebhookURLHost     string
	WebhookEndpointID  string
	WebhookSecretToken string
	AccessToken        string
	ExpiresTs          int64
	RefreshToken       string
}

// toRepository creates an instance of Repository based on the repositoryRaw.
// This is intended to be called when we need to compose a Repository relationship.
func (raw *repositoryRaw) toRepository() *api.Repository {
	return &api.Repository{
		ID: raw.ID,

		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		VCSID:     raw.VCSID,
		ProjectID: raw.ProjectID,

		Name:               raw.Name,
		FullPath:           raw.FullPath,
		WebURL:             raw.WebURL,
		BranchFilter:       raw.BranchFilter,
		BaseDirectory:      raw.BaseDirectory,
		FilePathTemplate:   raw.FilePathTemplate,
		SchemaPathTemplate: raw.SchemaPathTemplate,
		SheetPathTemplate:  raw.SheetPathTemplate,
		EnableSQLReviewCI:  raw.EnableSQLReviewCI,
		ExternalID:         raw.ExternalID,
		ExternalWebhookID:  raw.ExternalWebhookID,
		WebhookURLHost:     raw.WebhookURLHost,
		WebhookEndpointID:  raw.WebhookEndpointID,
		WebhookSecretToken: raw.WebhookSecretToken,
		AccessToken:        raw.AccessToken,
		ExpiresTs:          raw.ExpiresTs,
		RefreshToken:       raw.RefreshToken,
	}
}

// CreateRepository creates an instance of Repository.
func (s *Store) CreateRepository(ctx context.Context, create *api.RepositoryCreate) (*api.Repository, error) {
	repositoryRaw, err := s.createRepositoryRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Repository with RepositoryCreate[%+v]", create)
	}
	repository, err := s.composeRepository(ctx, repositoryRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Repository with repositoryRaw[%+v]", repositoryRaw)
	}
	return repository, nil
}

// GetRepository gets an instance of Repository.
func (s *Store) GetRepository(ctx context.Context, find *api.RepositoryFind) (*api.Repository, error) {
	repositoryRaw, err := s.getRepositoryRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Repository with RepositoryFind[%+v]", find)
	}
	if repositoryRaw == nil {
		return nil, nil
	}
	repository, err := s.composeRepository(ctx, repositoryRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Repository with repositoryRaw[%+v]", repositoryRaw)
	}
	return repository, nil
}

// FindRepository finds a list of Repository instances.
func (s *Store) FindRepository(ctx context.Context, find *api.RepositoryFind) ([]*api.Repository, error) {
	repositoryRawList, err := s.findRepositoryRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Repository list with RepositoryFind[%+v]", find)
	}
	var repositoryList []*api.Repository
	for _, raw := range repositoryRawList {
		repository, err := s.composeRepository(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Repository with repositoryRaw[%+v]", raw)
		}
		repositoryList = append(repositoryList, repository)
	}
	return repositoryList, nil
}

// PatchRepository patches an instance of Repository.
func (s *Store) PatchRepository(ctx context.Context, patch *api.RepositoryPatch) (*api.Repository, error) {
	repositoryRaw, err := s.patchRepositoryRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Repository with RepositoryPatch[%+v]", patch)
	}
	repository, err := s.composeRepository(ctx, repositoryRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Repository with repositoryRaw[%+v]", repositoryRaw)
	}
	return repository, nil
}

// DeleteRepository deletes an existing repository by ID.
func (s *Store) DeleteRepository(ctx context.Context, delete *api.RepositoryDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	if err := s.deleteRepositoryImpl(ctx, tx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

//
// private functions
//

func (s *Store) composeRepository(ctx context.Context, raw *repositoryRaw) (*api.Repository, error) {
	repository := raw.toRepository()

	creator, err := s.GetPrincipalByID(ctx, repository.CreatorID)
	if err != nil {
		return nil, err
	}
	repository.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, repository.UpdaterID)
	if err != nil {
		return nil, err
	}
	repository.Updater = updater

	vcs, err := s.GetVCSByID(ctx, repository.VCSID)
	if err != nil {
		return nil, err
	}
	// We should always expect VCS to exist when ID isn't the default zero.
	if repository.VCSID > 0 && vcs == nil {
		return nil, errors.Errorf("VCS not found for ID: %v", repository.VCSID)
	}
	repository.VCS = vcs

	project, err := s.GetProjectByID(ctx, repository.ProjectID)
	if err != nil {
		return nil, err
	}
	repository.Project = project

	return repository, nil
}

// createRepositoryRaw creates a new repository.
func (s *Store) createRepositoryRaw(ctx context.Context, create *api.RepositoryCreate) (*repositoryRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	repository, err := s.createRepositoryImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return repository, nil
}

// findRepositoryRaw retrieves a list of repositories based on find.
func (s *Store) findRepositoryRaw(ctx context.Context, find *api.RepositoryFind) ([]*repositoryRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findRepositoryImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getRepositoryRaw retrieves a single repository based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getRepositoryRaw(ctx context.Context, find *api.RepositoryFind) (*repositoryRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findRepositoryImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d repositories with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchRepositoryRaw updates an existing repository by ID.
// Returns ENOTFOUND if repository does not exist.
func (s *Store) patchRepositoryRaw(ctx context.Context, patch *api.RepositoryPatch) (*repositoryRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	repository, err := patchRepositoryImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return repository, nil
}

// createRepositoryImpl creates a new repository.
func (s *Store) createRepositoryImpl(ctx context.Context, tx *Tx, create *api.RepositoryCreate) (*repositoryRaw, error) {
	// Updates the project workflow_type to "VCS"
	workflowType := string(api.VCSWorkflow)
	projectPatch := api.ProjectPatch{
		ID:           create.ProjectID,
		UpdaterID:    create.CreatorID,
		WorkflowType: &workflowType,
	}
	if _, err := s.patchProjectRawTx(ctx, tx, &projectPatch); err != nil {
		return nil, err
	}

	var repository repositoryRaw
	// Insert row into database.
	if s.db.mode == common.ReleaseModeDev {
		query := `
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
				sheet_path_template,
				enable_sql_review_ci,
				external_id,
				external_webhook_id,
				webhook_url_host,
				webhook_endpoint_id,
				webhook_secret_token,
				access_token,
				expires_ts,
				refresh_token
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
			RETURNING id, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, sheet_path_template, enable_sql_review_ci, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
		`
		if err := tx.QueryRowContext(ctx, query,
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
			create.SheetPathTemplate,
			create.EnableSQLReviewCI,
			create.ExternalID,
			create.ExternalWebhookID,
			create.WebhookURLHost,
			create.WebhookEndpointID,
			create.WebhookSecretToken,
			create.AccessToken,
			create.ExpiresTs,
			create.RefreshToken,
		).Scan(
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
			&repository.SheetPathTemplate,
			&repository.EnableSQLReviewCI,
			&repository.ExternalID,
			&repository.ExternalWebhookID,
			&repository.WebhookURLHost,
			&repository.WebhookEndpointID,
			&repository.WebhookSecretToken,
			&repository.AccessToken,
			&repository.ExpiresTs,
			&repository.RefreshToken,
		); err != nil {
			if err == sql.ErrNoRows {
				return nil, common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return nil, FormatError(err)
		}
		return &repository, nil
	}
	query := `
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
			enable_sql_review_ci,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token,
			access_token,
			expires_ts,
			refresh_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, enable_sql_review_ci, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
	`
	if err := tx.QueryRowContext(ctx, query,
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
		create.EnableSQLReviewCI,
		create.ExternalID,
		create.ExternalWebhookID,
		create.WebhookURLHost,
		create.WebhookEndpointID,
		create.WebhookSecretToken,
		create.AccessToken,
		create.ExpiresTs,
		create.RefreshToken,
	).Scan(
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
		&repository.EnableSQLReviewCI,
		&repository.ExternalID,
		&repository.ExternalWebhookID,
		&repository.WebhookURLHost,
		&repository.WebhookEndpointID,
		&repository.WebhookSecretToken,
		&repository.AccessToken,
		&repository.ExpiresTs,
		&repository.RefreshToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &repository, nil
}

func findRepositoryImpl(ctx context.Context, tx *Tx, find *api.RepositoryFind) ([]*repositoryRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.VCSID; v != nil {
		where, args = append(where, fmt.Sprintf("vcs_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.WebhookEndpointID; v != nil {
		where, args = append(where, fmt.Sprintf("webhook_endpoint_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.WebURL; v != nil {
		where, args = append(where, fmt.Sprintf("web_url = $%d", len(args)+1)), append(args, *v)
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
			sheet_path_template,
			enable_sql_review_ci,
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

	// Iterate over result set and deserialize rows into repoRawList.
	var repoRawList []*repositoryRaw
	for rows.Next() {
		var repository repositoryRaw
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
			&repository.SheetPathTemplate,
			&repository.EnableSQLReviewCI,
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

		repoRawList = append(repoRawList, &repository)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return repoRawList, nil
}

// patchRepositoryImpl updates a repository by ID. Returns the new state of the repository after update.
func patchRepositoryImpl(ctx context.Context, tx *Tx, patch *api.RepositoryPatch) (*repositoryRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.BranchFilter; v != nil {
		set, args = append(set, fmt.Sprintf("branch_filter = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.BaseDirectory; v != nil {
		set, args = append(set, fmt.Sprintf("base_directory = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.FilePathTemplate; v != nil {
		set, args = append(set, fmt.Sprintf("file_path_template = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SchemaPathTemplate; v != nil {
		set, args = append(set, fmt.Sprintf("schema_path_template = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.SheetPathTemplate; v != nil {
		set, args = append(set, fmt.Sprintf("sheet_path_template = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EnableSQLReviewCI; v != nil {
		set, args = append(set, fmt.Sprintf("enable_sql_review_ci = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.AccessToken; v != nil {
		set, args = append(set, fmt.Sprintf("access_token = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExpiresTs; v != nil {
		set, args = append(set, fmt.Sprintf("expires_ts = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.RefreshToken; v != nil {
		set, args = append(set, fmt.Sprintf("refresh_token = $%d", len(args)+1)), append(args, *v)
	}
	where := []string{}
	if v := patch.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.WebURL; v != nil {
		where, args = append(where, fmt.Sprintf("web_url = $%d", len(args)+1)), append(args, *v)
	}
	if len(where) == 0 {
		return nil, common.Errorf(common.Invalid, "missing predicate in where clause for patching repository")
	}

	var repository repositoryRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE repository
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, vcs_id, project_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, sheet_path_template, enable_sql_review_ci, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
		`,
		args...,
	).Scan(
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
		&repository.SheetPathTemplate,
		&repository.EnableSQLReviewCI,
		&repository.ExternalID,
		&repository.ExternalWebhookID,
		&repository.WebhookURLHost,
		&repository.WebhookEndpointID,
		&repository.WebhookSecretToken,
		&repository.AccessToken,
		&repository.ExpiresTs,
		&repository.RefreshToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("repository ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &repository, nil
}

// deleteRepositoryImpl permanently deletes a repository by ID.
func (s *Store) deleteRepositoryImpl(ctx context.Context, tx *Tx, delete *api.RepositoryDelete) error {
	// Updates the project workflow_type to "UI"
	workflowType := string(api.UIWorkflow)
	projectPatch := api.ProjectPatch{
		ID:           delete.ProjectID,
		UpdaterID:    delete.DeleterID,
		WorkflowType: &workflowType,
	}
	if _, err := s.patchProjectRawTx(ctx, tx, &projectPatch); err != nil {
		return err
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM repository WHERE project_id = $1`, delete.ProjectID); err != nil {
		return FormatError(err)
	}
	return nil
}
