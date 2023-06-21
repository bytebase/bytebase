package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// InboxMessage is the API message for inbox.
type InboxMessage struct {
	UID int

	// Domain specific fields
	ReceiverUID int
	ActivityUID int
	Status      api.InboxStatus
}

// InboxSummaryMessage is the API message for inbox summary info.
// This is used by the frontend to render the inbox sidebar item without fetching the actual inbox items.
// This returns json instead of jsonapi since it't not dealing with a particular resource.
type InboxSummaryMessage struct {
	Unread      int
	UnreadError int
}

// FindInboxMessage is the API message for finding the inbox.
type FindInboxMessage struct {
	// Domain specific fields
	ReceiverUID *int
	// If specified, then it will only fetch "UNREAD" item or "READ" item whose activity created after "CreatedAfterTs"
	ReadCreatedAfterTs *int64
}

// UpdateInboxMessage is the API message to update the inbox.
type UpdateInboxMessage struct {
	UID int

	// Domain specific fields
	Status api.InboxStatus
}

// CreateInbox creates an instance of Inbox.
func (s *Store) CreateInbox(ctx context.Context, create *InboxMessage) (*InboxMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	inbox, err := createInboxImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return inbox, nil
}

// FindInbox finds a list of Inbox instances.
func (s *Store) FindInbox(ctx context.Context, find *FindInboxMessage) ([]*InboxMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findInboxImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

// PatchInbox patches an instance of Inbox.
// Returns ENOTFOUND if inbox does not exist.
func (s *Store) PatchInbox(ctx context.Context, patch *UpdateInboxMessage) (*InboxMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	inbox, err := patchInboxImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return inbox, nil
}

// FindInboxSummary returns the inbox summary for a particular principal.
func (s *Store) FindInboxSummary(ctx context.Context, principalID int) (*InboxSummaryMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `SELECT COUNT(*) FROM inbox WHERE receiver_id = $1 AND status = 'UNREAD'`
	var inboxSummary InboxSummaryMessage
	if err := tx.QueryRowContext(ctx, query, principalID).Scan(&inboxSummary.Unread); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if inboxSummary.Unread > 0 {
		query2 := `SELECT COUNT(*) FROM inbox, activity WHERE inbox.receiver_id = $1 AND inbox.status = 'UNREAD' AND inbox.activity_id = activity.id AND activity.level = 'ERROR'`
		if err := tx.QueryRowContext(ctx, query2, principalID).Scan(&inboxSummary.UnreadError); err != nil {
			if err == sql.ErrNoRows {
				return nil, common.FormatDBErrorEmptyRowWithQuery(query2)
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &inboxSummary, nil
}

//
// private function
//

// createInboxImpl creates a new inbox.
func createInboxImpl(ctx context.Context, tx *Tx, create *InboxMessage) (*InboxMessage, error) {
	// Insert row into database.
	query := `
		INSERT INTO inbox (
			receiver_id,
			activity_id,
			status
		)
		VALUES ($1, $2, 'UNREAD')
		RETURNING id, receiver_id, activity_id, status
	`
	var response InboxMessage
	if err := tx.QueryRowContext(ctx, query,
		create.ReceiverUID,
		create.ActivityUID,
	).Scan(
		&response.UID,
		&response.ReceiverUID,
		&response.ActivityUID,
		&response.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	return &response, nil
}

func findInboxImpl(ctx context.Context, tx *Tx, find *FindInboxMessage) ([]*InboxMessage, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	if v := find.ReceiverUID; v != nil {
		where, args = append(where, fmt.Sprintf("receiver_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ReadCreatedAfterTs; v != nil {
		where, args = append(where, fmt.Sprintf("(status != 'READ' OR created_ts >= $%d)", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			inbox.id,
			inbox.receiver_id,
			inbox.activity_id,
			inbox.status
		FROM inbox
		LEFT JOIN activity ON inbox.activity_id = activity.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY activity.created_ts DESC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into inboxList.
	var inboxList []*InboxMessage
	for rows.Next() {
		var inbox InboxMessage
		if err := rows.Scan(
			&inbox.UID,
			&inbox.ReceiverUID,
			&inbox.ActivityUID,
			&inbox.Status,
		); err != nil {
			return nil, err
		}
		inboxList = append(inboxList, &inbox)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inboxList, nil
}

// patchInboxImpl updates a inbox by ID. Returns the new state of the inbox after update.
func patchInboxImpl(ctx context.Context, tx *Tx, patch *UpdateInboxMessage) (*InboxMessage, error) {
	// Build UPDATE clause.
	set, args := []string{"status = $1"}, []any{patch.Status}
	args = append(args, patch.UID)

	var response InboxMessage
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE inbox
		SET `+strings.Join(set, ", ")+`
		WHERE id = $2
		RETURNING id, receiver_id, activity_id, status
	`,
		args...,
	).Scan(
		&response.UID,
		&response.ReceiverUID,
		&response.ActivityUID,
		&response.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("inbox ID not found: %d", patch.UID)}
		}
		return nil, err
	}
	return &response, nil
}
