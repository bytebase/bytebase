ALTER TABLE principal ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE principal SET created_at = to_timestamp(created_ts);
ALTER TABLE principal DROP COLUMN created_ts;

ALTER TABLE db ADD COLUMN sync_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE db SET sync_at = to_timestamp(last_successful_sync_ts) WHERE last_successful_sync_ts > 0;
ALTER TABLE db DROP COLUMN last_successful_sync_ts;

ALTER TABLE sheet ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE sheet SET created_at = to_timestamp(created_ts);
ALTER TABLE sheet DROP COLUMN created_ts;

ALTER TABLE pipeline ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE pipeline SET created_at = to_timestamp(created_ts);
ALTER TABLE pipeline DROP COLUMN created_ts;

ALTER TABLE task ADD COLUMN earliest_allowed_at TIMESTAMPTZ NULL;
UPDATE task SET earliest_allowed_at = to_timestamp(earliest_allowed_ts) WHERE earliest_allowed_ts > 0;
ALTER TABLE task DROP COLUMN earliest_allowed_ts;

ALTER TABLE task_run ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE task_run SET created_at = to_timestamp(created_ts);
ALTER TABLE task_run DROP COLUMN created_ts;

ALTER TABLE task_run ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE task_run SET updated_at = to_timestamp(updated_ts);
ALTER TABLE task_run DROP COLUMN updated_ts;

ALTER TABLE task_run ADD COLUMN started_at TIMESTAMPTZ NULL;
UPDATE task_run SET started_at = to_timestamp(started_ts) WHERE started_ts > 0;
ALTER TABLE task_run DROP COLUMN started_ts;

ALTER TABLE plan ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE plan SET created_at = to_timestamp(created_ts);
ALTER TABLE plan DROP COLUMN created_ts;

ALTER TABLE plan ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE plan SET updated_at = to_timestamp(updated_ts);
ALTER TABLE plan DROP COLUMN updated_ts;

ALTER TABLE plan_check_run ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE plan_check_run SET created_at = to_timestamp(created_ts);
ALTER TABLE plan_check_run DROP COLUMN created_ts;

ALTER TABLE plan_check_run ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE plan_check_run SET updated_at = to_timestamp(updated_ts);
ALTER TABLE plan_check_run DROP COLUMN updated_ts;

ALTER TABLE issue ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE issue SET created_at = to_timestamp(created_ts);
ALTER TABLE issue DROP COLUMN created_ts;

ALTER TABLE issue ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE issue SET updated_at = to_timestamp(updated_ts);
ALTER TABLE issue DROP COLUMN updated_ts;

ALTER TABLE audit_log ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE audit_log SET created_at = to_timestamp(created_ts);
ALTER TABLE audit_log DROP COLUMN created_ts;

ALTER TABLE issue_comment ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE issue_comment SET created_at = to_timestamp(created_ts);
ALTER TABLE issue_comment DROP COLUMN created_ts;

ALTER TABLE issue_comment ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE issue_comment SET updated_at = to_timestamp(updated_ts);
ALTER TABLE issue_comment DROP COLUMN updated_ts;

ALTER TABLE query_history ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE query_history SET created_at = to_timestamp(created_ts);
ALTER TABLE query_history DROP COLUMN created_ts;

ALTER TABLE anomaly ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE anomaly SET updated_at = to_timestamp(updated_ts);
ALTER TABLE anomaly DROP COLUMN updated_ts;

ALTER TABLE worksheet ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE worksheet SET created_at = to_timestamp(created_ts);
ALTER TABLE worksheet DROP COLUMN created_ts;

ALTER TABLE worksheet ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE worksheet SET updated_at = to_timestamp(updated_ts);
ALTER TABLE worksheet DROP COLUMN updated_ts;

ALTER TABLE changelist ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE changelist SET updated_at = to_timestamp(updated_ts);
ALTER TABLE changelist DROP COLUMN updated_ts;

ALTER TABLE export_archive ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE export_archive SET created_at = to_timestamp(created_ts);
ALTER TABLE export_archive DROP COLUMN created_ts;
