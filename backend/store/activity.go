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
	case api.ActivityIssueApprovalNotify:
		var originalPayload api.ActivityIssueApprovalNotifyPayload
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
	case api.ActivityIssueApprovalNotify:
		originalPayload := &api.ActivityIssueApprovalNotifyPayload{
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
	case "REJECTED":
		return storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_REJECTED
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
	case storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_REJECTED:
		return "REJECTED"
	default:
		return ""
	}
}

// ActivityMessage is the API message for activity.
type ActivityMessage struct {
	UID       int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	CreatorUID int
	UpdaterUID int
	// The object where this activity belongs
	// e.g if Type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
	ContainerUID int

	// Domain specific fields
	Type    api.ActivityType
	Level   api.ActivityLevel
	Comment string
	Payload string
}

// FindActivityMessage is the API message for listing activities.
type FindActivityMessage struct {
	UID             *int
	CreatorUID      *int
	LevelList       []api.ActivityLevel
	TypeList        []api.ActivityType
	ContainerUID    *int
	CreatedTsAfter  *int64
	CreatedTsBefore *int64
	Limit           *int
	Offset          *int
	// If specified, sorts the returned list by id in <<ORDER>>
	// Different use cases want different orders.
	// e.g. Issue activity list wants ASC, while view recent activity list wants DESC.
	Order *api.SortOrder
}

// UpdateActivityMessage updates the activity.
type UpdateActivityMessage struct {
	UID        int
	CreatorUID *int
	UpdaterUID int
	Comment    *string
	Level      *api.ActivityLevel
	Payload    *string
}

// CreateActivityV2 creates an instance of Activity.
func (s *Store) CreateActivityV2(ctx context.Context, create *ActivityMessage) (*ActivityMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	activityList, err := createActivityImplV2(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if len(activityList) != 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d activities, expect 1", len(activityList))}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return activityList[0], nil
}

// BatchCreateActivityV2 creates activities in batch.
func (s *Store) BatchCreateActivityV2(ctx context.Context, creates []*ActivityMessage) ([]*ActivityMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	activityList, err := createActivityImplV2(ctx, tx, creates...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return activityList, nil
}

// UpdateActivityV2 updates the activity.
// Returns ENOTFOUND if activity does not exist.
func (s *Store) UpdateActivityV2(ctx context.Context, update *UpdateActivityMessage) (*ActivityMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	set, args := []string{"updater_id = $1"}, []any{update.UpdaterUID}
	if v := update.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("comment = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if v := update.Level; v != nil {
		set, args = append(set, fmt.Sprintf("level = $%d", len(args)+1)), append(args, *v)
	}

	where, args := []string{fmt.Sprintf("id = $%d", len(args)+1)}, append(args, update.UID)
	if v := update.CreatorUID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}

	query := fmt.Sprintf(`
		UPDATE activity
		SET %s
		WHERE %s
		RETURNING
			id,
			creator_id,
			updater_id,
			created_ts,
			updated_ts,
			container_id,
			type,
			level,
			comment,
			payload
	`, strings.Join(set, ", "), strings.Join(where, " AND "))

	var activity ActivityMessage
	var protoPayload string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&activity.UID,
		&activity.CreatorUID,
		&activity.UpdaterUID,
		&activity.CreatedTs,
		&activity.UpdatedTs,
		&activity.ContainerUID,
		&activity.Type,
		&activity.Level,
		&activity.Comment,
		&protoPayload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("cannot found activity with id: %d", update.UID)}
		}
		return nil, err
	}
	if activity.Payload, err = convertProtoPayloadToAPIPayload(activity.Type, protoPayload); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &activity, nil
}

// GetActivityV2 gets the activity.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) GetActivityV2(ctx context.Context, uid int) (*ActivityMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	activity, err := getActivityImplV2(ctx, tx, uid)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return activity, nil
}

// ListActivityV2 lists the activity.
func (s *Store) ListActivityV2(ctx context.Context, find *FindActivityMessage) ([]*ActivityMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := listActivityImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func getActivityImplV2(ctx context.Context, tx *Tx, uid int) (*ActivityMessage, error) {
	list, err := listActivityImplV2(ctx, tx, &FindActivityMessage{
		UID: &uid,
	})
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d activities with id %+v, expect 1. ", len(list), uid)}
	}
	return list[0], nil
}

func listActivityImplV2(ctx context.Context, tx *Tx, find *FindActivityMessage) ([]*ActivityMessage, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ContainerUID; v != nil {
		where, args = append(where, fmt.Sprintf("container_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorUID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.LevelList; len(v) > 0 {
		var queryValues []string
		for _, level := range v {
			queryValues, args = append(queryValues, fmt.Sprintf("level = $%d", len(args)+1)), append(args, level)
		}
		where = append(where, fmt.Sprintf("(%s)", strings.Join(queryValues, " OR ")))
	}
	if v := find.TypeList; len(v) > 0 {
		var queryValues []string
		for _, t := range v {
			queryValues, args = append(queryValues, fmt.Sprintf("type = $%d", len(args)+1)), append(args, t)
		}
		where = append(where, fmt.Sprintf("(%s)", strings.Join(queryValues, " OR ")))
	}
	if v := find.CreatedTsAfter; v != nil {
		where, args = append(where, fmt.Sprintf("created_ts >= $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatedTsBefore; v != nil {
		where, args = append(where, fmt.Sprintf("created_ts <= $%d", len(args)+1)), append(args, *v)
	}

	query := `
		SELECT
			id,
			creator_id,
			updater_id,
			created_ts,
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
	} else {
		query += " ORDER BY id ASC"
	}
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into activityRawList.
	var activityList []*ActivityMessage
	for rows.Next() {
		var activity ActivityMessage
		var protoPayload string
		if err := rows.Scan(
			&activity.UID,
			&activity.CreatorUID,
			&activity.UpdaterUID,
			&activity.CreatedTs,
			&activity.UpdatedTs,
			&activity.ContainerUID,
			&activity.Type,
			&activity.Level,
			&activity.Comment,
			&protoPayload,
		); err != nil {
			return nil, err
		}
		if protoPayload == "" {
			protoPayload = "{}"
		}
		if activity.Payload, err = convertProtoPayloadToAPIPayload(activity.Type, protoPayload); err != nil {
			return nil, err
		}
		activityList = append(activityList, &activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return activityList, nil
}

func createActivityImplV2(ctx context.Context, tx *Tx, creates ...*ActivityMessage) ([]*ActivityMessage, error) {
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
			create.CreatorUID,
			create.CreatorUID,
			create.ContainerUID,
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

	var activityList []*ActivityMessage
	rows, err := tx.QueryContext(ctx, query.String(), values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var activity ActivityMessage
		var protoPayload string
		if err := rows.Scan(
			&activity.UID,
			&activity.CreatorUID,
			&activity.CreatedTs,
			&activity.UpdaterUID,
			&activity.UpdatedTs,
			&activity.ContainerUID,
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
		activityList = append(activityList, &activity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activityList, nil
}
