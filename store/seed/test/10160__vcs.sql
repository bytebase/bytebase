-- This is for demo purpose and the applciation_id has alraedy been deleted
INSERT INTO
    vcs (
        id,
        creator_id,
        updater_id,
        name,
        `type`,
        instance_url,
        api_url,
        application_id,
        secret,
        access_token,
        expires_ts,
        refresh_token
    )
VALUES
    (
        16001,
        101,
        101,
        'bytebase.gitlab.com',
        'GITLAB_SELF_HOST',
        'https://gitlab.bytebase.com',
        'https://gitlab.bytebase.com/api/v4',
        '0d1f706e68e3bf22be6712752726a3eec2a9684ec098a9695015f484165ed922',
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        0,
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    );