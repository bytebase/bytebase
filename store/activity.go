package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

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

// CreateActivity creates an instance of Activity.
func (s *Store) CreateActivity(ctx context.Context, create *api.ActivityCreate) (*api.Activity, error) {
	activityRaw, err := s.createActivityRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Activity with ActivityCreate[%+v]", create)
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Activity with activityRaw[%+v]", activityRaw)
	}
	return activity, nil
}

// BatchCreateActivity creates activities in batch.
func (s *Store) BatchCreateActivity(ctx context.Context, creates []*api.ActivityCreate) ([]*api.Activity, error) {
	if len(creates) == 0 {
		return nil, nil
	}
	activityRawList, err := s.batchCreateActivityRaw(ctx, creates)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create TaskCheckRun with TaskCheckRunCreates[%+v]", creates)
	}
	var activityList []*api.Activity
	for _, activityRaw := range activityRawList {
		activity, err := s.composeActivity(ctx, activityRaw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose activity with activityRaw %+v", activityRaw)
		}
		activityList = append(activityList, activity)
	}
	return activityList, nil
}

// GetActivityByID gets an instance of Activity.
func (s *Store) GetActivityByID(ctx context.Context, id int) (*api.Activity, error) {
	activityRaw, err := s.getActivityRawByID(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Activity with ID %d", id)
	}
	if activityRaw == nil {
		return nil, nil
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Activity with activityRaw[%+v]", activityRaw)
	}
	return activity, nil
}

// FindActivity finds a list of Activity instances.
func (s *Store) FindActivity(ctx context.Context, find *api.ActivityFind) ([]*api.Activity, error) {
	activityRawList, err := s.findActivityRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Activity list with ActivityFind[%+v]", find)
	}
	var activityList []*api.Activity
	for _, raw := range activityRawList {
		activity, err := s.composeActivity(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Activity with activityRaw[%+v]", raw)
		}
		activityList = append(activityList, activity)
	}
	return activityList, nil
}

// PatchActivity patches an instance of Activity.
func (s *Store) PatchActivity(ctx context.Context, patch *api.ActivityPatch) (*api.Activity, error) {
	activityRaw, err := s.patchActivityRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Activity with ActivityPatch[%+v]", patch)
	}
	activity, err := s.composeActivity(ctx, activityRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Activity with activityRaw[%+v]", activityRaw)
	}
	return activity, nil
}

//
// private function
//

func (s *Store) batchCreateActivityRaw(ctx context.Context, creates []*api.ActivityCreate) ([]*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	activityRawList, err := createActivityImpl(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activityRawList, nil
}

// createActivityRaw creates a new activity.
func (s *Store) createActivityRaw(ctx context.Context, create *api.ActivityCreate) (*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	activityRawList, err := createActivityImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if len(activityRawList) != 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d activities, expect 1", len(activityRawList))}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activityRawList[0], nil
}

// findActivityRaw retrieves a list of activities based on the find condition.
func (s *Store) findActivityRaw(ctx context.Context, find *api.ActivityFind) ([]*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findActivityImpl(ctx, tx, find)
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
	defer tx.Rollback()

	list, err := findActivityImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d activities with filter %+v, expect 1. ", len(list), find)}
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
	defer tx.Rollback()

	activity, err := patchActivityImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return activity, nil
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

// createActivityImpl creates activities.
func createActivityImpl(ctx context.Context, tx *Tx, creates ...*api.ActivityCreate) ([]*activityRaw, error) {
	var query strings.Builder
	var values []interface{}
	var queryValues []string

	if _, err := query.WriteString(
		`INSERT INTO activity (
			creator_id,
			updater_id,
			container_id,
			type,
			level,
			comment,
			payload
		) VALUES
    `); err != nil {
		return nil, err
	}
	for i, create := range creates {
		if create.Payload == "" {
			create.Payload = "{}"
		}
		values = append(values,
			create.CreatorID,
			create.CreatorID,
			create.ContainerID,
			create.Type,
			create.Level,
			create.Comment,
			create.Payload,
		)
		const count = 7
		queryValues = append(queryValues, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*count+1, i*count+2, i*count+3, i*count+4, i*count+5, i*count+6, i*count+7))
	}
	if _, err := query.WriteString(strings.Join(queryValues, ",")); err != nil {
		return nil, err
	}
	if _, err := query.WriteString(` RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload`); err != nil {
		return nil, err
	}

	var activityRawList []*activityRaw
	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var activityRaw activityRaw
		if err := rows.Scan(
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
		activityRawList = append(activityRawList, &activityRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return activityRawList, nil
}

func findActivityImpl(ctx context.Context, tx *Tx, find *api.ActivityFind) ([]*activityRaw, error) {
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
	if v := find.Order; v != nil {
		query += fmt.Sprintf(" ORDER BY created_ts %s", *v)
	}
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
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
func patchActivityImpl(ctx context.Context, tx *Tx, patch *api.ActivityPatch) (*activityRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, api.Role(*v))
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	var activityRaw activityRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE activity
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, container_id, type, level, comment, payload
	`, len(args)),
		args...,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("activity ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &activityRaw, nil
}

// BackfillSQLEditorActivity backfills SQL editor activities.
// TODO(d): remove this after the backfill.
func (s *Store) BackfillSQLEditorActivity(ctx context.Context) error {
	typePrefix := string(api.ActivitySQLEditorQuery)
	activityList, err := s.FindActivity(ctx, &api.ActivityFind{TypePrefix: &typePrefix})
	if err != nil {
		return err
	}
	for _, activity := range activityList {
		queryPayload := &api.ActivitySQLEditorQueryPayload{}
		if err := json.Unmarshal([]byte(activity.Payload), queryPayload); err != nil {
			return err
		}
		if queryPayload.InstanceID > 0 {
			continue
		}
		// Backfill instance ID.
		queryPayload.InstanceID = activity.ContainerID
		// Backfill database ID.
		if queryPayload.DatabaseName != "" {
			database, err := s.GetDatabase(ctx, &api.DatabaseFind{InstanceID: &queryPayload.InstanceID, Name: &queryPayload.DatabaseName})
			if err != nil {
				return err
			}
			if database != nil {
				queryPayload.DatabaseID = database.ID
			}
		}

		activityPayloadBytes, err := json.Marshal(queryPayload)
		if err != nil {
			return err
		}
		activityPayload := string(activityPayloadBytes)
		if _, err := s.patchActivityRaw(ctx, &api.ActivityPatch{
			ID:        activity.ID,
			UpdaterID: activity.UpdaterID,
			Payload:   &activityPayload,
		}); err != nil {
			return err
		}
	}
	return nil
}
