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
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d inboxes with filter %+v, expect 1", len(list), find)}
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

// FindInboxSummary returns the inbox summary for a particular principal
func (s *InboxService) FindInboxSummary(ctx context.Context, principalID int) (*api.InboxSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	row, err := tx.QueryContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM inbox WHERE receiver_id = ? AND status = 'UNREAD')
	`,
		principalID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var inboxSummary api.InboxSummary
	if err := row.Scan(
		&inboxSummary.HasUnread,
	); err != nil {
		return nil, FormatError(err)
	}

	if inboxSummary.HasUnread {
		row2, err := tx.QueryContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM inbox, activity WHERE inbox.receiver_id = ? AND inbox.status = 'UNREAD' AND inbox.activity_id = activity.id AND activity.level = 'ERROR')
	`,
			principalID,
		)

		if err != nil {
			return nil, FormatError(err)
		}
		defer row2.Close()

		row2.Next()
		if err := row2.Scan(
			&inboxSummary.HasUnreadError,
		); err != nil {
			return nil, FormatError(err)
		}
	} else {
		inboxSummary.HasUnreadError = false
	}

	return &inboxSummary, nil
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
		create.ReceiverID,
		create.ActivityID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var inbox api.Inbox
	var activityID int
	if err := row.Scan(
		&inbox.ID,
		&inbox.ReceiverID,
		&activityID,
		&inbox.Status,
	); err != nil {
		return nil, FormatError(err)
	}

	activityFind := &api.ActivityFind{
		ID: &activityID,
	}
	inbox.Activity, err = s.activityService.FindActivity(ctx, activityFind)
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
	if v := find.ReceiverID; v != nil {
		where, args = append(where, "receiver_id = ?"), append(args, *v)
	}
	if v := find.ReadCreatedAfterTs; v != nil {
		where, args = append(where, "(status != 'READ' OR created_ts >= ?)"), append(args, *v)
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
			activity.level,
		    activity.comment,
			activity.payload
		FROM inbox, activity
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY activity.created_ts DESC`,
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
			&inbox.ReceiverID,
			&inbox.Status,
			&inbox.Activity.ID,
			&inbox.Activity.CreatorID,
			&inbox.Activity.CreatedTs,
			&inbox.Activity.UpdaterID,
			&inbox.Activity.UpdatedTs,
			&inbox.Activity.ContainerID,
			&inbox.Activity.Type,
			&inbox.Activity.Level,
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
		var activityID int
		if err := row.Scan(
			&inbox.ID,
			&inbox.ReceiverID,
			&activityID,
			&inbox.Status,
		); err != nil {
			return nil, FormatError(err)
		}

		activityFind := &api.ActivityFind{
			ID: &activityID,
		}
		inbox.Activity, err = s.activityService.FindActivity(ctx, activityFind)
		if err != nil {
			return nil, FormatError(err)
		}

		return &inbox, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("inbox ID not found: %d", patch.ID)}
}
