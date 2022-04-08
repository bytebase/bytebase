# Schema Update Guide

This document goes over the scheme update process during development and production release.

All schemas are located at [store/migration](https://github.com/bytebase/bytebase/tree/main/store/migration). Bytebase schema release follows semantic versioning.

Prod

- [prod](https://github.com/bytebase/bytebase/tree/main/store/migration/prod) schema.
- `LATEST.sql` and `LATEST_DATA.sql` will be the schema and data to initialize a new database. 
- The version directory such as `1.1` contains all the DDL statements used to migrate from previous version of schema such as `1.0`. When a prod release binary starts, it will apply all the schema migrations from current version up to the latest schema release version.

Dev

- [dev](https://github.com/bytebase/bytebase/tree/main/store/migration/dev) schema.
- `LATEST.sql` and `LATEST_DATA.sql` will only be used to view the latest schema.
- When a dev release binary starts, it will use the LATEST.sql in release directory to initialize the database and then apply all the DDLs in the dev directory if not yet.

Demo Data

- When the binary starts in Demo mode, we will seed demo data to provide an overview of the product.
- [Dev demo data](https://github.com/bytebase/bytebase/tree/main/store/demo/dev).
- [Prod demo data](https://github.com/bytebase/bytebase/tree/main/store/demo/prod).

## Development

1. Add a DDL SQL file such as `dev/20220408000000__schema_version_type.sql`. The prefix numbers should be the date time format when the PR is sent.
2. Update `LATEST.sql` and `LATEST_DATA.sql` w.r.t. the DDL changes. (TODO: auto-generate the latest schema)
3. Update Dev Demo data if needed.
4. Since we use the same code for both dev and release schemas, we should add if-else branching to read storage differently based on schema version, such as [this example](https://github.com/bytebase/bytebase/pull/1039).

## Release
Releaser should take the following steps for schema update release, at most once a month. A DDL file should only be moved forward to release only if the feature is completed and well tested.

1. Create a new version directory in the release directory, such as `release/1.6`.
1. Move DDLs to be released from `dev` to `release/1.6`. Rename DDL SQL file prefixes in sure the prefixes are in consecutively increasing order starting from `0000`.
2. Update `prod/LATEST.sql` and `prod/LATEST_DATA.sql` w.r.t. DDLs to be released. (TODO: auto-generate the latest schema)
3. Copy over Dev Demo data to Prod Demo directory for the changes to be released.
