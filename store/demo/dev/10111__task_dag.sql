-- task_dag for task dependency

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

INSERT INTO
    task_dag (
        id,
        from_task_id,
        to_task_id
    )
VALUES
    (
        11102,
        11018,
        11019
    );

ALTER SEQUENCE task_dag_id_seq RESTART WITH 11103;
