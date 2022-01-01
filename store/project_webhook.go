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
	_ api.ProjectWebhookService = (*ProjectWebhookService)(nil)
)

// ProjectWebhookService represents a service for managing projectWebhook.
type ProjectWebhookService struct {
	l  *zap.Logger
	db *DB
}

// NewProjectWebhookService returns a new instance of ProjectWebhookService.
func NewProjectWebhookService(logger *zap.Logger, db *DB) *ProjectWebhookService {
	return &ProjectWebhookService{l: logger, db: db}
}

// CreateProjectWebhook creates a new projectWebhook.
func (s *ProjectWebhookService) CreateProjectWebhook(ctx context.Context, create *api.ProjectWebhookCreate) (*api.ProjectWebhook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectWebhook, err := createProjectWebhook(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhook, nil
}

// FindProjectWebhookList retrieves a list of projectWebhooks based on find.
func (s *ProjectWebhookService) FindProjectWebhookList(ctx context.Context, find *api.ProjectWebhookFind) ([]*api.ProjectWebhook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectWebhookList(ctx, tx, find)
	if err != nil {
		return []*api.ProjectWebhook{}, err
	}

	return list, nil
}

// FindProjectWebhook retrieves a single projectWebhook based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ProjectWebhookService) FindProjectWebhook(ctx context.Context, find *api.ProjectWebhookFind) (*api.ProjectWebhook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectWebhookList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d project hooks with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchProjectWebhook updates an existing projectWebhook by ID.
// Returns ENOTFOUND if projectWebhook does not exist.
func (s *ProjectWebhookService) PatchProjectWebhook(ctx context.Context, patch *api.ProjectWebhookPatch) (*api.ProjectWebhook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectWebhook, err := patchProjectWebhook(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhook, nil
}

// DeleteProjectWebhook deletes an existing projectWebhook by ID.
func (s *ProjectWebhookService) DeleteProjectWebhook(ctx context.Context, delete *api.ProjectWebhookDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteProjectWebhook(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createProjectWebhook creates a new projectWebhook.
func createProjectWebhook(ctx context.Context, tx *Tx, create *api.ProjectWebhookCreate) (*api.ProjectWebhook, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_webhook (
			creator_id,
			updater_id,
			project_id,
			type,
			name,
			url,
			activity_list
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Type,
		create.Name,
		create.URL,
		strings.Join(create.ActivityList, ","),
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectWebhook api.ProjectWebhook
	var activityList string
	if err := row.Scan(
		&projectWebhook.ID,
		&projectWebhook.CreatorID,
		&projectWebhook.CreatedTs,
		&projectWebhook.UpdaterID,
		&projectWebhook.UpdatedTs,
		&projectWebhook.ProjectID,
		&projectWebhook.Type,
		&projectWebhook.Name,
		&projectWebhook.URL,
		&activityList,
	); err != nil {
		return nil, FormatError(err)
	}
	projectWebhook.ActivityList = strings.Split(activityList, ",")

	return &projectWebhook, nil
}

func findProjectWebhookList(ctx context.Context, tx *Tx, find *api.ProjectWebhookFind) (_ []*api.ProjectWebhook, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, "project_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			project_id,
			type,
		    name,
			url,
			activity_list
		FROM project_webhook
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.ProjectWebhook, 0)
	for rows.Next() {
		var projectWebhook api.ProjectWebhook
		var activityList string
		if err := rows.Scan(
			&projectWebhook.ID,
			&projectWebhook.CreatorID,
			&projectWebhook.CreatedTs,
			&projectWebhook.UpdaterID,
			&projectWebhook.UpdatedTs,
			&projectWebhook.ProjectID,
			&projectWebhook.Type,
			&projectWebhook.Name,
			&projectWebhook.URL,
			&activityList,
		); err != nil {
			return nil, FormatError(err)
		}
		projectWebhook.ActivityList = strings.Split(activityList, ",")

		if v := find.ActivityType; v != nil {
			for _, activity := range projectWebhook.ActivityList {
				if api.ActivityType(activity) == *v {
					list = append(list, &projectWebhook)
					break
				}
			}
		} else {
			list = append(list, &projectWebhook)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchProjectWebhook updates a projectWebhook by ID. Returns the new state of the projectWebhook after update.
func patchProjectWebhook(ctx context.Context, tx *Tx, patch *api.ProjectWebhookPatch) (*api.ProjectWebhook, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.URL; v != nil {
		set, args = append(set, "url = ?"), append(args, *v)
	}
	if v := patch.ActivityList; v != nil {
		set, args = append(set, "activity_list = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE project_webhook
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectWebhook api.ProjectWebhook
		var activityList string
		if err := row.Scan(
			&projectWebhook.ID,
			&projectWebhook.CreatorID,
			&projectWebhook.CreatedTs,
			&projectWebhook.UpdaterID,
			&projectWebhook.UpdatedTs,
			&projectWebhook.ProjectID,
			&projectWebhook.Type,
			&projectWebhook.Name,
			&projectWebhook.URL,
			&activityList,
		); err != nil {
			return nil, FormatError(err)
		}
		projectWebhook.ActivityList = strings.Split(activityList, ",")

		return &projectWebhook, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project hook ID not found: %d", patch.ID)}
}

// deleteProjectWebhook permanently deletes a projectWebhook by ID.
func deleteProjectWebhook(ctx context.Context, tx *Tx, delete *api.ProjectWebhookDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_webhook WHERE id = ?`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
