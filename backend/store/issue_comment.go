package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type IssueCommentMessage struct {
	UID       int
	CreatedTs int64
	UpdatedTs int64
	IssueUID  int
	Payload   *storepb.IssueCommentPayload
	Creator   *UserMessage

	creatorUID int
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
	where := []string{"TRUE"}
	args := []any{}
	if v := find.UID; v != nil {
		where = append(where, fmt.Sprintf("id = $%d", len(args)+1))
		args = append(args, *v)
	}
	if v := find.IssueUID; v != nil {
		where = append(where, fmt.Sprintf("issue_id = $%d", len(args)+1))
		args = append(args, *v)
	}

	limitOffsetClause := ""
	if v := find.Limit; v != nil {
		limitOffsetClause = fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		limitOffsetClause += fmt.Sprintf(" OFFSET %d", *v)
	}

	rows, err := s.db.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			creator_id,
			created_ts,
			updated_ts,
			issue_id,
			payload
		FROM
			issue_comment
		WHERE %s
		ORDER BY id ASC
		%s
	`, strings.Join(where, " AND "), limitOffsetClause), args...)
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
			&ic.creatorUID,
			&ic.CreatedTs,
			&ic.UpdatedTs,
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

	for _, ic := range issueComments {
		creator, err := s.GetUserByID(ctx, ic.creatorUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get creator")
		}
		ic.Creator = creator
	}

	return issueComments, nil
}

func (s *Store) CreateIssueCommentTaskUpdateStatus(ctx context.Context, issueUID int, tasks []string, status storepb.IssueCommentPayload_TaskUpdate_Status, creatorUID int, comment string) error {
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

	_, err := s.CreateIssueComment(ctx, create, creatorUID)
	return err
}

func (s *Store) CreateIssueComment(ctx context.Context, create *IssueCommentMessage, creatorUID int) (*IssueCommentMessage, error) {
	query := `
		INSERT INTO issue_comment (
			creator_id,
			issue_id,
			payload
		) VALUES (
			$1,
			$2,
			$3
		) RETURNING id, created_ts, updated_ts
	`

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	var id int
	var createdTs, updatedTs int64
	if err := s.db.db.QueryRowContext(ctx, query, creatorUID, create.IssueUID, payload).Scan(&id, &createdTs, &updatedTs); err != nil {
		return nil, errors.Wrapf(err, "failed to insert")
	}

	ic := &IssueCommentMessage{
		UID:        id,
		CreatedTs:  createdTs,
		UpdatedTs:  updatedTs,
		IssueUID:   create.IssueUID,
		Payload:    create.Payload,
		creatorUID: creatorUID,
	}

	creator, err := s.GetUserByID(ctx, ic.creatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get creator")
	}
	ic.Creator = creator

	return ic, nil
}

func (s *Store) UpdateIssueComment(ctx context.Context, patch *UpdateIssueCommentMessage) error {
	set, args := []string{"updated_ts = $2"}, []any{time.Now().Unix()}

	if v := patch.Comment; v != nil {
		set, args = append(set, fmt.Sprintf("payload = payload || jsonb_build_object('comment',$%d::TEXT)", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.UID)
	query := `UPDATE issue_comment SET ` + strings.Join(set, ", ") + fmt.Sprintf(` WHERE id = $%d`, len(args))
	if _, err := s.db.db.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update issue comment")
	}
	return nil
}
