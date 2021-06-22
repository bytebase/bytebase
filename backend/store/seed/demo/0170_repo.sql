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
        external_webhook_id,
        webhook_endpoint_id,
        secret_token
    )
VALUES
    (
        17001,
        101,
        101,
        16001,
        3001,
        "project1",
        "bytebase-test/project1",
        "http://gitlab.bytebase.com/bytebase-test/project1",
        "bytebase",
        "master",
        -- Refers to the bytebase-test/project1
        "7",
        -- Refers to the webhook in bytebase-test/project1
        "5",
        "8368dde4-e352-4696-9fef-26bd3af6ec40",
        "VFN2lgKDRLWjJ25B"
    );