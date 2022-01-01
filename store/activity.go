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
	_ api.ActivityService = (*ActivityService)(nil)
)

// ActivityService represents a service for managing activity.
type ActivityService struct {
	l  *zap.Logger
	db *DB
}

// NewActivityService returns a new instance of ActivityService.
func NewActivityService(logger *zap.Logger, db *DB) *ActivityService {
	return &ActivityService{l: logger, db: db}
}

// CreateActivity creates a new activity.
func (s *ActivityService) CreateActivity(ctx context.Context, create *api.ActivityCreate) (*api.Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	activity, err := createActivity(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activity, nil
}

// FindActivityList retrieves a list of activitys based on find.
func (s *ActivityService) FindActivityList(ctx context.Context, find *api.ActivityFind) ([]*api.Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findActivityList(ctx, tx, find)
	if err != nil {
		return []*api.Activity{}, err
	}

	return list, nil
}

// FindActivity retrieves a single activity based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ActivityService) FindActivity(ctx context.Context, find *api.ActivityFind) (*api.Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findActivityList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d activities with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// PatchActivity updates an existing activity by ID.
// Returns ENOTFOUND if activity does not exist.
func (s *ActivityService) PatchActivity(ctx context.Context, patch *api.ActivityPatch) (*api.Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	activity, err := patchActivity(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activity, nil
}

// DeleteActivity deletes an existing activity by ID.
func (s *ActivityService) DeleteActivity(ctx context.Context, delete *api.ActivityDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteActivity(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createActivity creates a new activity.
func createActivity(ctx context.Context, tx *Tx, create *api.ActivityCreate) (*api.Activity, error) {
	// Insert row into activity.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO activity (
			creator_id,
			updater_id,
			container_id,
			`+"`type`,"+`
			`+"`level`,"+`
			comment,
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, `+"`type`, level, comment, payload"+`
	`,
		create.CreatorID,
		create.CreatorID,
		create.ContainerID,
		create.Type,
		create.Level,
		create.Comment,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var activity api.Activity
	if err := row.Scan(
		&activity.ID,
		&activity.CreatorID,
		&activity.CreatedTs,
		&activity.UpdaterID,
		&activity.UpdatedTs,
		&activity.ContainerID,
		&activity.Type,
		&activity.Level,
		&activity.Comment,
		&activity.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &activity, nil
}

func findActivityList(ctx context.Context, tx *Tx, find *api.ActivityFind) (_ []*api.Activity, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.ContainerID; v != nil {
		where, args = append(where, "container_id = ?"), append(args, *v)
	}

	var query = `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
			container_id,
		    ` + "`type`," + `
			` + "`level`," + `
		    comment,
			payload
		FROM activity
		WHERE ` + strings.Join(where, " AND ")
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" ORDER BY updated_ts DESC LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Activity, 0)
	for rows.Next() {
		var activity api.Activity
		if err := rows.Scan(
			&activity.ID,
			&activity.CreatorID,
			&activity.CreatedTs,
			&activity.UpdaterID,
			&activity.UpdatedTs,
			&activity.ContainerID,
			&activity.Type,
			&activity.Level,
			&activity.Comment,
			&activity.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &activity)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchActivity updates a activity by ID. Returns the new state of the activity after update.
func patchActivity(ctx context.Context, tx *Tx, patch *api.ActivityPatch) (*api.Activity, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Comment; v != nil {
		set, args = append(set, "comment = ?"), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE activity
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, `+"`type`, level, comment, payload"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var activity api.Activity
		if err := row.Scan(
			&activity.ID,
			&activity.CreatorID,
			&activity.CreatedTs,
			&activity.UpdaterID,
			&activity.UpdatedTs,
			&activity.ContainerID,
			&activity.Type,
			&activity.Level,
			&activity.Comment,
			&activity.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &activity, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("activity ID not found: %d", patch.ID)}
}

// deleteActivity permanently deletes a activity by ID.
func deleteActivity(ctx context.Context, tx *Tx, delete *api.ActivityDelete) error {
	// Remove row from activity.
	if _, err := tx.ExecContext(ctx, `DELETE FROM activity WHERE id = ?`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
