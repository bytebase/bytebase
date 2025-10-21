package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ProjectWebhookMessage is the store model for an project webhook.
type ProjectWebhookMessage struct {
	// Output only fields.
	//
	// ID is the unique identifier of the project webhook.
	ID        int
	ProjectID string
	Payload   *storepb.ProjectWebhook
}

// UpdateProjectWebhookMessage is the message for updating project webhooks.
type UpdateProjectWebhookMessage struct {
	Payload *storepb.ProjectWebhook
}

// FindProjectWebhookMessage is the message for finding project webhooks,
// if all fields are nil, it will list all project webhooks.
type FindProjectWebhookMessage struct {
	ID        *int
	ProjectID *string
	URL       *string
	EventType *storepb.Activity_Type
}

// CreateProjectWebhookV2 creates an instance of ProjectWebhook.
func (s *Store) CreateProjectWebhookV2(ctx context.Context, projectID string, create *ProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	query := `
		INSERT INTO project_webhook (
			project,
			payload
		)
		VALUES ($1, $2)
		RETURNING id
	`
	payload := []byte("{}")
	if create.Payload != nil {
		p, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal")
		}
		payload = p
	}

	projectWebhook := ProjectWebhookMessage{
		ProjectID: projectID,
		Payload:   create.Payload,
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	if err := tx.QueryRowContext(ctx, query,
		projectID,
		payload,
	).Scan(
		&projectWebhook.ID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectID)
	return &projectWebhook, nil
}

// FindProjectWebhookV2 finds a list of ProjectWebhook instances.
func (s *Store) FindProjectWebhookV2(ctx context.Context, find *FindProjectWebhookMessage) ([]*ProjectWebhookMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	webhooks, err := s.findProjectWebhookImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return webhooks, nil
}

// GetProjectWebhookV2 gets an instance of ProjectWebhook.
func (s *Store) GetProjectWebhookV2(ctx context.Context, find *FindProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	webhooks, err := s.findProjectWebhookImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if len(webhooks) == 0 {
		return nil, nil
	}
	if len(webhooks) > 1 {
		return nil, errors.Errorf("expected find one project webhook wit %+v, but found %d", find, len(webhooks))
	}

	return webhooks[0], nil
}

// UpdateProjectWebhookV2 updates an instance of ProjectWebhook.
func (s *Store) UpdateProjectWebhookV2(ctx context.Context, projectResourceID string, projectWebhookID int, update *UpdateProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	var payload []byte
	if update.Payload != nil {
		p, err := protojson.Marshal(update.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		payload = p
	}

	projectWebhook := ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{},
	}
	var returnedPayload []byte
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE project_webhook
		SET payload = $1
		WHERE id = $2
		RETURNING id, project, payload
	`,
		payload,
		projectWebhookID,
	).Scan(
		&projectWebhook.ID,
		&projectWebhook.ProjectID,
		&returnedPayload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project hook ID not found: %d", projectWebhookID)}
		}
		return nil, err
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal(returnedPayload, projectWebhook.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectResourceID)
	return &projectWebhook, nil
}

// DeleteProjectWebhookV2 deletes an existing projectWebhook by projectUID and url.
func (s *Store) DeleteProjectWebhookV2(ctx context.Context, projectResourceID string, projectWebhookUID int) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM project_webhook WHERE id = $1`, projectWebhookUID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectResourceID)
	return nil
}

func (*Store) findProjectWebhookImplV2(ctx context.Context, txn *sql.Tx, find *FindProjectWebhookMessage) ([]*ProjectWebhookMessage, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.URL; v != nil {
		where, args = append(where, fmt.Sprintf("payload->>'url' = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := txn.QueryContext(ctx, `
		SELECT
			id,
			project,
			payload
		FROM project_webhook
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projectWebhooks []*ProjectWebhookMessage
	for rows.Next() {
		projectWebhook := ProjectWebhookMessage{
			Payload: &storepb.ProjectWebhook{},
		}
		var payload []byte

		if err := rows.Scan(
			&projectWebhook.ID,
			&projectWebhook.ProjectID,
			&payload,
		); err != nil {
			return nil, err
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, projectWebhook.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		if v := find.EventType; v != nil {
			found := false
			for _, activity := range projectWebhook.Payload.Activities {
				if activity == *v {
					found = true
					break
				}
			}
			if found {
				projectWebhooks = append(projectWebhooks, &projectWebhook)
			}
		} else {
			projectWebhooks = append(projectWebhooks, &projectWebhook)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projectWebhooks, nil
}
