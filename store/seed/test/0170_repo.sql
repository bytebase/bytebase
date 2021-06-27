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
        "Bytebase Blog",
        "bytebase-demo/bbblog",
        "http://gitlab.bytebase.com/bytebase-demo/bbblog",
        "bytebase",
        "master",
        -- Refers to the bytebase-demo/bbblog
        "13",
        -- Refers to the webhook in bytebase-demo/bbblog
        "59",
        "5effe17f-947a-4eff-b0b5-244e26c5cd68",
        "mQqrGRRnAuuL59F8"
    );