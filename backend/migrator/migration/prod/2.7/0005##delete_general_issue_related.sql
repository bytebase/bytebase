-- task_run
DELETE FROM
    task_run
WHERE
    task_run.task_id IN (
        SELECT
            DISTINCT(task.id)
        FROM
            task
        WHERE
            type = 'bb.task.general'
    );

-- task
DELETE FROM
    task
WHERE
    type = 'bb.task.general';

-- stage
DELETE FROM
    stage
WHERE
    stage.pipeline_id IN (
        SELECT
            DISTINCT(pipeline.id)
        FROM
            pipeline
            JOIN issue ON issue.pipeline_id = pipeline.id
            AND issue.type = 'bb.issue.general'
    );

-- issue
DELETE FROM
    issue
WHERE
    issue.type = 'bb.issue.general';

-- pipeline
DELETE FROM
    pipeline
WHERE
    (
        SELECT
            COUNT(1)
        FROM
            stage
        where
            stage.pipeline_id = pipeline.id
    ) = 0;