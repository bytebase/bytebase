package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ErrTrialNotEligible means the workspace already has subscription history or a license.
var ErrTrialNotEligible = errors.New("workspace is not eligible for a trial")

// SubscriptionMessage is the message for a workspace subscription.
type SubscriptionMessage struct {
	Workspace string
	Payload   *storepb.SubscriptionPayload
	Etag      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UpsertSubscription creates or updates a subscription by workspace.
func (s *Store) UpsertSubscription(ctx context.Context, workspace string, payload *storepb.SubscriptionPayload) (*SubscriptionMessage, error) {
	p, err := protojson.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	sub := &SubscriptionMessage{
		Payload: &storepb.SubscriptionPayload{},
	}
	var payloadBytes []byte
	if err := s.GetDB().QueryRowContext(ctx,
		`INSERT INTO subscription (workspace, payload) VALUES ($1, $2)
		 ON CONFLICT (workspace) DO UPDATE SET payload = $2, updated_at = now()
		 RETURNING workspace, payload, created_at, updated_at`,
		workspace, p,
	).Scan(&sub.Workspace, &payloadBytes, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to upsert subscription")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, sub.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload")
	}
	sub.Etag = fmt.Sprintf("%d", sub.UpdatedAt.UnixMilli())
	return sub, nil
}

// GetSubscriptionByWorkspace gets a subscription by workspace.
// Returns (nil, nil) if not found.
func (s *Store) GetSubscriptionByWorkspace(ctx context.Context, workspace string) (*SubscriptionMessage, error) {
	sub := &SubscriptionMessage{
		Payload: &storepb.SubscriptionPayload{},
	}
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT workspace, payload, created_at, updated_at FROM subscription WHERE workspace = $1`,
		workspace,
	).Scan(&sub.Workspace, &payload, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get subscription")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, sub.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload")
	}
	sub.Etag = generateEtag(sub.UpdatedAt)
	return sub, nil
}

// CreateTrialLicense stores a trial license only when the workspace has no
// subscription history and no existing license.
func (s *Store) CreateTrialLicense(ctx context.Context, workspace string, license string) error {
	existing, err := s.GetSubscriptionByWorkspace(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to check subscription history")
	}
	if existing != nil {
		return ErrTrialNotEligible
	}

	if _, err := s.UpdateSettingAtomic(ctx, workspace, storepb.SettingName_SYSTEM, func(current proto.Message) (proto.Message, error) {
		systemSetting, ok := current.(*storepb.SystemSetting)
		if !ok {
			return nil, errors.Errorf("invalid system setting value type %T", current)
		}
		if systemSetting.License != "" {
			return nil, ErrTrialNotEligible
		}
		systemSetting.License = license
		return systemSetting, nil
	}, nil); err != nil {
		if errors.Is(err, ErrTrialNotEligible) {
			return err
		}
		return errors.Wrap(err, "failed to update system setting")
	}
	return nil
}
