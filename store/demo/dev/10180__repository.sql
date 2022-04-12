-- This is for demo purpose and the webhook has already been deleted
INSERT INTO
    repository (
        id,
        creator_id,
        updater_id,
        vcs_id,
        project_id,
        name,
        full_path,
        web_url,
        branch_filter,
        base_directory,
        file_path_template,
        schema_path_template,
        sheet_path_template,
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
        18001,
        101,
        101,
        17001,
        3003,
        'Blog',
        'bytebase-demo/blog',
        'http://gitlab.bytebase.com/bytebase-demo/blog',
        'master',
        'bytebase',
        '{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql',
        '{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql',
        'sheet/{{NAME}}.sql',
        -- Refers to the bytebase-demo/blog
        '13',
        -- Refers to the webhook in bytebase-demo/blog
        '60',
        'https://demo.bytebase.com',
        'e99bf622-7f58-4d6b-a5be-b97af313d7ea',
        'xxxxxxxxxxxxxxxx',
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        0,
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    );

INSERT INTO
    repository (
        id,
        creator_id,
        updater_id,
        vcs_id,
        project_id,
        name,
        full_path,
        web_url,
        branch_filter,
        base_directory,
        file_path_template,
        schema_path_template,
        sheet_path_template,
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
        18002,
        101,
        101,
        17001,
        3002,
        'Shop',
        'bytebase-demo/shop',
        'http://gitlab.bytebase.com/bytebase-demo/shop',
        'master',
        'bytebase',
        '{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql',
        '{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql',
        'sheet/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql',
        -- Refers to the bytebase-demo/shop
        '14',
        -- Refers to the webhook in bytebase-demo/shop
        '61',
        'https://demo.bytebase.com',
        'c5e30130-7322-4f84-953a-d08168c047d1',
        'xxxxxxxxxxxxxxxx',
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        0,
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    );

-- This is for demo purpose and the webhook has already been deleted
INSERT INTO
    repository (
        id,
        creator_id,
        updater_id,
        vcs_id,
        project_id,
        name,
        full_path,
        web_url,
        branch_filter,
        base_directory,
        file_path_template,
        schema_path_template,
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
        18003,
        101,
        101,
        17001,
        3005,
        'Tenant',
        'bytebase-demo/tenant',
        'http://gitlab.bytebase.com/bytebase-demo/tenant',
        'master',
        'bytebase',
        '{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql',
        '.{{DB_NAME}}__LATEST.sql',
        -- Refers to the bytebase-demo/tenant
        '15',
        -- Refers to the webhook in bytebase-demo/tenant
        '62',
        'https://demo.bytebase.com',
        'e48bf625-7f58-4d6b-a5be-b97af313d7ea',
        'xxxxxxxxxxxxxxxx',
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        0,
        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    );

ALTER SEQUENCE repository_id_seq RESTART WITH 18004;
