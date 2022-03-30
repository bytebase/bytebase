-- This is for demo purpose and the applciation_id has alraedy been deleted
INSERT INTO
    vcs (
        id,
        creator_id,
        updater_id,
        name,
        type,
        instance_url,
        api_url,
        application_id,
        secret
    )
VALUES
    (
        17001,
        101,
        101,
        'bytebase.gitlab.com',
        'GITLAB_SELF_HOST',
        'https://gitlab.bytebase.com',
        'https://gitlab.bytebase.com/api/v4',
        'fda62e44b5388b1ca6e72d5a7028a3c2c47157fc13fd98328e2bd446fae98fd8',
        '4ac5cf6f2e400398e34f753a5222432838e84349e8d5120c4adaa6a65278b765'
    );

ALTER SEQUENCE vcs_id_seq RESTART WITH 17002;
