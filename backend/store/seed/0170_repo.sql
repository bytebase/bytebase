INSERT INTO
    repo (
        id,
        creator_id,
        updater_id,
        vcs_id,
        project_id,
        name,
        full_path,
        web_url,
        external_id,
        webhook_id
    )
VALUES
    (
        17001,
        1001,
        1001,
        16001,
        3001,
        "project1",
        "bytebase-test/project1",
        "http://gitlab.bytebase.com/bytebase-test/project1",
        -- Refers to the bytebase-test/project1
        "7",
        -- Refers to the webhook in bytebase-test/project1
        "5"
    );