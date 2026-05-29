package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// VCSProviderUserMessage is the store message for a VCS provider user.
type VCSProviderUserMessage struct {
	Workspace  string
	VCSType    v1pb.VCSType
	UserID     string
	LastSeenAt time.Time
	Payload    *storepb.VCSProviderUserPayload
}

// CountActiveVCSProviderUsers counts active VCS provider users in the workspace.
func (s *Store) CountActiveVCSProviderUsers(ctx context.Context, workspace string, activeWindow time.Duration) (int, error) {
	var count int
	if err := s.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM vcs_provider_user
		WHERE workspace = $1
			AND last_seen_at >= now() - make_interval(secs => $2)
	`, workspace, activeWindow.Seconds()).Scan(&count); err != nil {
		return 0, errors.Wrapf(err, "failed to count active VCS provider users")
	}
	return count, nil
}

// TouchVCSProviderUser refreshes an active VCS provider user, or inserts/reactivates
// one when doing so would not exceed the active user limit.
func (s *Store) TouchVCSProviderUser(ctx context.Context, workspace string, user *VCSProviderUserMessage, activeWindow time.Duration, limit int) (bool, error) {
	payload := user.Payload
	if payload == nil {
		payload = &storepb.VCSProviderUserPayload{}
	}
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal VCS provider user payload")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := AcquireAdvisoryXactLock(ctx, tx, AdvisoryLockKeyVCSProviderUser); err != nil {
		return false, errors.Wrapf(err, "failed to acquire VCS provider user lock")
	}

	vcsType := user.VCSType.String()
	var active bool
	if err := tx.QueryRowContext(ctx, `
		SELECT last_seen_at >= now() - make_interval(secs => $4)
		FROM vcs_provider_user
		WHERE workspace = $1 AND vcs_type = $2 AND user_id = $3
	`, workspace, vcsType, user.UserID, activeWindow.Seconds()).Scan(&active); err != nil && err != sql.ErrNoRows {
		return false, errors.Wrapf(err, "failed to get VCS provider user")
	}

	if active {
		if _, err := tx.ExecContext(ctx, `
			UPDATE vcs_provider_user
			SET last_seen_at = now(), payload = $4
			WHERE workspace = $1 AND vcs_type = $2 AND user_id = $3
		`, workspace, vcsType, user.UserID, payloadBytes); err != nil {
			return false, errors.Wrapf(err, "failed to update VCS provider user")
		}
		if err := tx.Commit(); err != nil {
			return false, errors.Wrapf(err, "failed to commit transaction")
		}
		return true, nil
	}

	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM vcs_provider_user
		WHERE workspace = $1
			AND last_seen_at >= now() - make_interval(secs => $2)
	`, workspace, activeWindow.Seconds()).Scan(&count); err != nil {
		return false, errors.Wrapf(err, "failed to count active VCS provider users")
	}
	if count >= limit {
		if err := tx.Commit(); err != nil {
			return false, errors.Wrapf(err, "failed to commit transaction")
		}
		return false, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO vcs_provider_user (workspace, vcs_type, user_id, last_seen_at, payload)
		VALUES ($1, $2, $3, now(), $4)
		ON CONFLICT (workspace, vcs_type, user_id) DO UPDATE SET
			last_seen_at = now(),
			payload = EXCLUDED.payload
	`, workspace, vcsType, user.UserID, payloadBytes); err != nil {
		return false, errors.Wrapf(err, "failed to upsert VCS provider user")
	}

	if err := tx.Commit(); err != nil {
		return false, errors.Wrapf(err, "failed to commit transaction")
	}
	return true, nil
}

// ListActiveVCSProviderUsers lists active VCS provider users in the workspace,
// sorted by most recently seen first.
func (s *Store) ListActiveVCSProviderUsers(ctx context.Context, workspace string, activeWindow time.Duration) ([]*VCSProviderUserMessage, error) {
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT workspace, vcs_type, user_id, last_seen_at, payload
		FROM vcs_provider_user
		WHERE workspace = $1
			AND last_seen_at >= now() - make_interval(secs => $2)
		ORDER BY last_seen_at DESC
	`, workspace, activeWindow.Seconds())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list active VCS provider users")
	}
	defer rows.Close()

	var users []*VCSProviderUserMessage
	for rows.Next() {
		user := &VCSProviderUserMessage{
			Payload: &storepb.VCSProviderUserPayload{},
		}
		var vcsType string
		var payload []byte
		if err := rows.Scan(&user.Workspace, &vcsType, &user.UserID, &user.LastSeenAt, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to scan VCS provider user")
		}
		value, ok := v1pb.VCSType_value[vcsType]
		if !ok {
			return nil, errors.Errorf("unknown VCS type %q", vcsType)
		}
		user.VCSType = v1pb.VCSType(value)
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, user.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal VCS provider user payload")
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to list active VCS provider users")
	}
	return users, nil
}
