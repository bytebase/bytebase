-- task_dag for gh-ost
INSERT INTO
    task_dag (
        id,
        from_task_id,
        to_task_id
    )
VALUES
    (
        11101,
        11017,
        11018
    );

-- task_dag for PITR tasks
INSERT INTO
    task_dag (
        id,
        from_task_id,
        to_task_id
    )
VALUES
    (
        11102,
        11019,
        11020
    );

ALTER SEQUENCE task_dag_id_seq RESTART WITH 11103;
