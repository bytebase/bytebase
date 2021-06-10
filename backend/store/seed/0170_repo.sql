INSERT INTO
    repo (
        id,
        creator_id,
        updater_id,
        vcs_id,
        project_id,
        external_id,
        webhook_id,
        name,
        full_path,
        web_url
    )
VALUES
    (
        17001,
        1001,
        1001,
        16001,
        3001,
        -- Refers to the bytebase-test/project1
        "7",
        -- Refers to the webhook in bytebase-test/project1
        "5",
        "project1",
        "bytebase-test/project1",
        "http://gitlab.bytebase.com/bytebase-test/project1"
    );