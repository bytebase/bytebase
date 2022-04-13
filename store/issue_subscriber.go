package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
)

// issueSubscriberRaw is the store model for an IssueSubscriber.
// Fields have exactly the same meanings as IssueSubscriber.
type issueSubscriberRaw struct {
	IssueID      int
	SubscriberID int
}

// toIssueSubscriber creates an instance of IssueSubscriber based on the issueSubscriberRaw.
// This is intended to be called when we need to compose an IssueSubscriber relationship.
func (raw *issueSubscriberRaw) toIssueSubscriber() *api.IssueSubscriber {
	return &api.IssueSubscriber{
		IssueID:      raw.IssueID,
		SubscriberID: raw.SubscriberID,
	}
}

// CreateIssueSubscriber creates an instance of IssueSubscriber
func (s *Store) CreateIssueSubscriber(ctx context.Context, create *api.IssueSubscriberCreate) (*api.IssueSubscriber, error) {
	issueSubRaw, err := s.createIssueSubscriberRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create IssueSubscriber with IssueSubscriberCreate[%+v], error[%w]", create, err)
	}
	issueSub, err := s.composeIssueSubscriber(ctx, issueSubRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose IssueSubscriber with issueSubRaw[%+v], error[%w]", issueSubRaw, err)
	}
	return issueSub, nil
}

// FindIssueSubscriber finds a list of IssueSubscriber instances
func (s *Store) FindIssueSubscriber(ctx context.Context, find *api.IssueSubscriberFind) ([]*api.IssueSubscriber, error) {
	issueSubRawList, err := s.findIssueSubscriberRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find IssueSubscriber list, error[%w]", err)
	}
	var issueSubList []*api.IssueSubscriber
	for _, raw := range issueSubRawList {
		issueSub, err := s.composeIssueSubscriber(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose IssueSubscriber with issueSubRaw[%+v], error[%w]", raw, err)
		}
		issueSubList = append(issueSubList, issueSub)
	}
	return issueSubList, nil
}

// DeleteIssueSubscriber deletes an existing issueSubscriber by ID.
func (s *Store) DeleteIssueSubscriber(ctx context.Context, delete *api.IssueSubscriberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteIssueSubscriberImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

//
// private functions
//

func (s *Store) composeIssueSubscriber(ctx context.Context, raw *issueSubscriberRaw) (*api.IssueSubscriber, error) {
	issueSubscriber := raw.toIssueSubscriber()

	subscriber, err := s.GetPrincipalByID(ctx, issueSubscriber.SubscriberID)
	if err != nil {
		return nil, err
	}
	issueSubscriber.Subscriber = subscriber

	return issueSubscriber, nil
}

// createIssueSubscriberRaw creates a new issueSubscriber.
func (s *Store) createIssueSubscriberRaw(ctx context.Context, create *api.IssueSubscriberCreate) (*issueSubscriberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	issueSubscriber, err := createIssueSubscriberImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return issueSubscriber, nil
}

// findIssueSubscriberRaw retrieves a list of issueSubscribers based on find.
func (s *Store) findIssueSubscriberRaw(ctx context.Context, find *api.IssueSubscriberFind) ([]*issueSubscriberRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findIssueSubscriberImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createIssueSubscriberImpl creates a new issueSubscriber.
func createIssueSubscriberImpl(ctx context.Context, tx *sql.Tx, create *api.IssueSubscriberCreate) (*issueSubscriberRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO issue_subscriber (
			issue_id,
			subscriber_id
		)
		VALUES ($1, $2)
		RETURNING issue_id, subscriber_id
	`,
		create.IssueID,
		create.SubscriberID,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var issueSubscriberRaw issueSubscriberRaw
	if err := row.Scan(
		&issueSubscriberRaw.IssueID,
		&issueSubscriberRaw.SubscriberID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &issueSubscriberRaw, nil
}

func findIssueSubscriberImpl(ctx context.Context, tx *sql.Tx, find *api.IssueSubscriberFind) ([]*issueSubscriberRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.IssueID; v != nil {
		where, args = append(where, fmt.Sprintf("issue_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, fmt.Sprintf("subscriber_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			issue_id,
			subscriber_id
		FROM issue_subscriber
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into issueSubscriberRawList.
	var issueSubscriberRawList []*issueSubscriberRaw
	for rows.Next() {
		var issueSubscriberRaw issueSubscriberRaw
		if err := rows.Scan(
			&issueSubscriberRaw.IssueID,
			&issueSubscriberRaw.SubscriberID,
		); err != nil {
			return nil, FormatError(err)
		}

		issueSubscriberRawList = append(issueSubscriberRawList, &issueSubscriberRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return issueSubscriberRawList, nil
}

// deleteIssueSubscriberImpl permanently deletes a issueSubscriber by ID.
func deleteIssueSubscriberImpl(ctx context.Context, tx *sql.Tx, delete *api.IssueSubscriberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM issue_subscriber WHERE issue_id = $1 AND subscriber_id = $2`, delete.IssueID, delete.SubscriberID); err != nil {
		return FormatError(err)
	}
	return nil
}
