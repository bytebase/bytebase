package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type ChangelogMessage struct {
	DatabaseUID int
	Payload     *storepb.ChangelogPayload

	// output only
	UID         int64
	CreatorUID  int64
	CreatedTime time.Time
}

type UpdateChangelogMessage struct {
	UID int64

	SyncHistoryUID *int64
	RevisionUID    *int64
	Status         *storepb.ChangelogTask_Status
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage, creatorUID int) (int64, error) {
	query := `
		INSERT INTO changelog (
			creator_id,
			database_id,
			payload
		) VALUES (
		 	$1,
			$2,
			$3
		)
		RETURNING id
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	p, err := protojson.Marshal(create.Payload)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	var id int64
	if err := tx.QueryRowContext(ctx, query, creatorUID, create.DatabaseUID, p).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return id, nil
}

func (s *Store) UpdateChangelog(ctx context.Context, update *UpdateChangelogMessage) error {
	args := []any{update.UID}
	var payloadSet []string

	if v := update.SyncHistoryUID; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('syncHistoryUid', $%d::TEXT)", len(args)+1))
		args = append(args, fmt.Sprintf("%d", *v))
	}
	if v := update.RevisionUID; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('revision', $%d::TEXT)", len(args)+1))
		args = append(args, fmt.Sprintf("%d", *v))
	}
	if v := update.Status; v != nil {
		payloadSet = append(payloadSet, fmt.Sprintf("jsonb_build_object('status', $%d::TEXT)", len(args)+1))
		args = append(args, v.String())
	}

	if len(payloadSet) == 0 {
		return errors.Errorf("update nothing")
	}

	query := fmt.Sprintf(`
		UPDATE changelog
		SET payload = jsonb_set(payload, '{task}', payload->'task' || %s)
		WHERE id = $1
	`, strings.Join(payloadSet, " || "))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	return nil
}
