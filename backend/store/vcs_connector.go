package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// VCSConnectorMessage is the message for a VCS connector.
type VCSConnectorMessage struct {
	// Related fields
	VCSUID        int
	VCSResourceID string
	ProjectID     string
	ResourceID    string

	Payload *storepb.VCSConnector

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
		where, args = append(where, fmt.Sprintf("vcs_connector.vcs_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("vcs_connector.resource_id = $%d", len(args)+1)), append(args, *v)
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
			vcs_connector.id AS id,
			vcs_id,
			vcs.resource_id,
			project.resource_id AS project_resource_id,
			vcs_connector.resource_id,
			vcs_connector.creator_id,
			vcs_connector.updater_id,
			vcs_connector.payload
		FROM vcs_connector
		LEFT JOIN project ON project.id = vcs_connector.project_id
		LEFT JOIN vcs ON vcs.id = vcs_connector.vcs_id
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
		var payloadStr string
		if err := rows.Scan(
			&vcsConnector.UID,
			&vcsConnector.VCSUID,
			&vcsConnector.VCSResourceID,
			&vcsConnector.ProjectID,
			&vcsConnector.ResourceID,
			&vcsConnector.CreatorID,
			&vcsConnector.UpdaterID,
			&payloadStr,
		); err != nil {
			return nil, err
		}
		var payload storepb.VCSConnector
		if err := protojson.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return nil, err
		}
		vcsConnector.Payload = &payload
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

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO vcs_connector (
			creator_id,
			updater_id,
			vcs_id,
			project_id,
			resource_id,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.UpdaterID,
		create.VCSUID,
		project.UID,
		create.ResourceID,
		payload,
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

	var payloadSet []string
	if v := update.Branch; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf("jsonb_build_object('branch', to_jsonb($%d::TEXT))", len(args)+1)), append(args, *v)
	}
	if v := update.BaseDirectory; v != nil {
		payloadSet, args = append(payloadSet, fmt.Sprintf("jsonb_build_object('baseDirectory', to_jsonb($%d::TEXT))", len(args)+1)), append(args, *v)
	}
	if len(payloadSet) != 0 {
		set = append(set, fmt.Sprintf(`payload = payload || %s`, strings.Join(payloadSet, "||")))
	}

	where := []string{}
	where, args = append(where, fmt.Sprintf("vcs_connector.id = $%d", len(args)+1)), append(args, update.UID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	query := `
		UPDATE vcs_connector
		SET ` + strings.Join(set, ", ") + `
		FROM project, vcs
		WHERE project.id = vcs_connector.project_id AND vcs.id = vcs_connector.vcs_id AND ` + strings.Join(where, " AND ")

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
		DELETE FROM vcs_connector
		USING project
		WHERE vcs_connector.project_id = project.id AND project.resource_id = $1 AND vcs_connector.resource_id = $2;`,
		projectID, resourceID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.removeProjectCache(projectID)
	return nil
}
