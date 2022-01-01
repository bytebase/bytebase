package store

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.IssueSubscriberService = (*IssueSubscriberService)(nil)
)

// IssueSubscriberService represents a service for managing issueSubscriber.
type IssueSubscriberService struct {
	l  *zap.Logger
	db *DB
}

// NewIssueSubscriberService returns a new instance of IssueSubscriberService.
func NewIssueSubscriberService(logger *zap.Logger, db *DB) *IssueSubscriberService {
	return &IssueSubscriberService{l: logger, db: db}
}

// CreateIssueSubscriber creates a new issueSubscriber.
func (s *IssueSubscriberService) CreateIssueSubscriber(ctx context.Context, create *api.IssueSubscriberCreate) (*api.IssueSubscriber, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	issueSubscriber, err := createIssueSubscriber(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return issueSubscriber, nil
}

// FindIssueSubscriberList retrieves a list of issueSubscribers based on find.
func (s *IssueSubscriberService) FindIssueSubscriberList(ctx context.Context, find *api.IssueSubscriberFind) ([]*api.IssueSubscriber, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findIssueSubscriberList(ctx, tx, find)
	if err != nil {
		return []*api.IssueSubscriber{}, err
	}

	return list, nil
}

// DeleteIssueSubscriber deletes an existing issueSubscriber by ID.
func (s *IssueSubscriberService) DeleteIssueSubscriber(ctx context.Context, delete *api.IssueSubscriberDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteIssueSubscriber(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createIssueSubscriber creates a new issueSubscriber.
func createIssueSubscriber(ctx context.Context, tx *Tx, create *api.IssueSubscriberCreate) (*api.IssueSubscriber, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO issue_subscriber (
			issue_id,
			subscriber_id
		)
		VALUES (?, ?)
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
	var issueSubscriber api.IssueSubscriber
	if err := row.Scan(
		&issueSubscriber.IssueID,
		&issueSubscriber.SubscriberID,
	); err != nil {
		return nil, FormatError(err)
	}

	return &issueSubscriber, nil
}

func findIssueSubscriberList(ctx context.Context, tx *Tx, find *api.IssueSubscriberFind) (_ []*api.IssueSubscriber, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.IssueID; v != nil {
		where, args = append(where, "issue_id = ?"), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, "subscriber_id = ?"), append(args, *v)
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.IssueSubscriber, 0)
	for rows.Next() {
		var issueSubscriber api.IssueSubscriber
		if err := rows.Scan(
			&issueSubscriber.IssueID,
			&issueSubscriber.SubscriberID,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &issueSubscriber)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteIssueSubscriber permanently deletes a issueSubscriber by ID.
func deleteIssueSubscriber(ctx context.Context, tx *Tx, delete *api.IssueSubscriberDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM issue_subscriber WHERE issue_id = ? AND subscriber_id = ?`, delete.IssueID, delete.SubscriberID); err != nil {
		return FormatError(err)
	}
	return nil
}
