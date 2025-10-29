-- Add partial index on task_run for active statuses
--
-- This migration adds a partial index to optimize queries filtering for active task runs.
--
-- Background:
-- Task runs go through state transitions: PENDING -> RUNNING -> (DONE|FAILED|CANCELED)
-- Once in a terminal state (DONE, FAILED, CANCELED), status never changes again.
-- Most task runs are in terminal states, but queries frequently filter for active ones (PENDING, RUNNING).
--
-- Problem:
-- Queries like "SELECT ... FROM task_run WHERE status IN ('RUNNING')" were doing sequential scans
-- because there was no index on the status column.
--
-- Solution:
-- Use a partial index that only covers active statuses (PENDING, RUNNING).
-- This is more efficient than a full index because:
-- - Smaller index size (only active tasks, not all historical completed tasks)
-- - Faster index maintenance (terminal states don't consume index space)
-- - Better cache efficiency (smaller index fits in memory)
-- - Matches the query pattern (filters for active statuses)
--
-- Example query optimized:
-- SELECT task_run.id, ... FROM task_run
-- LEFT JOIN task ON task.id = task_run.task_id
-- WHERE task_run.status IN ('RUNNING')
-- ORDER BY task_run.id ASC

CREATE INDEX idx_task_run_active_status_id ON task_run(status, id)
WHERE status IN ('PENDING', 'RUNNING');
