# Metadata Database Auth Refactor Design

## Context

Bytebase now supports AWS RDS IAM authentication for the metadata PostgreSQL database. The merged implementation lives directly in `backend/store/metadata_db_auth.go`, while metadata database connection creation lives in `backend/store/db_connection.go`.

Bytebase also supports cloud IAM authentication for managed instances, but those paths use `db.ConnectionConfig`, datasource protobuf fields, optional static credentials, optional AWS assume-role, and engine-specific connection setup. Metadata database auth is different: it is configured from startup `PG_URL`, uses runtime environment credentials, and runs during server bootstrap.

This refactor should improve metadata database auth organization without unifying it with managed-instance auth code. There are only two high-level call sites, and a shared cloud primitive layer would add coupling without enough payoff.

## Goals

- Move metadata database auth code out of the top-level `store` package into one focused package layer.
- Keep managed-instance cloud auth code unchanged.
- Introduce provider-neutral metadata database auth entrypoints now, with AWS as the only implementation in this refactor.
- Avoid leaking provider-specific runtime parameter names into `db_connection.go`.
- Preserve existing AWS RDS IAM behavior exactly.

## Non-Goals

- Do not add GCP or Azure metadata database auth in this refactor.
- Do not create shared cloud auth primitives under `backend/common`.
- Do not change Helm values or documented `PG_URL` parameters.
- Do not change managed-instance auth behavior for PostgreSQL, MySQL, OpenSearch, SQL Server, CosmosDB, or DynamoDB.
- Do not alter metadata database connection pooling, tracing, file watching, or reconnection behavior except for calling the new auth package.

## Package Layout

Create one new package:

```text
backend/store/dbauth/
  auth.go
  aws.go
  aws_test.go
```

The package is metadata-database-specific. Future providers should be added as sibling files:

```text
backend/store/dbauth/gcp.go
backend/store/dbauth/azure.go
```

This keeps the path shallow and avoids a nested `store/metadatadb/auth` package.

## Public API

`backend/store/dbauth` should expose:

```go
func Configure(ctx context.Context, pgxConfig *pgx.ConnConfig) ([]stdlib.OptionOpenDB, error)

func IsKeywordValueRuntimeParam(key string) bool
```

`Configure` owns metadata database auth setup. It may mutate `pgxConfig` and return `stdlib.OptionOpenDB` values. This is intentionally broader than a password-token abstraction because future providers may need different connection hooks:

- AWS uses `OptionBeforeConnect` to set a fresh RDS IAM token as the password.
- GCP Cloud SQL may need to set `pgxConfig.DialFunc`.
- Azure may need token/password injection or provider-specific connection behavior.

`IsKeywordValueRuntimeParam` lets `db_connection.go` classify keyword-value PostgreSQL DSNs without knowing provider-specific Bytebase auth parameter names.

## Data Flow

`backend/store/db_connection.go` should parse the PostgreSQL URL and delegate auth setup:

```go
pgxConfig, err := pgx.ParseConfig(pgURL)
if err != nil {
    return nil, errors.Wrap(err, "failed to parse database URL")
}

openOptions, err := dbauth.Configure(ctx, pgxConfig)
if err != nil {
    return nil, err
}

pgxConfig.Tracer = &metadataDBTracer{}
db := stdlib.OpenDB(*pgxConfig, openOptions...)
```

`isKeywordValuePGURL` should continue to recognize standard PostgreSQL keyword-value fields locally, but provider-specific Bytebase auth keys should be checked through `dbauth.IsKeywordValueRuntimeParam(key)`.

## AWS Provider Behavior

Move the current AWS implementation from `backend/store/metadata_db_auth.go` into `backend/store/dbauth/aws.go`.

The behavior must remain the same:

1. Detect `bytebase_aws_rds_iam=true`.
2. Read `bytebase_aws_region`.
3. Delete `bytebase_aws_rds_iam` and `bytebase_aws_region` from `pgxConfig.RuntimeParams`, regardless of whether IAM is enabled.
4. If AWS IAM is disabled, return no open options.
5. If enabled, require region, database user, host, and port.
6. Reject fallback hosts and TLS fallback.
7. Require verified TLS.
8. Load AWS default config with the configured region.
9. Return `stdlib.OptionBeforeConnect`, which signs a fresh auth token and assigns `connConfig.Password`.

The AWS token provider interface should remain internal to `dbauth` so tests can inject a fake token builder without exporting unnecessary surface area.

## Future Provider Direction

Future GCP and Azure metadata database auth should extend `dbauth.Configure`, not `backend/store/db_connection.go`.

The current AWS parameters remain supported for compatibility:

```text
bytebase_aws_rds_iam=true
bytebase_aws_region=us-east-1
```

If future provider parameters are added, their provider-specific names should be owned by `dbauth` and exposed to `db_connection.go` only through `IsKeywordValueRuntimeParam`.

This design does not require a provider registry. Providers are compile-time code, and a registry would be more abstraction than the current problem needs.

## Testing

Move the current metadata database auth tests into `backend/store/dbauth/aws_test.go`, adjusted for package `dbauth`.

Keep `backend/store/db_connection_test.go`, but make sure keyword-value DSN detection still covers a DSN with Bytebase auth params.

Test coverage should include:

- AWS disabled returns no open options and strips Bytebase AWS params.
- AWS URI-style DSN parses and configures endpoint, region, and user.
- AWS keyword-value DSN parses and configures endpoint, region, and user.
- Missing region, user, host, and port return errors.
- `sslmode=disable`, `sslmode=require`, and TLS fallback continue to return errors.
- Fallback host rejection remains.
- `BeforeConnect` sets the generated token as the password.
- Token generation errors are wrapped with endpoint, region, and user context.
- `Configure` returns one open option for AWS enabled and none for disabled.
- `isFilePath` still treats keyword-value DSNs with Bytebase auth params as DSNs, not file paths.

## Validation

After implementation, run:

```bash
gofmt -w backend/store/db_connection.go backend/store/db_connection_test.go backend/store/dbauth/*.go
go test -v -count=1 ./backend/store ./backend/store/dbauth
golangci-lint run --allow-parallel-runners
```

Because this is Go code, repeat `golangci-lint run --allow-parallel-runners` until it reports no issues.
