-- Tracks webhook delivery for pipeline events (PIPELINE_FAILED or PIPELINE_COMPLETED).
-- One row per plan at any time - mutually exclusive events.
-- Row is deleted when user clicks BatchRunTasks to reset notification state.
CREATE TABLE plan_webhook_delivery (
    plan_id BIGINT PRIMARY KEY REFERENCES plan(id),
    -- Event type: 'PIPELINE_FAILED' or 'PIPELINE_COMPLETED'
    event_type TEXT NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
