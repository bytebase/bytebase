-- End user id starts at 101, we reserve the range between 1 ~ 100 for internal use.
-- Setting the id explicitly changes the next id value to be +1
-- 101
INSERT INTO
    principal (
        creator_id,
        updater_id,
        type,
        auth_provider,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'END_USER',
        'BYTEBASE',
        'Demo Owner',
        'demo@example.com',
        -- 1024
        '$2a$10$/65QFlHOmDzXshEMt/qYuunbJrXtRLcaYDcRODbyOPa/9/N0N8Zc2'
    );

-- 102
INSERT INTO
    principal (
        creator_id,
        updater_id,
        type,
        auth_provider,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'END_USER',
        'BYTEBASE',
        'Jerry DBA',
        'jerry@example.com',
        -- 2048
        '$2a$10$Q2NJib9bRvDkap1N1RDP2O3HyxjCldwfvGoxAzZL5gbbBgKAFD4cq'
    );

-- 103
INSERT INTO
    principal (
        creator_id,
        updater_id,
        type,
        auth_provider,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'END_USER',
        'BYTEBASE',
        'Tom Dev',
        'tom@example.com',
        -- 4096
        '$2a$10$X5bvIWk4BKhEaZqlNLGjgOUB09i97olKBfjTQT49zMtNGnhoy6GIW'
    );

-- 104
INSERT INTO
    principal (
        creator_id,
        updater_id,
        type,
        auth_provider,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        'END_USER',
        'BYTEBASE',
        'Jane Dev',
        'jane@example.com',
        -- 8192
        '$2a$10$2QTgsuKDTGYe68lpeqRqouw1dQTRhssrYSnwQVzQUctQvhnVIccRa'
    );
