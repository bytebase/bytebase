-- Pseudo allUsers account id is 2.
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        type,
        name,
        email,
        password_hash
    )
VALUES
    (
        2,
        2,
        2,
        'SYSTEM_BOT',
        'All Users',
        'allUsers',
        ''
    );