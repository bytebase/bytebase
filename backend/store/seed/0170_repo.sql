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
        base_directory,
        branch_filter,
        external_id,
        webhook_id,
        webhook_url
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
        "bytebase",
        "feature/*",
        -- Refers to the bytebase-test/project1
        "7",
        -- Refers to the webhook in bytebase-test/project1
        "5",
        "http://gitlab.bytebase.com/hook/gitlab/8368dde4-e352-4696-9fef-26bd3af6ec40"
    );