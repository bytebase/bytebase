INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6001,
        101,
        101,
        5001,
        'RDS dev',
        'MYSQL',
        'bytebase-demo-dev-mysql8.cerr43rttews.us-west-1.rds.amazonaws.com',
        '3306',
        'https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=bytebase-demo-dev-mysql8;is-cluster=false'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6002,
        101,
        101,
        5002,
        'RDS integration',
        'MYSQL',
        'bytebase-demo-integration-mysql8.cerr43rttews.us-west-1.rds.amazonaws.com',
        '3306',
        'https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=bytebase-demo-integration-mysql8;is-cluster=false'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6003,
        101,
        101,
        5003,
        'RDS staging',
        'MYSQL',
        'bytebase-demo-staging-mysql57.cerr43rttews.us-west-1.rds.amazonaws.com',
        '3306',
        'https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=bytebase-demo-staging-mysql57;is-cluster=false'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6004,
        101,
        101,
        5004,
        'RDS prod',
        'MYSQL',
        'bytebase-demo-prod-mysql57.cerr43rttews.us-west-1.rds.amazonaws.com',
        '3306',
        'https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=bytebase-demo-prod-mysql57;is-cluster=false'
    );