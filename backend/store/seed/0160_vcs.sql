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
        access_token_expiration_ts,
        refresh_token
    )
VALUES
    (
        16001,
        1001,
        1001,
        'GitLab Bytebase',
        'GITLAB_SELF_HOST',
        'http://gitlab.bytebase.com',
        'http://gitlab.bytebase.com/api/v4',
        '0d1f706e68e3bf22be6712752726a3eec2a9684ec098a9695015f484165ed922',
        '77deb508304dd73fa9b6868d319e1832889c6fc8ea087f400e6efe35d7c6dd87',
        '11f2e3a493cab3bafe56a5d8371ff1bd256c2a7386b005688c422313dcd215bf',
        0,
        'bba8a841ab2b60ba172ee0161152dd0b1f721726bbed801b5d91b1ec903a9e75'
    );