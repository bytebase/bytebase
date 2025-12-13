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
	UID          int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IssueUID     int
	Payload      *storepb.IssueCommentPayload
	CreatorEmail string
}

type FindIssueCommentMessage struct {
	UID      *int
	IssueUID *int

	Limit  *int
	Offset *int
}

type UpdateIssueCommentMessage struct {
	UID int

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
			id,
			creator,
			created_at,
			updated_at,
			issue_id,
			payload
		FROM
			issue_comment
		WHERE TRUE
	`)

	if v := find.UID; v != nil {
		q.And("id = ?", *v)
	}
	if v := find.IssueUID; v != nil {
		q.And("issue_id = ?", *v)
	}

	q.Space("ORDER BY id ASC")
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
			&ic.UID,
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

func (s *Store) CreateIssueCommentTaskUpdateStatus(ctx context.Context, issueUID int, tasks []string, status storepb.TaskRun_Status, creator string, comment string) error {
	create := &IssueCommentMessage{
		IssueUID: issueUID,
		Payload: &storepb.IssueCommentPayload{
			Comment: comment,
			Event: &storepb.IssueCommentPayload_TaskUpdate_{
				TaskUpdate: &storepb.IssueCommentPayload_TaskUpdate{
					Tasks:    tasks,
					ToStatus: &status,
				},
			},
		},
	}

	_, err := s.CreateIssueComment(ctx, create, creator)
	return err
}

func (s *Store) CreateIssueComment(ctx context.Context, create *IssueCommentMessage, creator string) (*IssueCommentMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	q := qb.Q().Space(`
		INSERT INTO issue_comment (
			creator,
			issue_id,
			payload
		) VALUES (
			?,
			?,
			?
		) RETURNING id, created_at, updated_at
	`, creator, create.IssueUID, payload)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.UID, &create.CreatedAt, &create.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to insert")
	}

	create.CreatorEmail = creator

	return create, nil
}

func (s *Store) UpdateIssueComment(ctx context.Context, patch *UpdateIssueCommentMessage) error {
	q := qb.Q().Space("UPDATE issue_comment SET updated_at = ?", time.Now())

	if v := patch.Comment; v != nil {
		q.Join(", ", "payload = payload || jsonb_build_object('comment',?::TEXT)", *v)
	}

	q.Space("WHERE id = ?", patch.UID)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update issue comment")
	}
	return nil
}
