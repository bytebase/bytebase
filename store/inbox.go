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
	_ api.InboxService = (*InboxService)(nil)
)

// InboxService represents a service for managing inbox.
type InboxService struct {
	l  *zap.Logger
	db *DB

	activityService api.ActivityService
}

// NewInboxService returns a new instance of InboxService.
func NewInboxService(logger *zap.Logger, db *DB, activityService api.ActivityService) *InboxService {
	return &InboxService{l: logger, db: db, activityService: activityService}
}

// CreateInbox creates a new inbox.
func (s *InboxService) CreateInbox(ctx context.Context, create *api.InboxCreate) (*api.Inbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	inbox, err := s.createInbox(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return inbox, nil
}

// FindInboxList retrieves a list of inboxs based on find.
func (s *InboxService) FindInboxList(ctx context.Context, find *api.InboxFind) ([]*api.Inbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInboxList(ctx, tx, find)
	if err != nil {
		return []*api.Inbox{}, err
	}

	return list, nil
}

// FindInbox retrieves a single inbox based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *InboxService) FindInbox(ctx context.Context, find *api.InboxFind) (*api.Inbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findInboxList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("inbox not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d inboxes with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchInbox updates an existing inbox by ID.
// Returns ENOTFOUND if inbox does not exist.
func (s *InboxService) PatchInbox(ctx context.Context, patch *api.InboxPatch) (*api.Inbox, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	inbox, err := s.patchInbox(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return inbox, nil
}

// createInbox creates a new inbox.
func (s *InboxService) createInbox(ctx context.Context, tx *Tx, create *api.InboxCreate) (*api.Inbox, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO inbox (
			receiver_id,
			activity_id,
			`+"`status`"+`
		)
		VALUES (?, ?, 'UNREAD')
		RETURNING id, receiver_id, activity_id, `+"`status`"+`
	`,
		create.ReceiverId,
		create.ActivityId,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var inbox api.Inbox
	var activityId int
	if err := row.Scan(
		&inbox.ID,
		&inbox.ReceiverId,
		&activityId,
		&inbox.Status,
	); err != nil {
		return nil, FormatError(err)
	}

	activityFind := &api.ActivityFind{
		ID: &activityId,
	}
	inbox.Activity, err = s.activityService.FindActivity(context.Background(), activityFind)
	if err != nil {
		return nil, FormatError(err)
	}

	return &inbox, nil
}

func findInboxList(ctx context.Context, tx *Tx, find *api.InboxFind) (_ []*api.Inbox, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where = append(where, "inbox.activity_id = activity.id")
	if v := find.ID; v != nil {
		where, args = append(where, "inbox.id = ?"), append(args, *v)
	}
	if v := find.ReceiverId; v != nil {
		where, args = append(where, "receiver_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    inbox.id,
		    receiver_id,
			`+"`status`,"+`
			activity.id,
			activity.creator_id,
		    activity.created_ts,
		    activity.updater_id,
		    activity.updated_ts,
			activity.container_id,
		    activity.type,
		    activity.comment,
			activity.payload
		FROM inbox, activity
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Inbox, 0)
	for rows.Next() {
		var inbox api.Inbox
		inbox.Activity = &api.Activity{}
		if err := rows.Scan(
			&inbox.ID,
			&inbox.ReceiverId,
			&inbox.Status,
			&inbox.Activity.ID,
			&inbox.Activity.CreatorId,
			&inbox.Activity.CreatedTs,
			&inbox.Activity.UpdaterId,
			&inbox.Activity.UpdatedTs,
			&inbox.Activity.ContainerId,
			&inbox.Activity.Type,
			&inbox.Activity.Comment,
			&inbox.Activity.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &inbox)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchInbox updates a inbox by ID. Returns the new state of the inbox after update.
func (s *InboxService) patchInbox(ctx context.Context, tx *Tx, patch *api.InboxPatch) (*api.Inbox, error) {
	// Build UPDATE clause.
	set, args := []string{"`status` = ?"}, []interface{}{patch.Status}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE inbox
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, receiver_id, activity_id, `+"`status`"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var inbox api.Inbox
		var activityId int
		if err := row.Scan(
			&inbox.ID,
			&inbox.ReceiverId,
			&activityId,
			&inbox.Status,
		); err != nil {
			return nil, FormatError(err)
		}

		activityFind := &api.ActivityFind{
			ID: &activityId,
		}
		inbox.Activity, err = s.activityService.FindActivity(context.Background(), activityFind)
		if err != nil {
			return nil, FormatError(err)
		}

		return &inbox, nil
	}

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("inbox ID not found: %d", patch.ID)}
}
