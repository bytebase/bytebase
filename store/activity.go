package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// activityRaw is the store model for an Activity.
// Fields have exactly the same meanings as Activity.
type activityRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	// The object where this activity belongs
	// e.g if Type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
	ContainerID int

	// Domain specific fields
	Type    api.ActivityType
	Level   api.ActivityLevel
	Comment string
	Payload string
}

// toActivity creates an instance of Activity based on the ActivityRaw.
// This is intended to be called when we need to compose an Activity relationship.
func (raw *activityRaw) toActivity() *api.Activity {
	return &api.Activity{
		ID: raw.ID,

		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		ContainerID: raw.ContainerID,

		Type:    raw.Type,
		Level:   raw.Level,
		Comment: raw.Comment,
		Payload: raw.Payload,
	}
}

// CreateActivity creates an instance of Activity
func (s *Store) CreateActivity(ctx context.Context, create *api.ActivityCreate) (*api.Activity, error) {
	activityRaw, err := s.createActivityRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Activity with ActivityCreate[%+v], error[%w]", create, err)
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Activity with activityRaw[%+v], error[%w]", activityRaw, err)
	}
	return activity, nil
}

// GetActivityByID gets an instance of Activity
func (s *Store) GetActivityByID(ctx context.Context, id int) (*api.Activity, error) {
	activityRaw, err := s.getActivityRawByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Failed to get Activity with ID[%d], error[%w]", id, err)
	}
	if activityRaw == nil {
		return nil, nil
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Activity with activityRaw[%+v], error[%w]", activityRaw, err)
	}
	return activity, nil
}

// FindActivity finds a list of Activity instances
func (s *Store) FindActivity(ctx context.Context, find *api.ActivityFind) ([]*api.Activity, error) {
	activityRawList, err := s.findActivityRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Activity list, error[%w]", err)
	}
	var activityList []*api.Activity
	for _, raw := range activityRawList {
		activity, err := s.composeActivity(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Activity with activityRaw[%+v], error[%w]", raw, err)
		}
		activityList = append(activityList, activity)
	}
	return activityList, nil
}

// PatchActivity patches an instance of Activity
func (s *Store) PatchActivity(ctx context.Context, patch *api.ActivityPatch) (*api.Activity, error) {
	activityRaw, err := s.patchActivityRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Activity with ActivityPatch[%+v], error[%w]", patch, err)
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Activity with activityRaw[%+v], error[%w]", activityRaw, err)
	}
	return activity, nil
}

// createActivityRaw creates a new activity.
func (s *Store) createActivityRaw(ctx context.Context, create *api.ActivityCreate) (*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	activity, err := createActivityImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activity, nil
}

// findActivityRaw retrieves a list of activities based on the find condition.
func (s *Store) findActivityRaw(ctx context.Context, find *api.ActivityFind) ([]*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findActivityImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getActivityRawByID retrieves a single activity based on ID.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getActivityRawByID(ctx context.Context, id int) (*activityRaw, error) {
	find := &api.ActivityFind{ID: &id}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findActivityImpl(ctx, tx.PTx, find)
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

// patchActivityRaw updates an existing activity by ID.
// Returns ENOTFOUND if activity does not exist.
func (s *Store) patchActivityRaw(ctx context.Context, patch *api.ActivityPatch) (*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	activity, err := patchActivityImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activity, nil
}

// DeleteActivity deletes an existing activity by ID.
func (s *Store) DeleteActivity(ctx context.Context, delete *api.ActivityDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteActivityImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

func (s *Store) composeActivity(ctx context.Context, raw *activityRaw) (*api.Activity, error) {
	activity := raw.toActivity()

	creator, err := s.GetPrincipalByID(ctx, activity.CreatorID)
	if err != nil {
		return nil, err
	}
	activity.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, activity.UpdaterID)
	if err != nil {
		return nil, err
	}
	activity.Updater = updater

	return activity, nil
}

// createActivityImpl creates a new activity.
func createActivityImpl(ctx context.Context, tx *sql.Tx, create *api.ActivityCreate) (*activityRaw, error) {
	// Insert row into activity.
	if create.Payload == "" {
		create.Payload = "{}"
	}
	row, err := tx.QueryContext(ctx, `
		INSERT INTO activity (
			creator_id,
			updater_id,
			container_id,
			type,
			level,
			comment,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload
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
	var activityRaw activityRaw
	if err := row.Scan(
		&activityRaw.ID,
		&activityRaw.CreatorID,
		&activityRaw.CreatedTs,
		&activityRaw.UpdaterID,
		&activityRaw.UpdatedTs,
		&activityRaw.ContainerID,
		&activityRaw.Type,
		&activityRaw.Level,
		&activityRaw.Comment,
		&activityRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &activityRaw, nil
}

func findActivityImpl(ctx context.Context, tx *sql.Tx, find *api.ActivityFind) ([]*activityRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ContainerID; v != nil {
		where, args = append(where, fmt.Sprintf("container_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.TypePrefix; v != nil {
		where, args = append(where, fmt.Sprintf("type LIKE $%d", len(args)+1)), append(args, fmt.Sprintf("%s%%", *v))
	}
	if v := find.Level; v != nil {
		where, args = append(where, fmt.Sprintf("level = $%d", len(args)+1)), append(args, *v)
	}

	var query = `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			container_id,
			type,
			level,
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

	// Iterate over result set and deserialize rows into activityRawList.
	var activityRawList []*activityRaw
	for rows.Next() {
		var activity activityRaw
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

		activityRawList = append(activityRawList, &activity)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return activityRawList, nil
}

// patchActivityImpl updates a activity by ID. Returns the new state of the activity after update.
func patchActivityImpl(ctx context.Context, tx *sql.Tx, patch *api.ActivityPatch) (*activityRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, api.Role(*v))
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE activity
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var activityRaw activityRaw
		if err := row.Scan(
			&activityRaw.ID,
			&activityRaw.CreatorID,
			&activityRaw.CreatedTs,
			&activityRaw.UpdaterID,
			&activityRaw.UpdatedTs,
			&activityRaw.ContainerID,
			&activityRaw.Type,
			&activityRaw.Level,
			&activityRaw.Comment,
			&activityRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		return &activityRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("activity ID not found: %d", patch.ID)}
}

// deleteActivityImpl permanently deletes a activity by ID.
func deleteActivityImpl(ctx context.Context, tx *sql.Tx, delete *api.ActivityDelete) error {
	// Remove row from activity.
	if _, err := tx.ExecContext(ctx, `DELETE FROM activity WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
