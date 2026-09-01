package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type IssueCommentMessage struct {
	ProjectID    string
	ResourceID   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IssueUID     int64
	Payload      *storepb.IssueCommentPayload
	CreatorEmail string
}

type FindIssueCommentMessage struct {
	ProjectID  string
	ResourceID *string
	IssueUID   *int64

	Limit  *int
	Offset *int
}

type UpdateIssueCommentMessage struct {
	ProjectID  string
	ResourceID string

	Comment *string
}

func (s *Store) GetIssueComment(ctx context.Context, find *FindIssueCommentMessage) (*IssueCommentMessage, error) {
	list, err := s.ListIssueComment(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > 1 {
		return nil, errors.Errorf("found %d issue comment, expected 1", len(list))
	}
	return list[0], nil
}

func (s *Store) ListIssueComment(ctx context.Context, find *FindIssueCommentMessage) ([]*IssueCommentMessage, error) {
	q := qb.Q().Space(`
		SELECT
			project,
			resource_id,
			creator,
			created_at,
			updated_at,
			issue_id,
			payload
		FROM
			issue_comment
		WHERE project = ?
	`, find.ProjectID)

	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}
	if v := find.IssueUID; v != nil {
		q.And("issue_id = ?", *v)
	}

	// resource_id is the primary key. created_at alone would not be total —
	// it defaults to now(), the transaction timestamp — so CreateIssueComments
	// offsets each row of a batch by its ordinal to keep insertion order; see
	// the comment there.
	q.Space("ORDER BY issue_comment.created_at ASC, issue_comment.resource_id ASC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query context")
	}
	defer rows.Close()

	var issueComments []*IssueCommentMessage
	for rows.Next() {
		ic := IssueCommentMessage{
			Payload: &storepb.IssueCommentPayload{},
		}
		var p []byte
		if err := rows.Scan(
			&ic.ProjectID,
			&ic.ResourceID,
			&ic.CreatorEmail,
			&ic.CreatedAt,
			&ic.UpdatedAt,
			&ic.IssueUID,
			&p,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(p, ic.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}
		issueComments = append(issueComments, &ic)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return issueComments, nil
}

// CreateIssueComments creates one or more root comments or events.
// Reply creation is not supported by this writer yet.
// For a single comment, it returns the created comment with UID, CreatedAt, and UpdatedAt filled in.
// For multiple comments, it performs a batch insert and returns nil.
func (s *Store) CreateIssueComments(ctx context.Context, creator string, creates ...*IssueCommentMessage) (*IssueCommentMessage, error) {
	if len(creates) == 0 {
		return nil, nil
	}

	// Prepare all payloads.
	projectIDs := make([]string, 0, len(creates))
	issueIDs := make([]int64, 0, len(creates))
	payloads := make([][]byte, 0, len(creates))
	threadRoots := make([]bool, 0, len(creates))
	for _, create := range creates {
		payload, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		projectIDs = append(projectIDs, create.ProjectID)
		issueIDs = append(issueIDs, create.IssueUID)
		payloads = append(payloads, payload)
		// Current event-less writes are roots. Event and hybrid rows remain
		// outside threads; replies will additionally require parent_id handling.
		threadRoots = append(threadRoots, create.Payload.GetEvent() == nil)
	}

	// Use UNNEST to insert all comments in one query.
	//
	// created_at defaults to now(), which is the transaction timestamp, so a
	// batch would otherwise land on one instant and ListIssueComment could only
	// fall back to its resource_id tiebreak — a random UUID, which scrambles the
	// activity feed of a multi-field UpdateIssue. Offsetting each row by its
	// ordinal keeps the batch in insertion order and makes created_at unique
	// within it, at a cost of microseconds.
	q := qb.Q().Space(`
		INSERT INTO issue_comment (creator, project, issue_id, payload, thread_state, created_at, updated_at)
		SELECT ?, c.project, c.issue_id, c.payload, CASE WHEN c.is_thread_root THEN 'OPEN' END, t.at, t.at
		FROM unnest(?::TEXT[], ?::INT[], ?::JSONB[], ?::BOOLEAN[])
		     WITH ORDINALITY AS c(project, issue_id, payload, is_thread_root, ordinality),
		     LATERAL (SELECT now() + ((c.ordinality - 1) * interval '1 microsecond')) AS t(at)
	`, creator, projectIDs, issueIDs, payloads, threadRoots)

	// For single comment, use RETURNING to get the created comment details.
	if len(creates) == 1 {
		q.Space("RETURNING resource_id, created_at, updated_at")
		query, args, err := q.ToSQL()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build sql")
		}

		create := creates[0]
		if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.ResourceID, &create.CreatedAt, &create.UpdatedAt); err != nil {
			return nil, errors.Wrapf(err, "failed to insert")
		}
		create.CreatorEmail = creator
		return create, nil
	}

	// For multiple comments, just execute without RETURNING.
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to batch insert comments")
	}

	return nil, nil
}

func (s *Store) UpdateIssueComment(ctx context.Context, patch *UpdateIssueCommentMessage) error {
	q := qb.Q().Space("UPDATE issue_comment SET updated_at = ?", time.Now())

	if v := patch.Comment; v != nil {
		q.Join(", ", "payload = payload || jsonb_build_object('comment',?::TEXT)", *v)
	}

	q.Space("WHERE project = ? AND resource_id = ?", patch.ProjectID, patch.ResourceID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update issue comment")
	}
	return nil
}
