package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.ProjectHookService = (*ProjectHookService)(nil)
)

// ProjectHookService represents a service for managing projectHook.
type ProjectHookService struct {
	l  *zap.Logger
	db *DB
}

// NewProjectHookService returns a new instance of ProjectHookService.
func NewProjectHookService(logger *zap.Logger, db *DB) *ProjectHookService {
	return &ProjectHookService{l: logger, db: db}
}

// CreateProjectHook creates a new projectHook.
func (s *ProjectHookService) CreateProjectHook(ctx context.Context, create *api.ProjectHookCreate) (*api.ProjectHook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectHook, err := createProjectHook(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectHook, nil
}

// FindProjectHookList retrieves a list of projectHooks based on find.
func (s *ProjectHookService) FindProjectHookList(ctx context.Context, find *api.ProjectHookFind) ([]*api.ProjectHook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectHookList(ctx, tx, find)
	if err != nil {
		return []*api.ProjectHook{}, err
	}

	return list, nil
}

// FindProjectHook retrieves a single projectHook based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ProjectHookService) FindProjectHook(ctx context.Context, find *api.ProjectHookFind) (*api.ProjectHook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findProjectHookList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project hook not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d project hooks with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchProjectHook updates an existing projectHook by ID.
// Returns ENOTFOUND if projectHook does not exist.
func (s *ProjectHookService) PatchProjectHook(ctx context.Context, patch *api.ProjectHookPatch) (*api.ProjectHook, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	projectHook, err := patchProjectHook(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectHook, nil
}

// DeleteProjectHook deletes an existing projectHook by ID.
// Returns ENOTFOUND if projectHook does not exist.
func (s *ProjectHookService) DeleteProjectHook(ctx context.Context, delete *api.ProjectHookDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteProjectHook(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createProjectHook creates a new projectHook.
func createProjectHook(ctx context.Context, tx *Tx, create *api.ProjectHookCreate) (*api.ProjectHook, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_hook (
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
		create.CreatorId,
		create.CreatorId,
		create.ProjectId,
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
	var projectHook api.ProjectHook
	var activityList string
	if err := row.Scan(
		&projectHook.ID,
		&projectHook.CreatorId,
		&projectHook.CreatedTs,
		&projectHook.UpdaterId,
		&projectHook.UpdatedTs,
		&projectHook.ProjectId,
		&projectHook.Type,
		&projectHook.Name,
		&projectHook.URL,
		&activityList,
	); err != nil {
		return nil, FormatError(err)
	}
	projectHook.ActivityList = strings.Split(activityList, ",")

	return &projectHook, nil
}

func findProjectHookList(ctx context.Context, tx *Tx, find *api.ProjectHookFind) (_ []*api.ProjectHook, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.ProjectId; v != nil {
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
		FROM project_hook
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.ProjectHook, 0)
	for rows.Next() {
		var projectHook api.ProjectHook
		var activityList string
		if err := rows.Scan(
			&projectHook.ID,
			&projectHook.CreatorId,
			&projectHook.CreatedTs,
			&projectHook.UpdaterId,
			&projectHook.UpdatedTs,
			&projectHook.ProjectId,
			&projectHook.Type,
			&projectHook.Name,
			&projectHook.URL,
			&activityList,
		); err != nil {
			return nil, FormatError(err)
		}
		projectHook.ActivityList = strings.Split(activityList, ",")

		if v := find.ActivityType; v != nil {
			for _, activity := range projectHook.ActivityList {
				if api.ActivityType(activity) == *v {
					list = append(list, &projectHook)
					break
				}
			}
		} else {
			list = append(list, &projectHook)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchProjectHook updates a projectHook by ID. Returns the new state of the projectHook after update.
func patchProjectHook(ctx context.Context, tx *Tx, patch *api.ProjectHookPatch) (*api.ProjectHook, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
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
		UPDATE project_hook
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
		var projectHook api.ProjectHook
		var activityList string
		if err := row.Scan(
			&projectHook.ID,
			&projectHook.CreatorId,
			&projectHook.CreatedTs,
			&projectHook.UpdaterId,
			&projectHook.UpdatedTs,
			&projectHook.ProjectId,
			&projectHook.Type,
			&projectHook.Name,
			&projectHook.URL,
			&activityList,
		); err != nil {
			return nil, FormatError(err)
		}
		projectHook.ActivityList = strings.Split(activityList, ",")

		return &projectHook, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project hook ID not found: %d", patch.ID)}
}

// deleteProjectHook permanently deletes a projectHook by ID.
func deleteProjectHook(ctx context.Context, tx *Tx, delete *api.ProjectHookDelete) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx, `DELETE FROM project_hook WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("project hook ID not found: %d", delete.ID)}
	}

	return nil
}
