package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// VCSConnectorMessage is the message for a VCS connector.
type VCSConnectorMessage struct {
	// Related fields
	VCSUID        int
	VCSResourceID string
	ProjectID     string
	ResourceID    string

	// Domain specific fields
	Title              string
	FullPath           string
	WebURL             string
	Branch             string
	BaseDirectory      string
	ExternalID         string
	ExternalWebhookID  string
	WebhookURLHost     string
	WebhookEndpointID  string
	WebhookSecretToken string

	// Output only fields
	UID         int
	CreatorID   int
	UpdaterID   int
	CreatedTime time.Time
	UpdatedTime time.Time
}

// FindVCSConnectorMessage is the API message for finding VCS connectors.
type FindVCSConnectorMessage struct {
	VCSUID     *int
	ProjectID  *string
	ResourceID *string
}

// UpdateVCSConnectorMessage is the message to update a VCS connector.
type UpdateVCSConnectorMessage struct {
	ProjectID string
	UpdaterID int
	UID       int

	// Domain specific fields
	Branch        *string
	BaseDirectory *string
}

// GetVCSConnector gets a VCS connector.
func (s *Store) GetVCSConnector(ctx context.Context, find *FindVCSConnectorMessage) (*VCSConnectorMessage, error) {
	vcsConnectors, err := s.ListVCSConnectors(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(vcsConnectors) == 0 {
		return nil, nil
	}
	if len(vcsConnectors) > 1 {
		return nil, errors.Errorf("expected 1 VCS connector, got %d", len(vcsConnectors))
	}
	return vcsConnectors[0], nil
}

// ListVCSConnectors returns a list of VCS connectors.
func (s *Store) ListVCSConnectors(ctx context.Context, find *FindVCSConnectorMessage) ([]*VCSConnectorMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.VCSUID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.vcs_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("repository.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project.resource_id = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
			branch,
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

	var vcsConnectors []*VCSConnectorMessage
	for rows.Next() {
		var vcsConnector VCSConnectorMessage
		if err := rows.Scan(
			&vcsConnector.UID,
			&vcsConnector.VCSUID,
			&vcsConnector.VCSResourceID,
			&vcsConnector.ProjectID,
			&vcsConnector.ResourceID,
			&vcsConnector.Title,
			&vcsConnector.FullPath,
			&vcsConnector.WebURL,
			&vcsConnector.Branch,
			&vcsConnector.BaseDirectory,
			&vcsConnector.ExternalID,
			&vcsConnector.ExternalWebhookID,
			&vcsConnector.WebhookURLHost,
			&vcsConnector.WebhookEndpointID,
			&vcsConnector.WebhookSecretToken,
		); err != nil {
			return nil, err
		}
		vcsConnectors = append(vcsConnectors, &vcsConnector)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return vcsConnectors, nil
}

// CreateVCSConnector creates a VCS connector.
func (s *Store) CreateVCSConnector(ctx context.Context, create *VCSConnectorMessage) (*VCSConnectorMessage, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectID})
	if err != nil {
		return nil, err
	}
	create.UpdaterID = create.CreatorID

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
			branch,
			base_directory,
			external_id,
			external_webhook_id,
			webhook_url_host,
			webhook_endpoint_id,
			webhook_secret_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.UpdaterID,
		create.VCSUID,
		project.UID,
		create.ResourceID,
		create.Title,
		create.FullPath,
		create.WebURL,
		create.Branch,
		create.BaseDirectory,
		create.ExternalID,
		create.ExternalWebhookID,
		create.WebhookURLHost,
		create.WebhookEndpointID,
		create.WebhookSecretToken,
	).Scan(
		&create.UID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.removeProjectCache(create.ProjectID)
	return create, nil
}

// UpdateVCSConnector updates a VCS connector.
func (s *Store) UpdateVCSConnector(ctx context.Context, update *UpdateVCSConnectorMessage) error {
	set, args := []string{"updater_id = $1"}, []any{update.UpdaterID}
	if v := update.Branch; v != nil {
		set, args = append(set, fmt.Sprintf("branch = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.BaseDirectory; v != nil {
		set, args = append(set, fmt.Sprintf("base_directory = $%d", len(args)+1)), append(args, *v)
	}
	where := []string{}
	where, args = append(where, fmt.Sprintf("repository.id = $%d", len(args)+1)), append(args, update.UID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	query := `
		UPDATE repository
		SET ` + strings.Join(set, ", ") + `
		FROM project, vcs
		WHERE project.id = repository.project_id AND vcs.id = repository.vcs_id AND ` + strings.Join(where, " AND ")

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteVCSConnector deletes a VCS connector.
func (s *Store) DeleteVCSConnector(ctx context.Context, projectID, resourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM repository
		USING project
		WHERE repository.project_id = project.id AND project.resource_id = $1 AND repository.resource_id = $2;`,
		projectID, resourceID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.removeProjectCache(projectID)
	return nil
}
