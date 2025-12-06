package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
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

// CreateProjectWebhook creates an instance of ProjectWebhook.
func (s *Store) CreateProjectWebhook(ctx context.Context, projectID string, create *ProjectWebhookMessage) (*ProjectWebhookMessage, error) {
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

	q := qb.Q().Space(`
		INSERT INTO project_webhook (
			project,
			payload
		)
		VALUES (?, ?)
		RETURNING id
	`, projectID, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	if err := tx.QueryRowContext(ctx, query, args...).Scan(
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

// ListProjectWebhooks lists a list of ProjectWebhook instances.
func (s *Store) ListProjectWebhooks(ctx context.Context, find *FindProjectWebhookMessage) ([]*ProjectWebhookMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	q := qb.Q().Space(`
		SELECT
			id,
			project,
			payload
		FROM project_webhook
		WHERE TRUE
	`)

	if v := find.ID; v != nil {
		q.And("id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		q.And("project = ?", *v)
	}
	if v := find.URL; v != nil {
		q.And("payload->>'url' = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projectWebhooks []*ProjectWebhookMessage // Declare it here
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
	webhooks := projectWebhooks

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return webhooks, nil
}

// GetProjectWebhook gets an instance of ProjectWebhook.
func (s *Store) GetProjectWebhook(ctx context.Context, find *FindProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	webhooks, err := s.ListProjectWebhooks(ctx, find)
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

// UpdateProjectWebhook updates an instance of ProjectWebhook.
func (s *Store) UpdateProjectWebhook(ctx context.Context, projectResourceID string, projectWebhookID int, update *UpdateProjectWebhookMessage) (*ProjectWebhookMessage, error) {
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

	q := qb.Q().Space(`
		UPDATE project_webhook
		SET payload = ?
		WHERE id = ?
		RETURNING id, project, payload
	`, payload, projectWebhookID)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	projectWebhook := ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{},
	}
	var returnedPayload []byte
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
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

// DeleteProjectWebhook deletes an existing projectWebhook by projectUID and url.
func (s *Store) DeleteProjectWebhook(ctx context.Context, projectResourceID string, projectWebhookUID int) error {
	q := qb.Q().Space("DELETE FROM project_webhook WHERE id = ?", projectWebhookUID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectResourceID)
	return nil
}
