-- task_dag for task dependency

INSERT INTO
    task_dag (
        id,
        from_task_id,
        to_task_id
    )
VALUES
    (
        101,
        11018,
        11017
    );

INSERT INTO
    task_dag (
        id,
        from_task_id,
        to_task_id
    )
VALUES
    (
        102,
        11019,
        11018
    );

ALTER SEQUENCE task_dag_id_seq RESTART WITH 103;
