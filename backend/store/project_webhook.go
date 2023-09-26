package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// ProjectWebhookMessage is the store model for an project webhook.
type ProjectWebhookMessage struct {
	// Type is the webhook type (e.g. SLACK, DISCORD, etc.).
	Type string
	// Title is the webhook name.
	Title string
	// URL is the webhook URL.
	URL string
	// ActivityList is the list of activities that the webhook is interested in.
	ActivityList []string
	// Output only fields.
	//
	// ID is the unique identifier of the project webhook.
	ID        int
	ProjectID int
}

// UpdateProjectWebhookMessage is the message for updating project webhooks.
type UpdateProjectWebhookMessage struct {
	// Title is the webhook name.
	Title *string
	// URL is the webhook URL.
	URL *string
	// ActivityList is the list of activities that the webhook is interested in.
	ActivityList []string
}

// FindProjectWebhookMessage is the message for finding project webhooks,
// if all fields are nil, it will list all project webhooks.
type FindProjectWebhookMessage struct {
	ID           *int
	ProjectID    *int
	URL          *string
	ActivityType *api.ActivityType
}

// CreateProjectWebhookV2 creates an instance of ProjectWebhook.
func (s *Store) CreateProjectWebhookV2(ctx context.Context, principalUID int, projectUID int, projectResourceID string, create *ProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	query := `
		INSERT INTO project_webhook (
			creator_id,
			updater_id,
			project_id,
			type,
			name,
			url,
			activity_list
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, project_id, type, name, url, activity_list
	`
	var projectWebhook ProjectWebhookMessage
	var txtArray pgtype.TextArray

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	if err := tx.QueryRowContext(ctx, query,
		principalUID,
		principalUID,
		projectUID,
		create.Type,
		create.Title,
		create.URL,
		create.ActivityList,
	).Scan(
		&projectWebhook.ID,
		&projectWebhook.ProjectID,
		&projectWebhook.Type,
		&projectWebhook.Title,
		&projectWebhook.URL,
		&txtArray,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := txtArray.AssignTo(&projectWebhook.ActivityList); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectResourceID)
	return &projectWebhook, nil
}

// FindProjectWebhookV2 finds a list of ProjectWebhook instances.
func (s *Store) FindProjectWebhookV2(ctx context.Context, find *FindProjectWebhookMessage) ([]*ProjectWebhookMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
func (s *Store) UpdateProjectWebhookV2(ctx context.Context, principalUID int, projectResourceID string, projectWebhookID int, update *UpdateProjectWebhookMessage) (*ProjectWebhookMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{principalUID}
	if v := update.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.URL; v != nil {
		set, args = append(set, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.ActivityList; v != nil {
		set, args = append(set, fmt.Sprintf("activity_list = $%d", len(args)+1)), append(args, v)
	}

	args = append(args, projectWebhookID)

	var projectWebhook ProjectWebhookMessage
	var txtArray pgtype.TextArray
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
	UPDATE project_webhook
	SET `+strings.Join(set, ", ")+`
	WHERE id = $%d
	RETURNING id, project_id, type, name, url, activity_list
`, len(args)),
		args...,
	).Scan(
		&projectWebhook.ID,
		&projectWebhook.ProjectID,
		&projectWebhook.Type,
		&projectWebhook.Title,
		&projectWebhook.URL,
		&txtArray,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("project hook ID not found: %d", projectWebhookID)}
		}
		return nil, err
	}
	if err := txtArray.AssignTo(&projectWebhook.ActivityList); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	s.removeProjectCache(projectResourceID)
	return &projectWebhook, nil
}

// DeleteProjectWebhookV2 deletes an existing projectWebhook by projectUID and url.
func (s *Store) DeleteProjectWebhookV2(ctx context.Context, projectResourceID string, projectWebhookUID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
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

func (*Store) findProjectWebhookImplV2(ctx context.Context, tx *Tx, find *FindProjectWebhookMessage) ([]*ProjectWebhookMessage, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.URL; v != nil {
		where, args = append(where, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			type,
			name,
			url,
			activity_list
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
		var projectWebhook ProjectWebhookMessage
		var txtArray pgtype.TextArray

		if err := rows.Scan(
			&projectWebhook.ID,
			&projectWebhook.Type,
			&projectWebhook.Title,
			&projectWebhook.URL,
			&txtArray,
		); err != nil {
			return nil, err
		}

		if err := txtArray.AssignTo(&projectWebhook.ActivityList); err != nil {
			return nil, err
		}

		if v := find.ActivityType; v != nil {
			for _, activity := range projectWebhook.ActivityList {
				if api.ActivityType(activity) == *v {
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
