-- SQL Review V2 run status slot: one row per (issue, reviewer type), reset in
-- place on re-run (created_at = now() on reset — the row is the current run,
-- not the slot's history). Results live in issue comments; the run carries
-- none. No standalone id on purpose: nothing may durably reference a run.
CREATE TABLE review_run (
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    issue_id bigint NOT NULL,
    -- Reviewer type: 'RULE' (standard rules) or 'GUIDELINE' (natural-language
    -- guidelines, performed by AI). No CHECK on purpose: the reviewer-id space
    -- is open.
    type text NOT NULL,
    -- Attempt number of the current run, 0-based like task_run.attempt.
    -- Bumped on every slot reset (issue created / SQL updated / manual
    -- re-run); it counts triggers, not completed executions. The completion
    -- transaction is fenced on it, so a superseded execution posts zero
    -- comments.
    attempt integer NOT NULL DEFAULT 0,
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED')),
    replica_id text,
    -- Stored as ReviewRunPayload (proto/store/store/review_run.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (project, issue_id, type),
    FOREIGN KEY (project, issue_id) REFERENCES issue(project, id)
);

-- Most rows are terminal; schedulers scan only active rows
-- (cf. idx_task_run_active_status_id).
CREATE INDEX idx_review_run_active_status ON review_run(status)
    WHERE status IN ('AVAILABLE', 'RUNNING');

-- For the heartbeat reaper's dead-replica lookup
-- (cf. idx_task_run_running_replica).
CREATE INDEX idx_review_run_running_replica ON review_run(replica_id)
    WHERE status = 'RUNNING' AND replica_id IS NOT NULL;
