# Bytebase Schema Update Playbook

All schemas are located at [store/migration](https://github.com/bytebase/bytebase/tree/main/store/migration). Bytebase schema release follows semantic versioning. This document goes over the scheme update process during development and release.

## Development
We use different schema versions for [dev](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_dev.go) and [release](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_release.go). 

If schema versions in dev/release profiles are the same,
1. Create a new directory under [migration directory](https://github.com/bytebase/bytebase/tree/main/store/migration) with the new version, e.g. 1.2.0.
2. Copy over the [latest.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/latest.sql) schema file and [latest_data.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/latest_data.sql) data file from the previous version. And make a PR for changes above.
3. Bump up the MINOR version in the [dev schema version](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_dev.go), e.g. 1.1.0 will be 1.2.0.
4. Add a new SQL file with DDL statements with a descriptive name, e.g. [sheet_vcs.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/sheet_vcs.sql).
5. Update latest.sql and latest_data.sql data file w.r.t. DDL changes.
6. Update the demo data in the test directory.

If schema versions in dev/release profiles are different,
1. Continue from step 4 above. Add DDL file and update latest.sql and latest_data.sql.
The changes in DDL files should be independent from each other. This allows us to move forward with the release independently.

Since we use the same code for both dev and release schemas, we should add if-else branching to read storage differently based on schema version.

## Release
We will take the following steps for the release with schema update, at most once a month. Every feature, a DDL file, should stage at least two weeks just in case any plan changes before moving to the release.
1. Bump up the MINOR version in the release schema version.
1. Copy over the demo data under demo/test directory to demo/release directory.
1. Move the DDL files and update latest.sql to the next version (unreleased) if they are not ready yet.
