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

// RepositoryMessage is the message for a repository.
type RepositoryMessage struct {
	// Output only
	UID int

	// Related fields
	VCSUID            int
	ProjectResourceID string

	// Domain specific fields
	Title              string
	FullPath           string
	WebURL             string
	BranchFilter       string
	BaseDirectory      string
	FilePathTemplate   string
	SchemaPathTemplate string
	SheetPathTemplate  string
	EnableSQLReviewCI  bool
	EnableCD           bool
	ExternalID         string
	ExternalWebhookID  string
	WebhookURLHost     string
	WebhookEndpointID  string
	WebhookSecretToken string
	AccessToken        string
	ExpiresTs          int64
	RefreshToken       string
}

// FindRepositoryMessage is the message for finding repositories.
type FindRepositoryMessage struct {
	UID               *int
	WebURL            *string
	VCSUID            *int
	ProjectResourceID *string
	WebhookEndpointID *string
}

// PatchRepositoryMessage is the API message for patching a repository.
type PatchRepositoryMessage struct {
	UID    *int
	WebURL *string

	// Domain specific fields
	BranchFilter       *string
	BaseDirectory      *string
	FilePathTemplate   *string
	SchemaPathTemplate *string
	SheetPathTemplate  *string
	EnableSQLReviewCI  *bool
	EnableCD           *bool
	AccessToken        *string
	ExpiresTs          *int64
	RefreshToken       *string
}

// CreateRepositoryV2 creates the repository.
func (s *Store) CreateRepositoryV2(ctx context.Context, create *RepositoryMessage, creatorUID int) (*RepositoryMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	repository, err := s.createRepositoryImplV2(ctx, tx, create, creatorUID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.removeProjectCache(create.ProjectResourceID)

	return repository, nil
}

// GetRepositoryV2 gets an instance of repository.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) GetRepositoryV2(ctx context.Context, find *FindRepositoryMessage) (*RepositoryMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := s.listRepositoryImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d repositories with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// ListRepositoryV2 lists repository.
func (s *Store) ListRepositoryV2(ctx context.Context, find *FindRepositoryMessage) ([]*RepositoryMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := s.listRepositoryImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

// PatchRepositoryV2 patches an instance of Repository.
func (s *Store) PatchRepositoryV2(ctx context.Context, patch *PatchRepositoryMessage, updaterID int) (*RepositoryMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	repository, err := s.patchRepositoryImplV2(ctx, tx, patch, updaterID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return repository, nil
}

// DeleteRepositoryV2 deletes an existing repository by ID.
func (s *Store) DeleteRepositoryV2(ctx context.Context, projectResourceID string, deleterID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.deleteRepositoryImplV2(ctx, tx, projectResourceID, deleterID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.removeProjectCache(projectResourceID)

	return nil
}

// createRepositoryImplV2 creates a new repository.
func (s *Store) createRepositoryImplV2(ctx context.Context, tx *Tx, create *RepositoryMessage, creatorID int) (*RepositoryMessage, error) {
	// Updates the project workflow_type to "VCS"
	// TODO(d): ideally, we should not update project fields on repository changes.
	workflowType := api.VCSWorkflow
	update := &UpdateProjectMessage{
		UpdaterID:  creatorID,
		ResourceID: create.ProjectResourceID,
		Workflow:   &workflowType,
	}
	project, err := s.updateProjectImplV2(ctx, tx, update)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("cannot found project %s", create.ProjectResourceID)}
	}

	repository := RepositoryMessage{
		ProjectResourceID: project.ResourceID,
	}
	// Insert row into database.
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
			enable_cd,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token,
			access_token,
			expires_ts,
			refresh_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, vcs_id, name, full_path, web_url, branch_filter, base_directory, file_path_template, schema_path_template, sheet_path_template, enable_sql_review_ci, enable_cd, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token, access_token, expires_ts, refresh_token
	`
	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.VCSUID,
		project.UID,
		create.Title,
		create.FullPath,
		create.WebURL,
		create.BranchFilter,
		create.BaseDirectory,
		create.FilePathTemplate,
		create.SchemaPathTemplate,
		create.SheetPathTemplate,
		false, /* EnableSQLReviewCI */
		true,  /* EnableCD */
		create.ExternalID,
		create.ExternalWebhookID,
		create.WebhookURLHost,
		create.WebhookEndpointID,
		create.WebhookSecretToken,
		create.AccessToken,
		create.ExpiresTs,
		create.RefreshToken,
	).Scan(
		&repository.UID,
		&repository.VCSUID,
		&repository.Title,
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
		return nil, err
	}
	return &repository, nil
}

func (*Store) listRepositoryImplV2(ctx context.Context, tx *Tx, find *FindRepositoryMessage) ([]*RepositoryMessage, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.VCSUID; v != nil {
		where, args = append(where, fmt.Sprintf("vcs_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.WebURL; v != nil {
		where, args = append(where, fmt.Sprintf("web_url = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.WebhookEndpointID; v != nil {
		where, args = append(where, fmt.Sprintf("webhook_endpoint_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			repository.id AS id,
			vcs_id,
			project.resource_id AS project_resource_id,
			repository.name AS name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			file_path_template,
			schema_path_template,
			sheet_path_template,
			enable_sql_review_ci,
			enable_cd,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token,
			access_token,
			expires_ts,
			refresh_token
		FROM repository
		LEFT JOIN project ON project.id = repository.project_id
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into repoRawList.
	var repoRawList []*RepositoryMessage
	for rows.Next() {
		var repository RepositoryMessage
		if err := rows.Scan(
			&repository.UID,
			&repository.VCSUID,
			&repository.ProjectResourceID,
			&repository.Title,
			&repository.FullPath,
			&repository.WebURL,
			&repository.BranchFilter,
			&repository.BaseDirectory,
			&repository.FilePathTemplate,
			&repository.SchemaPathTemplate,
			&repository.SheetPathTemplate,
			&repository.EnableSQLReviewCI,
			&repository.EnableCD,
			&repository.ExternalID,
			&repository.ExternalWebhookID,
			&repository.WebhookURLHost,
			&repository.WebhookEndpointID,
			&repository.WebhookSecretToken,
			&repository.AccessToken,
			&repository.ExpiresTs,
			&repository.RefreshToken,
		); err != nil {
			return nil, err
		}

		repoRawList = append(repoRawList, &repository)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repoRawList, nil
}

// patchRepositoryImpl updates a repository by ID. Returns the new state of the repository after update.
// Returns ENOTFOUND if repository does not exist.
func (*Store) patchRepositoryImplV2(ctx context.Context, tx *Tx, patch *PatchRepositoryMessage, updaterID int) (*RepositoryMessage, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{updaterID}
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
	if v := patch.AccessToken; v != nil {
		set, args = append(set, fmt.Sprintf("access_token = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ExpiresTs; v != nil {
		set, args = append(set, fmt.Sprintf("expires_ts = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.RefreshToken; v != nil {
		set, args = append(set, fmt.Sprintf("refresh_token = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EnableSQLReviewCI; v != nil {
		set, args = append(set, fmt.Sprintf("enable_sql_review_ci = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.EnableCD; v != nil {
		set, args = append(set, fmt.Sprintf("enable_cd = $%d", len(args)+1)), append(args, *v)
	}

	where := []string{}
	if v := patch.UID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.WebURL; v != nil {
		where, args = append(where, fmt.Sprintf("web_url = $%d", len(args)+1)), append(args, *v)
	}
	if len(where) == 0 {
		return nil, common.Errorf(common.Invalid, "missing predicate in where clause for patching repository")
	}

	var repository RepositoryMessage
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE repository
		SET `+strings.Join(set, ", ")+`
		FROM project
		WHERE project.id = repository.project_id AND `+strings.Join(where, " AND ")+`
		RETURNING
			repository.id AS id,
			vcs_id,
			project.resource_id AS project_resource_id,
			repository.name AS name,
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
		`,
		args...,
	).Scan(
		&repository.UID,
		&repository.VCSUID,
		&repository.ProjectResourceID,
		&repository.Title,
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
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("repository ID not found: %d", patch.UID)}
		}
		return nil, err
	}
	return &repository, nil
}

// deleteRepositoryImplV2 permanently deletes a repository by ID.
func (s *Store) deleteRepositoryImplV2(ctx context.Context, tx *Tx, projectResourceID string, deleterID int) error {
	// Updates the project workflow_type to "UI"
	// TODO(d): ideally, we should not update project fields on repository changes.
	workflowType := api.UIWorkflow
	update := &UpdateProjectMessage{
		UpdaterID:  deleterID,
		ResourceID: projectResourceID,
		Workflow:   &workflowType,
	}
	project, err := s.updateProjectImplV2(ctx, tx, update)
	if err != nil {
		return err
	}
	if project == nil {
		return &common.Error{Code: common.NotFound, Err: errors.Errorf("cannot found project %s", projectResourceID)}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM repository WHERE project_id = $1`, project.UID); err != nil {
		return err
	}
	return nil
}
