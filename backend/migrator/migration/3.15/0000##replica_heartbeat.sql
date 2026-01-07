-- Create replica heartbeat table
CREATE TABLE replica_heartbeat (
    replica_id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMPTZ NOT NULL
);

-- Add replica_id column to task_run
ALTER TABLE task_run ADD COLUMN replica_id TEXT;

CREATE INDEX idx_task_run_replica_id ON task_run(replica_id) WHERE replica_id IS NOT NULL;

-- Mark existing RUNNING task runs as FAILED
UPDATE task_run
SET status = 'FAILED',
    result = '{"detail": "Marked as failed during heartbeat migration"}'
WHERE status = 'RUNNING';
