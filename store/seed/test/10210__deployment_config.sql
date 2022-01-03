-- Deployment config with regional deployments for tenant mode project.
INSERT INTO
    deployment_config (
        id,
        creator_id,
        updater_id,
        project_id,
        name,
        config
    )
VALUES
    (
        21001,
        101,
        101,
        3005,
        'regional-1',
        '{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Dev"]},{"key":"bb.location","operator":"In","values":["earth"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Integration"]},{"key":"bb.tenant","operator":"Exists"}]}}}]}'
    );

INSERT INTO
    deployment_config (
        id,
        creator_id,
        updater_id,
        project_id,
        name,
        config
    )
VALUES
    (
        21002,
        101,
        101,
        3006,
        'regional-2',
        '{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Dev"]},{"key":"bb.location","operator":"In","values":["earth"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Integration"]},{"key":"bb.tenant","operator":"Exists"}]}}}]}'
    );
