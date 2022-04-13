package store

import (
	"context"
	"database/sql"
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
func (s *ProjectWebhookService) CreateProjectWebhook(ctx context.Context, create *api.ProjectWebhookCreate) (*api.ProjectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectWebhook, err := createProjectWebhook(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhook, nil
}

// FindProjectWebhookList retrieves a list of projectWebhooks based on find.
func (s *ProjectWebhookService) FindProjectWebhookList(ctx context.Context, find *api.ProjectWebhookFind) ([]*api.ProjectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectWebhookList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindProjectWebhook retrieves a single projectWebhook based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ProjectWebhookService) FindProjectWebhook(ctx context.Context, find *api.ProjectWebhookFind) (*api.ProjectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectWebhookList(ctx, tx.PTx, find)
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
func (s *ProjectWebhookService) PatchProjectWebhook(ctx context.Context, patch *api.ProjectWebhookPatch) (*api.ProjectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectWebhook, err := patchProjectWebhook(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
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
	defer tx.PTx.Rollback()

	if err := deleteProjectWebhook(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createProjectWebhook creates a new projectWebhook.
func createProjectWebhook(ctx context.Context, tx *sql.Tx, create *api.ProjectWebhookCreate) (*api.ProjectWebhookRaw, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Type,
		create.Name,
		create.URL,
		create.ActivityList,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectWebhookRaw api.ProjectWebhookRaw
	if err := row.Scan(
		&projectWebhookRaw.ID,
		&projectWebhookRaw.CreatorID,
		&projectWebhookRaw.CreatedTs,
		&projectWebhookRaw.UpdaterID,
		&projectWebhookRaw.UpdatedTs,
		&projectWebhookRaw.ProjectID,
		&projectWebhookRaw.Type,
		&projectWebhookRaw.Name,
		&projectWebhookRaw.URL,
		&projectWebhookRaw.ActivityList,
	); err != nil {
		return nil, FormatError(err)
	}

	return &projectWebhookRaw, nil
}

func findProjectWebhookList(ctx context.Context, tx *sql.Tx, find *api.ProjectWebhookFind) ([]*api.ProjectWebhookRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
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

	// Iterate over result set and deserialize rows into projectWebhookRawList.
	var projectWebhookRawList []*api.ProjectWebhookRaw
	for rows.Next() {
		var projectWebhookRaw api.ProjectWebhookRaw
		if err := rows.Scan(
			&projectWebhookRaw.ID,
			&projectWebhookRaw.CreatorID,
			&projectWebhookRaw.CreatedTs,
			&projectWebhookRaw.UpdaterID,
			&projectWebhookRaw.UpdatedTs,
			&projectWebhookRaw.ProjectID,
			&projectWebhookRaw.Type,
			&projectWebhookRaw.Name,
			&projectWebhookRaw.URL,
			&projectWebhookRaw.ActivityList,
		); err != nil {
			return nil, FormatError(err)
		}

		if v := find.ActivityType; v != nil {
			for _, activity := range projectWebhookRaw.ActivityList {
				if api.ActivityType(activity) == *v {
					projectWebhookRawList = append(projectWebhookRawList, &projectWebhookRaw)
					break
				}
			}
		} else {
			projectWebhookRawList = append(projectWebhookRawList, &projectWebhookRaw)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhookRawList, nil
}

// patchProjectWebhook updates a projectWebhook by ID. Returns the new state of the projectWebhook after update.
func patchProjectWebhook(ctx context.Context, tx *sql.Tx, patch *api.ProjectWebhookPatch) (*api.ProjectWebhookRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.URL; v != nil {
		set, args = append(set, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ActivityList; v != nil {
		activities := strings.Split(*v, ",")
		set, args = append(set, fmt.Sprintf("activity_list = $%d", len(args)+1)), append(args, activities)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE project_webhook
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectWebhookRaw api.ProjectWebhookRaw
		if err := row.Scan(
			&projectWebhookRaw.ID,
			&projectWebhookRaw.CreatorID,
			&projectWebhookRaw.CreatedTs,
			&projectWebhookRaw.UpdaterID,
			&projectWebhookRaw.UpdatedTs,
			&projectWebhookRaw.ProjectID,
			&projectWebhookRaw.Type,
			&projectWebhookRaw.Name,
			&projectWebhookRaw.URL,
			&projectWebhookRaw.ActivityList,
		); err != nil {
			return nil, FormatError(err)
		}

		return &projectWebhookRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project hook ID not found: %d", patch.ID)}
}

// deleteProjectWebhook permanently deletes a projectWebhook by ID.
func deleteProjectWebhook(ctx context.Context, tx *sql.Tx, delete *api.ProjectWebhookDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_webhook WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
