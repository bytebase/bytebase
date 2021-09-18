-- Task run for task 11004
INSERT INTO
    task_check_run (
        creator_id,
        updater_id,
        task_id,
        `status`,
        `type`,
        comment,
        result,
        payload
    )
VALUES
    (
        1,
        1,
        11004,
        'FAILED',
        'bb.task-check.database.connect',
        'failed to connect database "shop": failed to connect database On-premises Staging MySQL/shop at "mysql.staging.example.com":"3306" with user "admin": dial tcp: lookup mysql.staging.example.com: no such host',
        '',
        ''
    );

INSERT INTO
    task_check_run (
        creator_id,
        updater_id,
        task_id,
        `status`,
        `type`,
        comment,
        result,
        payload
    )
VALUES
    (
        1,
        1,
        11004,
        'DONE',
        'bb.task-check.database.statement.syntax',
        '',
        '{"resultList":[{"status":"SUCCESS","title":"Syntax OK","content":"OK"}]}',
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","dbType":"MYSQL","charset":"utf8mb4","collation":"utf8mb4_general_ci"}'
    );