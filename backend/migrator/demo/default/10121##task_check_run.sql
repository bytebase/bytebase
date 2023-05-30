-- Task run for task 11004
INSERT INTO
    task_check_run (
        id,
        creator_id,
        updater_id,
        task_id,
        status,
        type,
        code,
        comment,
        result,
        payload
    )
VALUES
    (
        12101,
        1,
        1,
        11004,
        'FAILED',
        'bb.task-check.database.connect',
        101,
        'failed to connect database "shop": failed to connect database On-premises Staging MySQL/shop at "mysql.staging.example.com":"3306" with user "admin": dial tcp: lookup mysql.staging.example.com: no such host',
        '{}',
        '{}'
    );

INSERT INTO
    task_check_run (
        id,
        creator_id,
        updater_id,
        task_id,
        status,
        type,
        code,
        comment,
        result,
        payload
    )
VALUES
    (
        12102,
        1,
        1,
        11004,
        'DONE',
        'bb.task-check.database.statement.syntax',
        0,
        '',
        '{"resultList":[{"status":"SUCCESS","code":0,"title":"Syntax OK","content":"OK"}]}',
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","dbType":"MYSQL","charset":"utf8mb4","collation":"utf8mb4_general_ci"}'
    );

ALTER SEQUENCE task_check_run_id_seq RESTART WITH 12103;
