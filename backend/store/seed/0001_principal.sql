-- End user id starts at 1001, we reserve the range between 1 ~ 1000 for internal use.
-- Setting the id explicitly changes the next id value to be +1
INSERT INTO
    principal (
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Demo Owner',
        'demo@example.com',
        -- 1024
        '$2a$10$/65QFlHOmDzXshEMt/qYuunbJrXtRLcaYDcRODbyOPa/9/N0N8Zc2'
    );

INSERT INTO
    principal (
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Jerry DBA',
        'jerry@example.com',
        -- aaa
        '$2a$10$a.o5.ELPUO8PKYGuWTSDseOqNssImU2b9qFgBaDKI7CKAKIhQYVfG'
    );

INSERT INTO
    principal (
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Tom Dev',
        'tom@example.com',
        -- aaa
        '$2a$10$cB0QuMqG0Bmz/j1LDI2gXOqUXtp.Yd87zRus6zxR026RyiyuWeJye'
    );

INSERT INTO
    principal (
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Alice DBA',
        'alice@example.com',
        -- aaa
        '$2a$10$pbkq/3m/AJ.rE8.6G/.Moez5Ld6dIaFKRtnsHNtYofHjupWWP/6YW'
    );

INSERT INTO
    principal (
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Jane Dev',
        'jane@example.com',
        -- aaa
        '$2a$10$DJ/T2SmdNiOAKXnuf.LQzenVYr4sIQSDu004Io1svmiRUmvAEMIw6'
    );