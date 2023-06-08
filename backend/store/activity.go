package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
		return nil, err
	}
	defer tx.Rollback()

	activityRawList, err := createActivityImpl(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return activityRawList, nil
}

// createActivityRaw creates a new activity.
func (s *Store) createActivityRaw(ctx context.Context, create *api.ActivityCreate) (*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return activityRawList[0], nil
}

// findActivityRaw retrieves a list of activities based on the find condition.
func (s *Store) findActivityRaw(ctx context.Context, find *api.ActivityFind) ([]*activityRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
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
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer tx.Rollback()

	activity, err := patchActivityImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
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
	var values []any
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
		payload, err := convertAPIPayloadToProtoPayload(create.Type, create.Payload)
		if err != nil {
			return nil, err
		}
		values = append(values,
			create.CreatorID,
			create.CreatorID,
			create.ContainerID,
			create.Type,
			create.Level,
			create.Comment,
			payload,
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
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var activityRaw activityRaw
		var protoPayload string
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
			&protoPayload,
		); err != nil {
			return nil, err
		}
		if activityRaw.Payload, err = convertProtoPayloadToAPIPayload(activityRaw.Type, protoPayload); err != nil {
			return nil, err
		}
		activityRawList = append(activityRawList, &activityRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activityRawList, nil
}

func findActivityImpl(ctx context.Context, tx *Tx, find *api.ActivityFind) ([]*activityRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ContainerID; v != nil {
		where, args = append(where, fmt.Sprintf("container_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.TypePrefixList; len(v) > 0 {
		var queryValues []string
		// Iterate over the typePrefix list and join each one with an OR condition.
		for _, str := range v {
			queryValues, args = append(queryValues, fmt.Sprintf("type LIKE $%d", len(args)+1)), append(args, fmt.Sprintf("%s%%", str))
		}
		where = append(where, fmt.Sprintf("(%s)", strings.Join(queryValues, " OR ")))
	}
	if v := find.LevelList; len(v) > 0 {
		var queryValues []string
		for _, level := range v {
			queryValues, args = append(queryValues, fmt.Sprintf("level = $%d", len(args)+1)), append(args, level)
		}
		where = append(where, fmt.Sprintf("(%s)", strings.Join(queryValues, " OR ")))
	}
	if v := find.SinceID; v != nil {
		where, args = append(where, fmt.Sprintf("id <= $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatedTsAfter; v != nil {
		where, args = append(where, fmt.Sprintf("created_ts >= $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatedTsBefore; v != nil {
		where, args = append(where, fmt.Sprintf("created_ts <= $%d", len(args)+1)), append(args, *v)
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
		query += fmt.Sprintf(" ORDER BY id %s", *v)
	}
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into activityRawList.
	var activityRawList []*activityRaw
	for rows.Next() {
		var activity activityRaw
		var protoPayload string
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
			&protoPayload,
		); err != nil {
			return nil, err
		}
		if activity.Payload, err = convertProtoPayloadToAPIPayload(activity.Type, protoPayload); err != nil {
			return nil, err
		}
		activityRawList = append(activityRawList, &activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return activityRawList, nil
}

// patchActivityImpl updates a activity by ID. Returns the new state of the activity after update.
func patchActivityImpl(ctx context.Context, tx *Tx, patch *api.ActivityPatch) (*activityRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, api.Role(*v))
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Level; v != nil {
		set, args = append(set, fmt.Sprintf("level = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	var activityRaw activityRaw
	var protoPayload string
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
		&protoPayload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("activity ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	var err error
	if activityRaw.Payload, err = convertProtoPayloadToAPIPayload(activityRaw.Type, protoPayload); err != nil {
		return nil, err
	}

	return &activityRaw, nil
}

func convertAPIPayloadToProtoPayload(activityType api.ActivityType, payload string) (string, error) {
	// TODO(zp): remove here when we migrate all payloads.
	switch activityType {
	case api.ActivityIssueCreate:
		// Unmarshal the payload to get the issue name.
		var originalPayload api.ActivityIssueCreatePayload
		if err := json.Unmarshal([]byte(payload), &originalPayload); err != nil {
			return "", err
		}
		protoPayload := &storepb.ActivityIssueCreatePayload{
			IssueName: originalPayload.IssueName,
		}
		newPayload, err := protojson.Marshal(protoPayload)
		if err != nil {
			return "", err
		}
		return string(newPayload), nil
	case api.ActivityIssueCommentCreate:
		var originalPayload api.ActivityIssueCommentCreatePayload
		if err := json.Unmarshal([]byte(payload), &originalPayload); err != nil {
			return "", err
		}
		protoPayload := &storepb.ActivityIssueCommentCreatePayload{
			IssueName: originalPayload.IssueName,
		}
		if originalPayload.ExternalApprovalEvent != nil {
			protoPayload.Event = &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_{
				ExternalApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent{
					Type:      convertAPIExternalApprovalEventTypeToStorePBType(originalPayload.ExternalApprovalEvent.Type),
					Action:    convertAPIExternalApprovalEventActionToStorePBAction(originalPayload.ExternalApprovalEvent.Action),
					StageName: originalPayload.ExternalApprovalEvent.StageName,
				},
			}
		} else if originalPayload.TaskRollbackBy != nil {
			protoPayload.Event = &storepb.ActivityIssueCommentCreatePayload_TaskRollbackBy_{
				TaskRollbackBy: &storepb.ActivityIssueCommentCreatePayload_TaskRollbackBy{
					IssueId:           int64(originalPayload.TaskRollbackBy.IssueID),
					TaskId:            int64(originalPayload.TaskRollbackBy.TaskID),
					RollbackByIssueId: int64(originalPayload.TaskRollbackBy.RollbackByIssueID),
					RollbackByTaskId:  int64(originalPayload.TaskRollbackBy.RollbackByTaskID),
				},
			}
		} else if originalPayload.ApprovalEvent != nil {
			protoPayload.Event = &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_{
				ApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent{
					Status: convertAPIApprovalEventStatusToStorePBStatus(originalPayload.ApprovalEvent.Status),
				},
			}
		}
		newPayload, err := protojson.Marshal(protoPayload)
		if err != nil {
			return "", err
		}
		return string(newPayload), nil
	case api.ActivityIssueApprovalStepPending:
		var originalPayload api.ActivityIssueApprovalStepPendingPayload
		if err := json.Unmarshal([]byte(payload), &originalPayload); err != nil {
			return "", err
		}
		return originalPayload.ProtoPayload, nil
	default:
		return payload, nil
	}
}

func convertProtoPayloadToAPIPayload(activityType api.ActivityType, payload string) (string, error) {
	// TODO(zp): remove here when we migrate all payloads.
	switch activityType {
	case api.ActivityIssueCreate:
		var protoPayload storepb.ActivityIssueCreatePayload
		if err := protojson.Unmarshal([]byte(payload), &protoPayload); err != nil {
			return "", err
		}
		originalPayload := &api.ActivityIssueCreatePayload{
			IssueName: protoPayload.IssueName,
		}
		newPayload, err := json.Marshal(originalPayload)
		if err != nil {
			return "", err
		}
		return string(newPayload), nil
	case api.ActivityIssueCommentCreate:
		var protoPayload storepb.ActivityIssueCommentCreatePayload
		if err := protojson.Unmarshal([]byte(payload), &protoPayload); err != nil {
			return "", err
		}
		originalPayload := &api.ActivityIssueCommentCreatePayload{
			IssueName: protoPayload.IssueName,
		}
		if protoPayload.Event != nil {
			switch event := protoPayload.Event.(type) {
			case *storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_:
				apiTp, err := convertStorePBTypeToAPIExternalApprovalEventType(event.ExternalApprovalEvent.Type)
				if err != nil {
					return "", err
				}
				apiAction, err := convertStorePBActionToAPIExternalApprovalEventAction(event.ExternalApprovalEvent.Action)
				if err != nil {
					return "", err
				}
				originalPayload.ExternalApprovalEvent = &api.ExternalApprovalEvent{
					Type:      apiTp,
					Action:    apiAction,
					StageName: event.ExternalApprovalEvent.StageName,
				}
			case *storepb.ActivityIssueCommentCreatePayload_TaskRollbackBy_:
				originalPayload.TaskRollbackBy = &api.TaskRollbackBy{
					IssueID:           int(event.TaskRollbackBy.IssueId),
					TaskID:            int(event.TaskRollbackBy.TaskId),
					RollbackByIssueID: int(event.TaskRollbackBy.RollbackByIssueId),
					RollbackByTaskID:  int(event.TaskRollbackBy.RollbackByTaskId),
				}
			case *storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_:
				originalPayload.ApprovalEvent = &api.ApprovalEvent{
					Status: convertStorePBStatusToAPIApprovalEventStatus(event.ApprovalEvent.Status),
				}
			}
		}
		newPayload, err := json.Marshal(originalPayload)
		if err != nil {
			return "", err
		}
		return string(newPayload), nil
	case api.ActivityIssueApprovalStepPending:
		originalPayload := &api.ActivityIssueApprovalStepPendingPayload{
			ProtoPayload: payload,
		}
		newPayload, err := json.Marshal(originalPayload)
		if err != nil {
			return "", err
		}
		return string(newPayload), nil
	default:
		return payload, nil
	}
}

func convertAPIExternalApprovalEventActionToStorePBAction(action api.ExternalApprovalEventActionType) storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action {
	switch action {
	case api.ExternalApprovalEventActionApprove:
		return storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_APPROVE
	case api.ExternalApprovalEventActionReject:
		return storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_REJECT
	default:
		return storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_UNSPECIFIED
	}
}

func convertAPIExternalApprovalEventTypeToStorePBType(eventType api.ExternalApprovalType) storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type {
	switch eventType {
	case api.ExternalApprovalTypeFeishu:
		return storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_TYPE_FEISHU
	default:
		return storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_TYPE_UNSPECIFIED
	}
}

func convertStorePBActionToAPIExternalApprovalEventAction(action storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action) (api.ExternalApprovalEventActionType, error) {
	switch action {
	case storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_APPROVE:
		return api.ExternalApprovalEventActionApprove, nil
	case storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_ACTION_REJECT:
		return api.ExternalApprovalEventActionReject, nil
	default:
		return api.ExternalApprovalEventActionType(""), nil
	}
}

func convertStorePBTypeToAPIExternalApprovalEventType(eventType storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type) (api.ExternalApprovalType, error) {
	switch eventType {
	case storepb.ActivityIssueCommentCreatePayload_ExternalApprovalEvent_TYPE_FEISHU:
		return api.ExternalApprovalTypeFeishu, nil
	default:
		return api.ExternalApprovalType(""), nil
	}
}

func convertAPIApprovalEventStatusToStorePBStatus(status string) storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_Status {
	switch status {
	case "APPROVED":
		return storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_APPROVED
	case "PENDING":
		return storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_PENDING
	default:
		return storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_STATUS_UNSPECIFIED
	}
}

func convertStorePBStatusToAPIApprovalEventStatus(status storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_Status) string {
	switch status {
	case storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_APPROVED:
		return "APPROVED"
	case storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_PENDING:
		return "PENDING"
	default:
		return ""
	}
}
