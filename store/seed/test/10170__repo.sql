-- This is for demo purpose and the webhook has already been deleted
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
        webhook_url_host,
        webhook_endpoint_id,
        webhook_secret_token,
        access_token,
        expires_ts,
        refresh_token
    )
VALUES
    (
        17001,
        101,
        101,
        16001,
        3003,
        "Blog",
        "bytebase-demo/blog",
        "http://gitlab.bytebase.com/bytebase-demo/blog",
        "bytebase",
        "master",
        -- Refers to the bytebase-demo/blog
        "13",
        -- Refers to the webhook in bytebase-demo/blog
        "60",
        "https://demo.bytebase.com",
        "e99bf622-7f58-4d6b-a5be-b97af313d7ea",
        "xxxxxxxxxxxxxxxx",
        "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        0,
        "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    );

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
        webhook_url_host,
        webhook_endpoint_id,
        webhook_secret_token,
        access_token,
        expires_ts,
        refresh_token
    )
VALUES
    (
        17002,
        101,
        101,
        16001,
        3002,
        "Shop",
        "bytebase-demo/shop",
        "http://gitlab.bytebase.com/bytebase-demo/shop",
        "bytebase",
        "master",
        -- Refers to the bytebase-demo/shop
        "14",
        -- Refers to the webhook in bytebase-demo/shop
        "61",
        "https://demo.bytebase.com",
        "c5e30130-7322-4f84-953a-d08168c047d1",
        "xxxxxxxxxxxxxxxx",
        "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        0,
        "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    );