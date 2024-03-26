package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// RepositoryMessage is the message for a repository.
type RepositoryMessage struct {
	// Related fields
	VCSUID            int
	VCSResourceID     string
	ProjectResourceID string
	ResourceID        string

	// Domain specific fields
	Title              string
	FullPath           string
	WebURL             string
	BranchFilter       string
	BaseDirectory      string
	ExternalID         string
	ExternalWebhookID  string
	WebhookURLHost     string
	WebhookEndpointID  string
	WebhookSecretToken string

	// Output only
	UID int
}

// FindRepositoryMessage is the message for finding repositories.
type FindRepositoryMessage struct {
	UID               *int
	WebURL            *string
	VCSUID            *int
	VCSResourceID     *string
	ProjectResourceID *string
	ResourceID        *string
	WebhookEndpointID *string
}

// PatchRepositoryMessage is the API message for patching a repository.
type PatchRepositoryMessage struct {
	ResourceID *string
	UID        *int
	WebURL     *string

	// Domain specific fields
	BranchFilter  *string
	BaseDirectory *string
}

// CreateRepositoryV2 creates the repository.
func (s *Store) CreateRepositoryV2(ctx context.Context, create *RepositoryMessage, creatorUID int) (*RepositoryMessage, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectResourceID})
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	repository, err := createRepositoryImplV2(ctx, tx, project, create, creatorUID)
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
func (s *Store) DeleteRepositoryV2(ctx context.Context, projectResourceID string) error {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &projectResourceID})
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteAllRepositoryImplV2(ctx, tx, project); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.removeProjectCache(projectResourceID)
	return nil
}

// createRepositoryImplV2 creates a new repository.
func createRepositoryImplV2(ctx context.Context, tx *Tx, project *ProjectMessage, create *RepositoryMessage, creatorID int) (*RepositoryMessage, error) {
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
			resource_id,
			name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, vcs_id, resource_id, name, full_path, web_url, branch_filter, base_directory, external_id, external_webhook_id, webhook_url_host, webhook_endpoint_id, webhook_secret_token
	`
	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.VCSUID,
		project.UID,
		create.ResourceID,
		create.Title,
		create.FullPath,
		create.WebURL,
		create.BranchFilter,
		create.BaseDirectory,
		create.ExternalID,
		create.ExternalWebhookID,
		create.WebhookURLHost,
		create.WebhookEndpointID,
		create.WebhookSecretToken,
	).Scan(
		&repository.UID,
		&repository.VCSUID,
		&repository.ResourceID,
		&repository.Title,
		&repository.FullPath,
		&repository.WebURL,
		&repository.BranchFilter,
		&repository.BaseDirectory,
		&repository.ExternalID,
		&repository.ExternalWebhookID,
		&repository.WebhookURLHost,
		&repository.WebhookEndpointID,
		&repository.WebhookSecretToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	repository.VCSResourceID = create.VCSResourceID
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
	if v := find.VCSResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("vcs.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
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
			vcs.resource_id,
			project.resource_id AS project_resource_id,
			repository.resource_id,
			repository.name AS name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token
		FROM repository
		LEFT JOIN project ON project.id = repository.project_id
		LEFT JOIN vcs ON vcs.id = repository.vcs_id
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
			&repository.VCSResourceID,
			&repository.ProjectResourceID,
			&repository.ResourceID,
			&repository.Title,
			&repository.FullPath,
			&repository.WebURL,
			&repository.BranchFilter,
			&repository.BaseDirectory,
			&repository.ExternalID,
			&repository.ExternalWebhookID,
			&repository.WebhookURLHost,
			&repository.WebhookEndpointID,
			&repository.WebhookSecretToken,
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

	where := []string{}
	if v := patch.UID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.resource_id = $%d", len(args)+1)), append(args, *v)
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
		FROM project, vcs
		WHERE project.id = repository.project_id AND vcs.id = repository.vcs_id AND `+strings.Join(where, " AND ")+`
		RETURNING
			repository.id AS id,
			vcs_id,
			vcs.resource_id,
			project.resource_id AS project_resource_id,
			repository.resource_id,
			repository.name AS name,
			full_path,
			web_url,
			branch_filter,
			base_directory,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token
		`,
		args...,
	).Scan(
		&repository.UID,
		&repository.VCSUID,
		&repository.VCSResourceID,
		&repository.ProjectResourceID,
		&repository.ResourceID,
		&repository.Title,
		&repository.FullPath,
		&repository.WebURL,
		&repository.BranchFilter,
		&repository.BaseDirectory,
		&repository.ExternalID,
		&repository.ExternalWebhookID,
		&repository.WebhookURLHost,
		&repository.WebhookEndpointID,
		&repository.WebhookSecretToken,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("repository ID not found: %d", patch.UID)}
		}
		return nil, err
	}
	return &repository, nil
}

// deleteAllRepositoryImplV2 permanently deletes a repository by ID.
func deleteAllRepositoryImplV2(ctx context.Context, tx *Tx, project *ProjectMessage) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM repository WHERE project_id = $1`, project.UID); err != nil {
		return err
	}
	return nil
}
