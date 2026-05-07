# AWS RDS IAM Auth for Metadata Database

Status: Draft for peer review

Issue: [BYT-9073](https://linear.app/bytebase/issue/BYT-9073/aws-rds-iam-auth-for-bytebase-metadata-database)

Date: 2026-04-29

## Background

BYT-9073 asks Bytebase to support AWS RDS IAM authentication for Bytebase's own metadata PostgreSQL database. The issue has no additional requirements or comments, so this design scopes the work to self-hosted Bytebase deployments that use external PostgreSQL as metadata storage.

The current metadata database connection path is `backend/store/db_connection.go`. `DBConnectionManager` accepts `PG_URL` or a file path containing `PG_URL`, parses the final value with `pgx.ParseConfig`, opens a `database/sql` pool through pgx stdlib, pings it, and configures max open connections from PostgreSQL server settings.

Bytebase already supports AWS RDS IAM auth for user-managed PostgreSQL instances in `backend/plugin/db/pg/pg.go`. That path uses `github.com/aws/aws-sdk-go-v2/feature/rds/auth.BuildAuthToken` plus existing AWS credential helpers. The metadata database path cannot directly reuse that datasource flow today because it receives only `PG_URL`, not a stored datasource protobuf.

AWS RDS IAM tokens are generated with AWS Signature Version 4, are valid for 15 minutes, and are used to authenticate a new database session. Existing database sessions are not invalidated when the token expires. For PostgreSQL, when a user has the `rds_iam` role, IAM authentication takes precedence over password authentication.

Standard implementations in other products follow the same general model: AWS credentials are configured separately from the PostgreSQL password, and the driver or product generates an RDS IAM token when opening a database connection. The AWS Advanced JDBC Wrapper exposes this as an IAM authentication plugin. Tools such as StrongDM, DBeaver, and DataGrip model AWS credentials separately from the PostgreSQL connection target.

## Goals

- Allow Bytebase to run against an AWS RDS or Aurora PostgreSQL metadata database without storing a static database password.
- Generate a valid IAM auth token for every new physical metadata database connection, including connections opened after the process has been running longer than 15 minutes.
- Keep existing `PG_URL` deployments backward compatible, including password-based `PG_URL`, file-based `PG_URL` rotation, and embedded local PostgreSQL.
- Use the AWS SDK default credential chain so EC2 instance profiles, ECS task roles, EKS IRSA, environment variables, shared config, and AWS profiles work according to standard SDK behavior.
- Keep the implementation isolated to metadata database connection setup, Helm configuration, and deployment documentation.

## Non-goals

- Do not add a UI workflow. This is a server startup and operator configuration feature.
- Do not provision AWS resources, RDS users, IAM policies, database grants, RDS Proxy, or Secrets Manager resources.
- Do not store AWS access keys in Bytebase-specific `PG_URL` parameters.
- Do not require RDS Proxy or Secrets Manager for the primary path.
- Do not change Bytebase's managed database instance connection model.

## Design

Implement native metadata database IAM authentication in `backend/store`. Bytebase should generate an AWS RDS auth token whenever the metadata connection pool opens a new physical PostgreSQL connection. This keeps the operator-facing `PG_URL` model, avoids storing a static database password, and matches the way AWS-supported drivers and database tools handle RDS IAM auth.

Operators enable IAM auth through reserved Bytebase parameters in `PG_URL`:

```text
postgres://bb_meta@mydb.abc.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1
```

The initial version supports these parameters:

- `bytebase_aws_rds_iam=true`: enables metadata database IAM auth.
- `bytebase_aws_region=<region>`: sets the AWS region used to sign the RDS auth token.

`bytebase_aws_region` is required in v1. Region derivation from RDS endpoint hostnames can be considered later, but requiring the region avoids surprising behavior with custom endpoints, Aurora endpoints, and future proxy setups.

Bytebase reads the `bytebase_aws_*` parameters from pgx's parsed runtime parameters and deletes them before opening the database, so they are not sent as PostgreSQL startup parameters. When IAM auth is enabled, Bytebase must require verified TLS so the generated auth token is sent only over a verified connection. When `bytebase_aws_rds_iam` is absent or false, existing behavior is unchanged.

The existing `PG_URL` file watcher remains available as a generic credential rotation mechanism, but it should not be the recommended RDS IAM path. An external token refresher pushes correctness to another process and swaps the whole `sql.DB` pool whenever the file changes. Native per-connection token generation is simpler for operators and better aligned with database pooling.

For the Helm chart, add optional values that render the same `PG_URL` shape without a password:

```yaml
bytebase:
  option:
    externalPg:
      awsRdsIam:
        enabled: true
        region: us-east-1
```

When `awsRdsIam.enabled` is true, the chart should render `PG_URL` with the username, host, port, database, `bytebase_aws_rds_iam=true`, and `bytebase_aws_region=<region>`, but without a password. The existing `serviceAccount.annotations` value is sufficient for EKS IRSA. Users of `existingPgURLSecret` can include the Bytebase query parameters directly in the secret.

The security model is the standard RDS IAM model:

- The PostgreSQL user must exist and be granted `rds_iam`.
- The AWS principal used by the Bytebase process must have `rds-db:connect` permission for the database resource and user.
- IAM auth requires verified TLS, such as `sslmode=verify-full`, with an AWS RDS CA bundle or platform trust store that validates the RDS certificate.
- No generated token should be persisted to disk, exported to logs, stored in settings, or exposed through metrics labels.

## Implementation design

Add a small private metadata database auth layer in `backend/store`.

Define a metadata auth extractor that runs after `pgx.ParseConfig`:

- it reads `bytebase_aws_rds_iam` and `bytebase_aws_region` from `pgxConfig.RuntimeParams`
- it deletes those Bytebase parameters from `RuntimeParams` so they are not sent to PostgreSQL
- it returns a metadata auth config used by Bytebase before opening the pool

Because pgx handles the connection string parsing after Bytebase classifies the input as a direct connection string, IAM auth works with URI-style values and compact keyword/value `PG_URL` values such as `host=... port=...`. Bytebase should reject pgx fallback configs while IAM is enabled because the generated token is scoped to one host:port endpoint.

The metadata auth config should contain:

- whether IAM auth is enabled
- AWS region
- host and port used to build the token endpoint
- database username

Define a token provider interface so tests can inject a fake token builder. The production provider loads AWS config once during metadata DB pool creation with `config.LoadDefaultConfig(ctx, config.WithRegion(region))`, stores the resulting credential provider, and calls `auth.BuildAuthToken(ctx, hostPort, region, user, credentials)` from the `BeforeConnect` hook.

In `createConnectionWithTracer`, call `pgx.ParseConfig`, extract and validate the metadata auth config, then open the pool with:

```go
stdlib.OpenDB(*pgxConfig, stdlib.OptionBeforeConnect(...))
```

The `BeforeConnect` hook should generate a token, assign it to `connConfig.Password`, and return any generation error. This makes token generation happen exactly when pgx opens a new physical connection.

Keep the existing `metadataDBTracer` behavior, startup ping, and `max_connections` probing. These operations will use the hook during initial connection creation.

Do not set `ConnMaxLifetime` to 15 minutes. Token expiration does not terminate established sessions, and forced churn would increase token generation pressure without improving correctness.

Startup should fail with a clear message when IAM auth is enabled but region, verified TLS, host, port, or user is missing, or when pgx parsed fallback configs are present. Token-generation errors should include enough context to identify the endpoint, region, and user, but must not include the generated token or a full secret-bearing `PG_URL`. If PostgreSQL rejects the token, Bytebase should surface the database ping or connection error normally.

Testing should cover:

- `PG_URL` parsing and Bytebase query-parameter stripping
- non-IAM `PG_URL` compatibility
- required field validation for region, verified TLS, user, host, port, and unsupported fallback configs
- `BeforeConnect` behavior with a fake token provider
- token-generation error propagation
- Helm rendering when `awsRdsIam.enabled` is true
- Helm rendering compatibility when IAM auth is disabled

A manual integration test should create an RDS or Aurora PostgreSQL user, grant `rds_iam`, attach an IAM role with `rds-db:connect`, start Bytebase with the IAM `PG_URL`, confirm migrations run, leave the process running past 15 minutes, and force new pool connections to verify fresh token generation.

## Appendix

### Alternatives considered

Native pgx `BeforeConnect` hook is the recommended approach. The token is generated exactly when a new physical connection is created. It fits `database/sql` pooling, avoids whole-pool swaps, and keeps the auth behavior local to the metadata connection layer.

External token generator plus `PG_URL` file watcher can work with the existing file-watcher behavior, but it pushes correctness to an external process. It also swaps the whole `sql.DB` pool whenever the file changes, which is heavier than refreshing auth only for new connections.

Separate typed metadata database config fields may be useful if metadata database configuration grows beyond `PG_URL`, but it is larger than needed for v1. Bytebase already centers metadata database setup on `PG_URL`, and reserved query parameters keep the initial surface small.

RDS Proxy and Secrets Manager can solve adjacent operational problems, but they either still need client-side IAM auth or introduce a different credential-management system. They should remain compatible, not required.

### Rollout and compatibility

This change is additive. Existing `PG_URL` deployments continue using static password auth or file-based rotation. No database migration is required. No frontend change is required. Helm changes should be backward compatible and only alter `PG_URL` rendering when `awsRdsIam.enabled` is true.

### Open questions for review

- Should v1 support explicit assume-role parameters for the metadata database, or should operators rely on the AWS SDK default chain and platform role configuration first?
- Should a later version derive `bytebase_aws_region` from standard RDS endpoint hostnames and allow the parameter as an override?
- Should RDS Proxy with IAM client authentication be listed as supported in v1, or treated as best-effort until tested?

### Source links

- [AWS RDS IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html)
- [AWS SDK for Go v2 RDS utilities](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/sdk-utilities-rds.html)
- [AWS Advanced JDBC IAM authentication plugin](https://github.com/aws/aws-advanced-jdbc-wrapper/blob/main/docs/using-the-jdbc-driver/using-plugins/UsingTheIamAuthenticationPlugin.md)
- [StrongDM RDS PostgreSQL IAM datasource](https://docs.strongdm.com/admin/resources/datasources/rds-postgresql-iam)
- [DBeaver AWS credentials](https://dbeaver.com/docs/dbeaver/AWS-Credentials/)
- [DataGrip AWS cloud connections](https://www.jetbrains.com/help/datagrip/clouds-aws.html)
