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
        '{"deployments":[{"name":"Dev Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Dev"]},{"key":"bb.location","operator":"In","values":["earth"]}]}}},{"name":"Integration Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Integration"]},{"key":"bb.tenant","operator":"Exists"}]}}}]}'
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
        '{"deployments":[{"name":"Dev Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Dev"]},{"key":"bb.location","operator":"In","values":["earth"]}]}}},{"name":"Integration Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Integration"]},{"key":"bb.tenant","operator":"Exists"}]}}}]}'
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
        21003,
        101,
        101,
        3007,
        'db-pattern',
        '{"deployments":[{"name":"Dev Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Dev"]},{"key":"bb.tenant","operator":"In","values":["dev"]}]}}},{"name":"PTR Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Staging"]},{"key":"bb.tenant","operator":"Exists"}]}}},{"name":"Release Stage","spec":{"selector":{"matchExpressions":[{"key":"bb.environment","operator":"In","values":["Prod"]},{"key":"bb.tenant","operator":"Exists"}]}}}]}'
    );

ALTER SEQUENCE deployment_config_id_seq RESTART WITH 21004;
