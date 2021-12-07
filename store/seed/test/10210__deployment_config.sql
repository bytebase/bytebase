-- Deployment config with regional deployments for project 3005.
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
        'regional',
        '{"deployments":[{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"IN","values":["us-central1","europe-west1"]}]}}},{"spec":{"selector":{"matchExpressions":[{"key":"location","operator":"EXISTS"}]}}}]}'
    );
