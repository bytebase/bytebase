# Schema Update Guide

All schemas are located at [store/migration](https://github.com/bytebase/bytebase/tree/main/store/migration). Bytebase schema release follows semantic versioning. This document goes over the scheme update process during development and release.

## Development
We use different schema versions for [dev](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_dev.go) and [release](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_release.go). 

If schema versions in dev/release profiles are the same (e.g. 1.1.0), which means we are starting a new MINOR version update cycle.
1. Create a new directory under [migration directory](https://github.com/bytebase/bytebase/tree/main/store/migration) with the new version, e.g. 1.2.0.
2. Copy over the schema and data file from the previous version. e.g. [1.1.0/latest.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/latest.sql) and [1.1.0/latest_data.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/latest_data.sql). And make a PR for the changes above.
3. Bump up the MINOR version in the [dev schema version](https://github.com/bytebase/bytebase/blob/main/bin/server/cmd/profile_dev.go), e.g. 1.1.0 will be 1.2.0.
4. Add a new SQL file with DDL statements with a descriptive name, e.g. [0000_github_com.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/0000_github_com.sql).
5. Update latest.sql and latest_data.sql data file w.r.t. DDL changes.
6. Update the demo data in the test directory.

If schema versions in dev/release profiles are different, which means we are in the middle of a MINOR version update cycle already.
1. Continue from step 4 above. Add DDL file and update latest.sql and latest_data.sql, e.g. [0001_sheet_vcs.sql](https://github.com/bytebase/bytebase/blob/main/store/migration/1.1.0/0001_sheet_vcs.sql).

Since we use the same code for both dev and release schemas, we should add if-else branching to read storage differently based on schema version.

## Release
We will take the following steps for the release with schema update, at most once a month. Every feature, a DDL file, should stage at least two weeks just in case any plan changes before moving to the release.
1. Bump up the MINOR version in the release schema version.
1. Copy over the demo data under demo/test directory to demo/release directory.
1. Move the DDL files and update latest.sql to the next version (unreleased) if they are not ready yet.
