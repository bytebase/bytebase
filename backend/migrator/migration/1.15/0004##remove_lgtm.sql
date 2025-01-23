DELETE FROM task_check_run
WHERE type = 'bb.task-check.issue.lgtm';

ALTER TABLE project DROP COLUMN lgtm_check;
