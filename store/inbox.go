package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// inboxRaw is the store model for an Inbox.
// Fields have exactly the same meanings as Inbox.
type inboxRaw struct {
	ID int

	// Domain specific fields
	ReceiverID  int
	ActivityRaw *activityRaw
	Status      api.InboxStatus
}

// toInbox creates an instance of Inbox based on the inboxRaw.
// This is intended to be called when we need to compose an Inbox relationship.
func (raw *inboxRaw) toInbox() *api.Inbox {
	return &api.Inbox{
		ID: raw.ID,

		ReceiverID: raw.ReceiverID,
		Status:     raw.Status,
	}
}

// CreateInbox creates an instance of Inbox
func (s *Store) CreateInbox(ctx context.Context, create *api.InboxCreate) (*api.Inbox, error) {
	inboxRaw, err := s.createInboxRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Inbox with InboxCreate[%+v], error[%w]", create, err)
	}
	inbox, err := s.composeInbox(ctx, inboxRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Inbox with inboxRaw[%+v], error[%w]", inboxRaw, err)
	}
	return inbox, nil
}

// GetInboxByID gets an instance of Inbox
func (s *Store) GetInboxByID(ctx context.Context, id int) (*api.Inbox, error) {
	find := &api.InboxFind{ID: &id}
	inboxRaw, err := s.getInboxRawByID(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to get Inbox with ID[%d], error[%w]", id, err)
	}
	if inboxRaw == nil {
		return nil, nil
	}
	inbox, err := s.composeInbox(ctx, inboxRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Inbox with inboxRaw[%+v], error[%w]", inboxRaw, err)
	}
	return inbox, nil
}

// FindInbox finds a list of Inbox instances
func (s *Store) FindInbox(ctx context.Context, find *api.InboxFind) ([]*api.Inbox, error) {
	inboxRawList, err := s.findInboxRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Inbox list, error[%w]", err)
	}
	var inboxList []*api.Inbox
	for _, raw := range inboxRawList {
		inbox, err := s.composeInbox(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Inbox with inboxRaw[%+v], error[%w]", raw, err)
		}
		inboxList = append(inboxList, inbox)
	}
	return inboxList, nil
}

// PatchInbox patches an instance of Inbox
func (s *Store) PatchInbox(ctx context.Context, patch *api.InboxPatch) (*api.Inbox, error) {
	inboxRaw, err := s.patchInboxRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Inbox with InboxPatch[%+v], error[%w]", patch, err)
	}
	inbox, err := s.composeInbox(ctx, inboxRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Inbox with inboxRaw[%+v], error[%w]", inboxRaw, err)
	}
	return inbox, nil
}

// FindInboxSummary returns the inbox summary for a particular principal
func (s *Store) FindInboxSummary(ctx context.Context, principalID int) (*api.InboxSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	row, err := tx.PTx.QueryContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM inbox WHERE receiver_id = $1 AND status = 'UNREAD')
	`,
		principalID,
	)
	if err != nil {
		return nil, FormatError(err)
	}

	row.Next()
	var inboxSummary api.InboxSummary
	if err := row.Scan(
		&inboxSummary.HasUnread,
	); err != nil {
		return nil, FormatError(err)
	}
	if err := row.Close(); err != nil {
		return nil, FormatError(err)
	}

	if inboxSummary.HasUnread {
		row2, err := tx.PTx.QueryContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM inbox, activity WHERE inbox.receiver_id = $1 AND inbox.status = 'UNREAD' AND inbox.activity_id = activity.id AND activity.level = 'ERROR')
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

// composeInbox composes an instance of Inbox by inboxRaw
func (s *Store) composeInbox(ctx context.Context, raw *inboxRaw) (*api.Inbox, error) {
	inbox := raw.toInbox()

	activity, err := s.composeActivity(ctx, raw.ActivityRaw)
	if err != nil {
		return nil, err
	}
	inbox.Activity = activity

	return inbox, nil
}

// createInboxRaw creates a new inbox.
func (s *Store) createInboxRaw(ctx context.Context, create *api.InboxCreate) (*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	inbox, err := s.createInboxImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return inbox, nil
}

// findInboxRaw retrieves a list of inboxes based on find.
func (s *Store) findInboxRaw(ctx context.Context, find *api.InboxFind) ([]*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findInboxImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getInboxRawByID retrieves a single inbox based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getInboxRawByID(ctx context.Context, find *api.InboxFind) (*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findInboxImpl(ctx, tx.PTx, find)
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

// patchInboxRaw updates an existing inbox by ID.
// Returns ENOTFOUND if inbox does not exist.
func (s *Store) patchInboxRaw(ctx context.Context, patch *api.InboxPatch) (*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	inbox, err := s.patchInboxImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return inbox, nil
}

// createInboxImpl creates a new inbox.
func (s *Store) createInboxImpl(ctx context.Context, tx *sql.Tx, create *api.InboxCreate) (*inboxRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO inbox (
			receiver_id,
			activity_id,
			status
		)
		VALUES ($1, $2, 'UNREAD')
		RETURNING id, receiver_id, activity_id, status
	`,
		create.ReceiverID,
		create.ActivityID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var inboxRaw inboxRaw
	var activityID int
	if err := row.Scan(
		&inboxRaw.ID,
		&inboxRaw.ReceiverID,
		&activityID,
		&inboxRaw.Status,
	); err != nil {
		return nil, FormatError(err)
	}

	activityRaw, err := s.getActivityRawByID(ctx, activityID)
	if err != nil {
		return nil, FormatError(err)
	}
	inboxRaw.ActivityRaw = activityRaw

	return &inboxRaw, nil
}

func findInboxImpl(ctx context.Context, tx *sql.Tx, find *api.InboxFind) ([]*inboxRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where = append(where, "inbox.activity_id = activity.id")
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("inbox.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ReceiverID; v != nil {
		where, args = append(where, fmt.Sprintf("receiver_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ReadCreatedAfterTs; v != nil {
		where, args = append(where, fmt.Sprintf("(status != 'READ' OR created_ts >= $%d)", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			inbox.id,
			receiver_id,
			status,
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

	// Iterate over result set and deserialize rows into inboxRawList.
	var inboxRawList []*inboxRaw
	for rows.Next() {
		var inboxRaw inboxRaw
		inboxRaw.ActivityRaw = &activityRaw{}
		if err := rows.Scan(
			&inboxRaw.ID,
			&inboxRaw.ReceiverID,
			&inboxRaw.Status,
			&inboxRaw.ActivityRaw.ID,
			&inboxRaw.ActivityRaw.CreatorID,
			&inboxRaw.ActivityRaw.CreatedTs,
			&inboxRaw.ActivityRaw.UpdaterID,
			&inboxRaw.ActivityRaw.UpdatedTs,
			&inboxRaw.ActivityRaw.ContainerID,
			&inboxRaw.ActivityRaw.Type,
			&inboxRaw.ActivityRaw.Level,
			&inboxRaw.ActivityRaw.Comment,
			&inboxRaw.ActivityRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		inboxRawList = append(inboxRawList, &inboxRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return inboxRawList, nil
}

// patchInboxImpl updates a inbox by ID. Returns the new state of the inbox after update.
func (s *Store) patchInboxImpl(ctx context.Context, tx *sql.Tx, patch *api.InboxPatch) (*inboxRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"status = $1"}, []interface{}{patch.Status}
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE inbox
		SET `+strings.Join(set, ", ")+`
		WHERE id = $2
		RETURNING id, receiver_id, activity_id, status
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var inboxRaw inboxRaw
		var activityID int
		if err := row.Scan(
			&inboxRaw.ID,
			&inboxRaw.ReceiverID,
			&activityID,
			&inboxRaw.Status,
		); err != nil {
			return nil, FormatError(err)
		}

		activityRaw, err := s.getActivityRawByID(ctx, activityID)
		if err != nil {
			return nil, FormatError(err)
		}
		inboxRaw.ActivityRaw = activityRaw

		return &inboxRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("inbox ID not found: %d", patch.ID)}
}
