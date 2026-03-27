package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SubscriptionMessage is the message for a workspace subscription.
type SubscriptionMessage struct {
	Workspace string
	Payload   *storepb.SubscriptionPayload
	Etag      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateSubscription creates a new subscription.
func (s *Store) CreateSubscription(ctx context.Context, create *SubscriptionMessage) (*SubscriptionMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	if err := s.GetDB().QueryRowContext(ctx,
		`INSERT INTO subscription (workspace, payload) VALUES ($1, $2) RETURNING created_at, updated_at`,
		create.Workspace, payload,
	).Scan(&create.CreatedAt, &create.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to insert subscription")
	}
	create.Etag = generateEtag(create.UpdatedAt)

	return create, nil
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

// UpdateSubscription updates a subscription by workspace.
func (s *Store) UpdateSubscription(ctx context.Context, workspace string, payload *storepb.SubscriptionPayload) (*SubscriptionMessage, error) {
	p, err := protojson.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	sub := &SubscriptionMessage{
		Payload: &storepb.SubscriptionPayload{},
	}
	var payloadBytes []byte
	if err := s.GetDB().QueryRowContext(ctx,
		`UPDATE subscription SET payload = $1, updated_at = now() WHERE workspace = $2
		 RETURNING workspace, payload, created_at, updated_at`,
		p, workspace,
	).Scan(&sub.Workspace, &payloadBytes, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
		return nil, errors.Wrapf(err, "failed to update subscription")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, sub.Payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload")
	}
	sub.Etag = fmt.Sprintf("%d", sub.UpdatedAt.UnixMilli())
	return sub, nil
}
