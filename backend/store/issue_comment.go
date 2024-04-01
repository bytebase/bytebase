package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type IssueCommentMessage struct {
	UID        int
	CreatorUID int
	CreatedTs  int64
	UpdaterUID int
	UpdatedTs  int64
	IssueUID   int
	Payload    *storepb.IssueCommentPayload
}

type FindIssueCommentMessage struct {
	IssueUID *int
}

func (s *Store) ListIssueComment(ctx context.Context, find *FindIssueCommentMessage) ([]*IssueCommentMessage, error) {
	where := []string{"TRUE"}
	args := []any{}
	if v := find.IssueUID; v != nil {
		where = append(where, fmt.Sprintf("issue_id = $%d", len(args)+1))
		args = append(args, *v)
	}

	rows, err := s.db.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			issue_id,
			payload
		FROM
			issue_comment
		WHERE %s
		ORDER BY id ASC
	`, strings.Join(where, " AND ")), args...)
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
			&ic.CreatorUID,
			&ic.CreatedTs,
			&ic.UpdaterUID,
			&ic.UpdatedTs,
			&ic.IssueUID,
			&p,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		if err := protojson.Unmarshal(p, ic.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}
		issueComments = append(issueComments, &ic)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return issueComments, nil
}

func (s *Store) CreateIssueComment(ctx context.Context, create *IssueCommentMessage, creatorUID int) error {
	query := `
		INSERT INTO issue_comment (
			creator_id,
			updater_id,
			issue_id,
			payload
		) VALUES (
			$1,
			$2,
			$3,
			$4
		)
	`

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	if _, err := s.db.db.ExecContext(ctx, query, creatorUID, creatorUID, create.IssueUID, payload); err != nil {
		return errors.Wrapf(err, "failed to insert")
	}

	return nil
}

func (s *Store) UpdateIssueComment(ctx context.Context, comment string, uid int, updaterUID int) error {
	query := `
		UPDATE issue_comment
		SET 
			updater_id = $1,
			payload = payload || jsonb_build_object('comment', $2)
		WHERE id = $3
	`
	if _, err := s.db.db.ExecContext(ctx, query, updaterUID, comment, uid); err != nil {
		return errors.Wrapf(err, "failed to update issue comment")
	}
	return nil
}
