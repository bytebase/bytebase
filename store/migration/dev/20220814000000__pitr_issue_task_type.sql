UPDATE
    issue
SET
    type = "bb.issue.database.restore.pitr"
WHERE
    issue.id IN (
        SELECT
            issue.id
        FROM
            issue
            JOIN task ON issue.pipeline_id = task.pipeline_id
        WHERE
            task.type = "bb.task.database.restore"
    );

UPDATE task SET type = "bb.task.database.restore.pitr.restore" WHERE type = "bb.task.database.pitr.restore";
UPDATE task SET type = "bb.task.database.resotre.pitr.cutover" WHERE type = "bb.task.database.pitr.cutover";
