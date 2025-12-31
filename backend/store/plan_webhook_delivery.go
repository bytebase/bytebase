package store

import (
	"context"
	"database/sql"
)

// ResetPlanWebhookDelivery deletes the webhook delivery record for a plan.
// Called when user clicks BatchRunTasks to enable new notifications on retry.
func (s *Store) ResetPlanWebhookDelivery(ctx context.Context, planID int64) error {
	query := `DELETE FROM plan_webhook_delivery WHERE plan_id = $1`
	_, err := s.GetDB().ExecContext(ctx, query, planID)
	return err
}

// ClaimPipelineFailureNotification attempts to claim the right to send PIPELINE_FAILED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineFailureNotification(ctx context.Context, planID int64) (bool, error) {
	query := `
		INSERT INTO plan_webhook_delivery (plan_id, event_type)
		VALUES ($1, 'PIPELINE_FAILED')
		ON CONFLICT (plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.GetDB().QueryRowContext(ctx, query, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}

// ClaimPipelineCompletionNotification attempts to claim the right to send PIPELINE_COMPLETED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineCompletionNotification(ctx context.Context, planID int64) (bool, error) {
	query := `
		INSERT INTO plan_webhook_delivery (plan_id, event_type)
		VALUES ($1, 'PIPELINE_COMPLETED')
		ON CONFLICT (plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.GetDB().QueryRowContext(ctx, query, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}
