package store

import (
	"context"
	"database/sql"
)

// ResetPlanWebhookDelivery deletes the webhook delivery record for a plan.
// Called when user clicks BatchRunTasks to enable new notifications on retry.
func (s *Store) ResetPlanWebhookDelivery(ctx context.Context, projectID string, planID int64) error {
	query := `DELETE FROM plan_webhook_delivery WHERE project = $1 AND plan_id = $2`
	_, err := s.GetDB().ExecContext(ctx, query, projectID, planID)
	return err
}

// ClaimPipelineFailureNotification attempts to claim the right to send PIPELINE_FAILED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineFailureNotification(ctx context.Context, projectID string, planID int64) (bool, error) {
	// RETURNING is needed to distinguish insert-success (row returned) from ON CONFLICT DO NOTHING (sql.ErrNoRows).
	query := `
		INSERT INTO plan_webhook_delivery (project, plan_id, event_type)
		VALUES ($1, $2, 'PIPELINE_FAILED')
		ON CONFLICT (project, plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.GetDB().QueryRowContext(ctx, query, projectID, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}

// ClaimPipelineCompletionNotification attempts to claim the right to send PIPELINE_COMPLETED webhook.
// Returns true if claimed (should send), false if already sent or claimed by another replica.
// HA-safe: PRIMARY KEY constraint prevents duplicate sends across replicas.
func (s *Store) ClaimPipelineCompletionNotification(ctx context.Context, projectID string, planID int64) (bool, error) {
	query := `
		INSERT INTO plan_webhook_delivery (project, plan_id, event_type)
		VALUES ($1, $2, 'PIPELINE_COMPLETED')
		ON CONFLICT (project, plan_id) DO NOTHING
		RETURNING plan_id
	`

	var id int64
	err := s.GetDB().QueryRowContext(ctx, query, projectID, planID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Already exists
	}
	return err == nil, err
}
