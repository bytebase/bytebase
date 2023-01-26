package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
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

// FindIssueSubscriber finds a list of IssueSubscriber instances.
func (s *Store) FindIssueSubscriber(ctx context.Context, find *api.IssueSubscriberFind) ([]*api.IssueSubscriber, error) {
	issueSubRawList, err := s.findIssueSubscriberRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find IssueSubscriber list with IssueSubscriberFind[%+v]", find)
	}
	var issueSubList []*api.IssueSubscriber
	for _, raw := range issueSubRawList {
		issueSub, err := s.composeIssueSubscriber(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose IssueSubscriber with issueSubRaw[%+v]", raw)
		}
		issueSubList = append(issueSubList, issueSub)
	}
	return issueSubList, nil
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

// findIssueSubscriberRaw retrieves a list of issueSubscribers based on find.
func (s *Store) findIssueSubscriberRaw(ctx context.Context, find *api.IssueSubscriberFind) ([]*issueSubscriberRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findIssueSubscriberImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func findIssueSubscriberImpl(ctx context.Context, tx *Tx, find *api.IssueSubscriberFind) ([]*issueSubscriberRaw, error) {
	// Build WHERE clause.
	where, args := []string{"TRUE"}, []interface{}{}
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
