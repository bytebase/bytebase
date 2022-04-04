# Schema Update Guide

All schemas are located at [store/migration](https://github.com/bytebase/bytebase/tree/main/store/migration). Bytebase schema release follows semantic versioning. This document goes over the scheme update process during development and release.

We use different schemas for [dev](https://github.com/bytebase/bytebase/tree/main/store/migration/dev) and [release](https://github.com/bytebase/bytebase/tree/main/store/migration/release). The `LATEST.sql` and `LATEST_DATA.sql` will be the schema and data to initialize a new database. The version directory such as `1.1` contains all the DDL statements used to migrate from previous version of schema such as `1.0`. There should only one version in dev schema directory.

## Development

Let's take an example that our release is at version `1.5` and dev at version `1.6`.

If there is `1.6/0000__placeholder.sql` file in the dev version directory,

1. rename the placeholder file to something meaningful related to feature such as `1.6/0000__roofshot.sql` and fill in the DDL.

Otherwise,

1. Add a new SQL file for the DDL such as `1.6/0001__moonshot.sql`. The prefix numbers should be in consecutively increasing order.

Other than the DDL change, we should also

1. Update `LATEST.sql` and `LATEST_DATA.sql` w.r.t. the DDL changes. (TODO: auto-generate the latest schema)
1. Update [dev demo data](https://github.com/bytebase/bytebase/tree/main/store/demo/dev) if needed.
1. Since we use the same code for both dev and release schemas, we should add if-else branching to read storage differently based on schema version. (TODO: add example later)

## Release
Releaser should take the following steps for schema update release, at most once a month. A DDL file should only be moved forward to release only if the feature is completed and well tested.

1. Create a new version directory in release, such as `release/1.6`.
1. Move DDLs to be released from `dev/1.6` to `release/1.6`. Rename DDL SQL file prefixes in sure the prefixes are in consecutively increasing order starting from `0000`.
1. Rename directory `dev/1.6` to `dev/1.7`. If it's empty, add a placeholder SQL file `dev/1.7/0000__placeholder.sql`. Rename DDL SQL file prefixes in sure the prefixes are in consecutively increasing order starting from `0000`.
1. Update `release/LATEST.sql` and `release/LATEST_DATA.sql` w.r.t. DDLs to be released. This is a [PR example](https://github.com/bytebase/bytebase/pull/1011) so far. (TODO: auto-generate the latest schema)
1. Copy over dev demo data to release demo directory for the changes to be released.
