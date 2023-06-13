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

// inboxRaw is the store model for an Inbox.
// Fields have exactly the same meanings as Inbox.
type inboxRaw struct {
	ID int

	// Domain specific fields
	ReceiverID int
	ActivityID int
	Status     api.InboxStatus
}

// toInbox creates an instance of Inbox based on the inboxRaw.
// This is intended to be called when we need to compose an Inbox relationship.
func (raw *inboxRaw) toInbox() *api.Inbox {
	return &api.Inbox{
		ID: raw.ID,

		ReceiverID: raw.ReceiverID,
		ActivityID: raw.ActivityID,
		Status:     raw.Status,
	}
}

// CreateInbox creates an instance of Inbox.
func (s *Store) CreateInbox(ctx context.Context, create *api.InboxCreate) (*api.Inbox, error) {
	inboxRaw, err := s.createInboxRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Inbox with InboxCreate[%+v]", create)
	}
	return inboxRaw.toInbox(), nil
}

// GetInboxByID gets an instance of Inbox.
func (s *Store) GetInboxByID(ctx context.Context, id int) (*api.Inbox, error) {
	find := &api.InboxFind{ID: &id}
	inboxRaw, err := s.getInboxRawByID(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Inbox with ID %d", id)
	}
	if inboxRaw == nil {
		return nil, nil
	}
	return inboxRaw.toInbox(), nil
}

// FindInbox finds a list of Inbox instances.
func (s *Store) FindInbox(ctx context.Context, find *api.InboxFind) ([]*api.Inbox, error) {
	inboxRawList, err := s.findInboxRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Inbox list with InboxFind[%+v]", find)
	}
	var inboxList []*api.Inbox
	for _, raw := range inboxRawList {
		inboxList = append(inboxList, raw.toInbox())
	}
	return inboxList, nil
}

// PatchInbox patches an instance of Inbox.
func (s *Store) PatchInbox(ctx context.Context, patch *api.InboxPatch) (*api.Inbox, error) {
	inboxRaw, err := s.patchInboxRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Inbox with InboxPatch[%+v]", patch)
	}
	return inboxRaw.toInbox(), nil
}

// FindInboxSummary returns the inbox summary for a particular principal.
func (s *Store) FindInboxSummary(ctx context.Context, principalID int) (*api.InboxSummary, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `SELECT EXISTS (SELECT 1 FROM inbox WHERE receiver_id = $1 AND status = 'UNREAD')`
	var inboxSummary api.InboxSummary
	if err := tx.QueryRowContext(ctx, query, principalID).Scan(&inboxSummary.HasUnread); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if inboxSummary.HasUnread {
		query2 := `SELECT EXISTS (SELECT 1 FROM inbox, activity WHERE inbox.receiver_id = $1 AND inbox.status = 'UNREAD' AND inbox.activity_id = activity.id AND activity.level = 'ERROR')`
		if err := tx.QueryRowContext(ctx, query2, principalID).Scan(&inboxSummary.HasUnreadError); err != nil {
			if err == sql.ErrNoRows {
				return nil, common.FormatDBErrorEmptyRowWithQuery(query2)
			}
			return nil, err
		}
	} else {
		inboxSummary.HasUnreadError = false
	}

	return &inboxSummary, nil
}

//
// private function
//

// createInboxRaw creates a new inbox.
func (s *Store) createInboxRaw(ctx context.Context, create *api.InboxCreate) (*inboxRaw, error) {
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

// findInboxRaw retrieves a list of inboxes based on find.
func (s *Store) findInboxRaw(ctx context.Context, find *api.InboxFind) ([]*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findInboxImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getInboxRawByID retrieves a single inbox based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getInboxRawByID(ctx context.Context, find *api.InboxFind) (*inboxRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findInboxImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d inboxes with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchInboxRaw updates an existing inbox by ID.
// Returns ENOTFOUND if inbox does not exist.
func (s *Store) patchInboxRaw(ctx context.Context, patch *api.InboxPatch) (*inboxRaw, error) {
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

// createInboxImpl creates a new inbox.
func createInboxImpl(ctx context.Context, tx *Tx, create *api.InboxCreate) (*inboxRaw, error) {
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
	var inboxRaw inboxRaw
	if err := tx.QueryRowContext(ctx, query,
		create.ReceiverID,
		create.ActivityID,
	).Scan(
		&inboxRaw.ID,
		&inboxRaw.ReceiverID,
		&inboxRaw.ActivityID,
		&inboxRaw.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	return &inboxRaw, nil
}

func findInboxImpl(ctx context.Context, tx *Tx, find *api.InboxFind) ([]*inboxRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
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

	// Iterate over result set and deserialize rows into inboxRawList.
	var inboxRawList []*inboxRaw
	for rows.Next() {
		var inboxRaw inboxRaw
		if err := rows.Scan(
			&inboxRaw.ID,
			&inboxRaw.ReceiverID,
			&inboxRaw.ActivityID,
			&inboxRaw.Status,
		); err != nil {
			return nil, err
		}
		inboxRawList = append(inboxRawList, &inboxRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inboxRawList, nil
}

// patchInboxImpl updates a inbox by ID. Returns the new state of the inbox after update.
func patchInboxImpl(ctx context.Context, tx *Tx, patch *api.InboxPatch) (*inboxRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"status = $1"}, []any{patch.Status}
	args = append(args, patch.ID)

	var inboxRaw inboxRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE inbox
		SET `+strings.Join(set, ", ")+`
		WHERE id = $2
		RETURNING id, receiver_id, activity_id, status
	`,
		args...,
	).Scan(
		&inboxRaw.ID,
		&inboxRaw.ReceiverID,
		&inboxRaw.ActivityID,
		&inboxRaw.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("inbox ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	return &inboxRaw, nil
}
