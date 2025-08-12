package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ProjectWebhookMessage is the store model for an project webhook.
type ProjectWebhookMessage struct {
	// Type is the webhook type (e.g. SLACK, DISCORD, etc.).
	Type string
	// Title is the webhook name.
	Title string
	// URL is the webhook URL.
	URL string
	// Events is the list of activities that the webhook is interested in.
	Events []string
	// Output only fields.
	//
	// ID is the unique identifier of the project webhook.
	ID        int
	ProjectID string
	Payload   *storepb.ProjectWebhookPayload
}

// UpdateProjectWebhookMessage is the message for updating project webhooks.
type UpdateProjectWebhookMessage struct {
	// Title is the webhook name.
	Title *string
	// URL is the webhook URL.
	URL *string
	// Events is the list of activities that the webhook is interested in.
	Events  []string
	Payload *storepb.ProjectWebhookPayload
}

// FindProjectWebhookMessage is the message for finding project webhooks,
// if all fields are nil, it will list all project webhooks.
type FindProjectWebhookMessage struct {
	ID        *int
	ProjectID *string
	URL       *string
	EventType *common.EventType
}

// CreateProjectWebhookV2 creates an instance of ProjectWebhook.
func (s *Store) CreateProjectWebhookV2(ctx context.Context, projectID string, create *ProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	query := `
		INSERT INTO project_webhook (
			project,
			type,
			name,
			url,
			event_list,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
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
		Type:      create.Type,
		Title:     create.Title,
		URL:       create.URL,
		Events:    create.Events,
		ProjectID: create.ProjectID,
		Payload:   create.Payload,
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	if err := tx.QueryRowContext(ctx, query,
		projectID,
		create.Type,
		create.Title,
		create.URL,
		create.Events,
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
	set, args := []string{}, []any{}
	if v := update.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.URL; v != nil {
		set, args = append(set, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.Events; v != nil {
		set, args = append(set, fmt.Sprintf("event_list = $%d", len(args)+1)), append(args, v)
	}
	if v := update.Payload; v != nil {
		p, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, p)
	}

	args = append(args, projectWebhookID)

	projectWebhook := ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhookPayload{},
	}
	var txtArray pgtype.TextArray
	var payload []byte
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
	UPDATE project_webhook
	SET `+strings.Join(set, ", ")+`
	WHERE id = $%d
	RETURNING id, project, type, name, url, event_list, payload
`, len(args)),
		args...,
	).Scan(
		&projectWebhook.ID,
		&projectWebhook.ProjectID,
		&projectWebhook.Type,
		&projectWebhook.Title,
		&projectWebhook.URL,
		&txtArray,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project hook ID not found: %d", projectWebhookID)}
		}
		return nil, err
	}
	if err := txtArray.AssignTo(&projectWebhook.Events); err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, projectWebhook.Payload); err != nil {
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
		where, args = append(where, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := txn.QueryContext(ctx, `
		SELECT
			id,
			type,
			name,
			url,
			event_list,
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
			Payload: &storepb.ProjectWebhookPayload{},
		}
		var txtArray pgtype.TextArray
		var payload []byte

		if err := rows.Scan(
			&projectWebhook.ID,
			&projectWebhook.Type,
			&projectWebhook.Title,
			&projectWebhook.URL,
			&txtArray,
			&payload,
		); err != nil {
			return nil, err
		}

		if err := txtArray.AssignTo(&projectWebhook.Events); err != nil {
			return nil, err
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, projectWebhook.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		if v := find.EventType; v != nil {
			for _, activity := range projectWebhook.Events {
				if common.EventType(activity) == *v {
					projectWebhooks = append(projectWebhooks, &projectWebhook)
					break
				}
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
