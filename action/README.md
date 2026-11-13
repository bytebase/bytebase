# About

`bytebase-action` helps to do common chores in database CI/CD with Bytebase.

## Commands

This action provides several subcommands to interact with Bytebase.

### `check`

Usage: `bytebase-action check [global flags]`

Checks the SQL files matching the `--file-pattern`. This is typically used for linting or pre-deployment validation within a CI pipeline. Use `--output` to save check results to a JSON file.

### `rollout`

Usage: `bytebase-action rollout [global flags] [rollout flags]`

Creates a new release and initiates a rollout issue in the specified Bytebase `--project`.
If a `--plan` is specified, it rolls out that specific plan.
Otherwise, it applies the SQL files matching the `--file-pattern` to the defined `--targets`.
The rollout will proceed up to the specified `--target-stage`.
The release is named using the `--release-id-template` flag.

## Configuration

This action is configured via command-line flags. Global flags apply to all commands, while some commands have specific flags.

### Global Flags

These flags apply to the main `bytebase-action` command and its subcommands (`check`, `rollout`).

-   **`--output`**: The output file location. The output file is a JSON file with the created resource names and check results.
    -   Default: `""` (empty string)
    -   For `check` command: outputs detailed check results including advices, affected rows, and risk levels
    -   For `rollout` command: outputs created resource names (release, plan, rollout) and the rollout result. See [Rollout result](#rollout-result).

-   **`--url`**: The Bytebase instance URL. Required.

-   **`--workload-identity`**: The workload identity email. The action signs in as it with the CI job's ID token, so the job stores no secret. This is the recommended sign-in.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_WORKLOAD_IDENTITY` environment variable.
    -   See [Sign-in](#sign-in) for where the ID token comes from.

-   **`--workload-identity-audience`**: The audience of the ID token requested from GitHub.
    -   Default: `bytebase`. If not provided via flag, reads from the `BYTEBASE_WORKLOAD_IDENTITY_AUDIENCE` environment variable.
    -   It must be one of the audiences the workload identity allows.

-   **`--service-account`**: The service account email.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_SERVICE_ACCOUNT` environment variable.

-   **`--service-account-secret`**: The service account password.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_SERVICE_ACCOUNT_SECRET` environment variable.
    -   *Note: Setting the environment variable `BYTEBASE_SERVICE_ACCOUNT_SECRET` is the recommended way to handle the secret.*

-   **`--access-token`**: A Bytebase access token to use as given.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_ACCESS_TOKEN` environment variable.
    -   The action cannot renew it, so a run that outlasts the token fails.

-   **`--custom-header`**: A custom HTTP header to include in Bytebase API requests.
    -   Format: `Name: value`
    -   Can be specified multiple times.
    -   This is useful when the Bytebase URL is protected by a header-based access proxy such as Cloudflare Access.
    -   Example: `--custom-header="Cookie: CF_Authorization=${CF_AUTHORIZATION_COOKIE}"`
    -   The headers are sent to Bytebase only, not to GitHub's ID token endpoint.

-   **`--project`**: The target Bytebase project name.
    -   Format: `projects/{project}`
    -   Default: `projects/hr`

-   **`--targets`**: A comma-separated list or multiple uses of the flag specifying the target databases or database groups.
    -   Used when `--plan` is not specified for the `rollout` command.
    -   Can specify a database group or individual databases.
    -   Formats:
        -   Workspace-instance database: `instances/{instance}/databases/{database}`
        -   Project-instance database: `projects/{project}/instances/{instance}/databases/{database}`
        -   Database Group: `projects/{project}/databaseGroups/{databaseGroup}`
    -   Default: `instances/test-sample-instance/databases/hr_test,instances/prod-sample-instance/databases/hr_prod`

-   **`--file-pattern`**: A glob pattern used to find SQL files.
    -   Used by subcommands like `check` and `rollout` (when `--plan` is not specified) to locate relevant files.
    -   Default: `""` (empty string)
    -   **Versioned Mode** (when `--declarative` is false):
        -   Migration filenames must conform to a versioning format
        -   The version part of the filename must start with an optional 'v' or 'V', followed by one or more numbers, with subsequent numbers separated by a dot
        -   Examples: `v1.2.3_description.sql`, `1.0_initial_schema.sql`, `V2_add_users_table.sql`
        -   The version is extracted based on the pattern `^[vV]?(\d+(\.\d+)*)`
    -   **Declarative Mode** (when `--declarative` is true):
        -   Filenames do not need to follow any versioning format
        -   Files can be named for clarity and organization (e.g., `tables.sql`, `views.sql`, `indexes.sql`)
        -   Version is automatically generated from the current timestamp
    -   **Migration Type Detection** (applies to versioned mode only):
        -   Migration type is specified using a comment at the top of the SQL file
        -   Format: `-- migration-type: ghost`
        -   The comment must appear before any SQL statements
        -   Case insensitive (e.g., `Ghost`, `GHOST`, `ghost` all work)
        -   Only `ghost` is supported for gh-ost migrations
        -   Example:
            ```sql
            -- migration-type: ghost
            ALTER TABLE large_table ADD COLUMN new_col VARCHAR(255);
            ```
        -   If no migration type is specified (or any other value), defaults to `MIGRATION_TYPE_UNSPECIFIED`

-   **`--declarative`** (experimental): Use declarative mode for SQL schema management instead of versioned migrations.
    -   Treats SQL files as desired state definitions rather than incremental changes
    -   Allows organizing schema across multiple files (e.g., `tables.sql`, `views.sql`)
    -   Versions are auto-generated using timestamp format `YYYYMMDD.HHMMSS`
    -   Default: `false`

### `check` Command Specific Flags

These flags are specific to the `check` subcommand (`bytebase-action check`).

-   **`--check-release`**: Determines whether to fail the command based on check results.
    -   Valid values:
        -   `SKIP`: Do not fail regardless of check results (default behavior).
        -   `FAIL_ON_WARNING`: Fail if there are warnings or errors in the check results.
        -   `FAIL_ON_ERROR`: Fail only if there are errors in the check results.
    -   Default: `SKIP`
    -   Note: Platform-specific outputs (GitHub comments, GitLab reports, etc.) are always generated before evaluating whether to fail.

### `rollout` Command Specific Flags

These flags are specific to the `rollout` subcommand (`bytebase-action rollout`).

-   **`--target-stage`**: The target stage up to which the rollout should proceed. If not specified, the rollout will be created but will not wait for completion.
    -   Format: `environments/{environment}`
    -   Example: `environments/prod`

-   **`--plan`**: The specific plan to rollout.
    -   Format: `projects/{project}/plans/{plan}`
    -   If specified, this shadows the `--file-pattern` and `--targets` flags, meaning they will be ignored.

## Rollout result

When `rollout` stops waiting, it reports how the rollout stands. The report is written whether the command passes or fails, and it never changes the exit code.

-   **Job log**: a block with one line per database, the files it ran, the statements that changed rows, and for a failure the statement and the database's error. A listed statement is cut to one line of 100 characters; a failed statement and its error are cut to 30 lines of 200 characters.
-   **`--output` file**: the same result under the `result` key, in full.
-   **GitHub**: the block as the step summary, and an error annotation for each failed database.
-   **GitLab**: `bytebase_annotations.json`, an [annotations report](https://docs.gitlab.com/ci/yaml/artifacts_reports/#artifactsreportsannotations) that links the job page to the rollout.

`result.outcome` says how the run ended:

| Outcome | Meaning | Exit code |
|---|---|---|
| `DONE` | Every task in the target stage is done or skipped | 0 |
| `FAILED` | A task failed | 1 |
| `CANCELED` | A task was canceled | 1 |
| `NOT_WAITED` | No `--target-stage` was given, so the rollout was created and nothing was awaited | 0 |
| `STAGE_NOT_FOUND` | The rollout has no stage for `--target-stage`, so nothing was deployed | 0 |
| `INTERRUPTED` | The action stopped before the stage finished, so the state is not final | 1 |

The result details one stage: the target stage, or an earlier stage when the rollout stopped there. Other stages carry task counts only.

```json
{
  "plan": "projects/example/plans/101",
  "rollout": "projects/example/plans/101/rollout",
  "result": {
    "outcome": "FAILED",
    "url": "https://bytebase.example.com/projects/example/plans/101/rollout",
    "targetStage": "environments/prod",
    "stages": [
      {
        "environment": "environments/prod",
        "taskCounts": { "FAILED": 1 },
        "tasks": [
          {
            "target": "instances/prod/databases/crm",
            "status": "FAILED",
            "attempt": 1,
            "affectedRows": 0,
            "filesNotRun": 1,
            "files": [
              {
                "version": "0004",
                "path": "migrations/0004_add_risk_score.sql",
                "status": "FAILED",
                "rolledBack": true,
                "affectedRows": 0,
                "commands": [
                  { "statement": "UPDATE account SET tier = 'basic' WHERE tier IS NULL", "affectedRows": 40, "rolledBack": true },
                  { "statement": "ALTER TABLE account ADD COLUMN risk_score int", "error": "ERROR: column \"risk_score\" of relation \"account\" already exists (SQLSTATE 42701)" }
                ]
              }
            ]
          }
        ]
      }
    ]
  }
}
```

Reading the rows:

-   `affectedRows` is the count the database returned. A file's count sums its commands, and a task's sums its files.
-   A missing `affectedRows` means the database driver reported none. It does not mean zero.
-   Commands in a transaction that rolled back keep their count, carry `rolledBack`, and are left out of the file and task totals.
-   A command with `earlierAttempt` committed in an attempt Bytebase then retried, which happens when a file runs outside a transaction. It counts toward the totals, and the same statement appears again from the final attempt.
-   A task's result is its latest run. `attempt` counts its runs, and `previousRuns` lists the earlier ones.
-   A task with `unchanged` finished with no file left to apply.

Statement text comes from the migration files, because the server logs each command as a byte range of its file, counted after the `-- txn-mode` and `-- txn-isolation` lines at its top are removed. The action reads a file from the job's checkout when it matches the release, so run `rollout` in a checkout of the repository. Without one it reads the file from Bytebase, which needs the `bb.sheets.get` permission. With neither, a command is named by its position, such as `statement 3`, and its count and error are still reported.

To keep the file on GitLab when the job fails:

```yaml
deploy:
  script:
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --plan=$PLAN --target-stage=environments/prod --output=bytebase-rollout.json
  artifacts:
    when: always
    paths: [bytebase-rollout.json]
    reports:
      annotations: bytebase_annotations.json
```

## Sign-in

The action signs in once, before any other request. It uses the first of these that is set:

1.  An access token, from `--access-token` or `BYTEBASE_ACCESS_TOKEN`.
2.  `--workload-identity`, the flag.
3.  A service account, from `--service-account` and `--service-account-secret` or their environment variables.
4.  `BYTEBASE_WORKLOAD_IDENTITY`, the environment variable.

A workload identity needs the CI job's ID token:

-   **GitHub Actions**: give the job `permissions: id-token: write`. The action requests the token itself.

    ```yaml
    jobs:
      sql-review:
        permissions:
          id-token: write
        runs-on: ubuntu-latest
        container:
          image: bytebase/bytebase-action
        env:
          BYTEBASE_WORKLOAD_IDENTITY: ci@workload.bytebase.com
        steps:
          - uses: actions/checkout@v4
          - run: bytebase-action check --url=https://bytebase.example.com --project=projects/example --targets=instances/example/databases/example --file-pattern="migrations/*.sql"
    ```

-   **GitLab CI and other OIDC issuers**: put the ID token in `BYTEBASE_ID_TOKEN`. Its `aud` must be one of the audiences the workload identity allows.

    ```yaml
    sql-review:
      image: bytebase/bytebase-action
      id_tokens:
        BYTEBASE_ID_TOKEN:
          aud: bytebase
      variables:
        BYTEBASE_WORKLOAD_IDENTITY: ci@workload.bytebase.com
      script:
        - bytebase-action check --url=https://bytebase.example.com --project=projects/example --targets=instances/example/databases/example --file-pattern="migrations/*.sql"
    ```

`BYTEBASE_ID_TOKEN` is also read on GitHub, and wins over the token GitHub would issue.

A failed sign-in ends the run with one line, `sign in as <identity>: <reason>`.

When a request is rejected as unauthenticated, the action signs in again. A workload identity can do so only while the CI still has a valid ID token: GitLab's expires at the job timeout.

## Using Declarative Mode

Declarative mode is an experimental feature currently in development that allows you to manage database schemas as desired state definitions rather than versioned migrations.

### How to Use

To enable declarative mode, add the `--declarative` flag to your command:

```bash
bytebase-action rollout --declarative --file-pattern="schema/*.sql" [other flags]
```

In declarative mode, your SQL files represent the complete desired state of your database schema. The system will automatically:
- Compare the desired state with the current database state
- Generate the necessary changes to transform the current state into the desired state

### Getting Started

When using declarative mode, you must follow these steps:

1. **Export Current Schema**: In the Bytebase database detail page, click `Export Schema` to download your current database schema.
2. **Edit Schema Files**: Start from and edit the downloaded schema files with your desired modifications.
3. **Run Declarative Rollout**: Use the `--declarative` flag to apply your changes.

### Important Limitations

1. **Database Support**: Currently only PostgreSQL is supported.

2. **Supported SQL Statements**: The following PostgreSQL statements are supported:
   - `CREATE SCHEMA`
   - `CREATE TABLE`
   - `CREATE INDEX` / `CREATE UNIQUE INDEX`
   - `CREATE VIEW` / `CREATE MATERIALIZED VIEW`
   - `CREATE SEQUENCE` / `ALTER SEQUENCE`
   - `CREATE FUNCTION`
   - `CREATE TYPE ... AS ENUM` (enum types)
   - `CREATE TYPE ... AS (...)` (composite types)
   - `CREATE TRIGGER`
   - `CREATE EXTENSION`
   - `COMMENT ON`

   > **Note**: Domain and range types (`CREATE DOMAIN`, `CREATE TYPE ... AS RANGE`) are parsed, but schema sync does not yet capture them, so databases relying on them are not yet fully supported in declarative mode.

3. **Schema Requirements**: You must use fully qualified names (with schema prefix) for all database objects in your schema files:
   ```sql
   -- Correct: fully qualified name
   CREATE TABLE public.users (
       id INTEGER NOT NULL,
       name VARCHAR(100) NOT NULL,
       CONSTRAINT pk_users PRIMARY KEY (id)
   );

   -- Incorrect: unqualified name
   CREATE TABLE users (
       id INTEGER NOT NULL,
       name VARCHAR(100) NOT NULL,
       CONSTRAINT pk_users PRIMARY KEY (id)
   );
   ```

4. **Constraint Requirements**: `PRIMARY KEY`, `UNIQUE`, `FOREIGN KEY`, and `CHECK` constraints must be defined at table level with explicit names. Only `NOT NULL`, `DEFAULT`, and `GENERATED` constraints are allowed at column level:
   ```sql
   -- Correct: table-level constraints with explicit names
   CREATE TABLE public.users (
       id INTEGER NOT NULL,              -- NOT NULL is allowed at column level
       email TEXT NOT NULL,              -- NOT NULL is allowed at column level
       created_at TIMESTAMP DEFAULT NOW(), -- DEFAULT is allowed at column level
       CONSTRAINT pk_users PRIMARY KEY (id),
       CONSTRAINT uk_users_email UNIQUE (email),
       CONSTRAINT chk_users_email CHECK (email LIKE '%@%')
   );

   -- Incorrect: PRIMARY KEY, UNIQUE, CHECK at column level
   CREATE TABLE public.users (
       id INTEGER PRIMARY KEY,           -- ERROR: PRIMARY KEY must be at table level
       email TEXT UNIQUE,                -- ERROR: UNIQUE must be at table level
       age INTEGER CHECK (age >= 0),     -- ERROR: CHECK must be at table level
       CONSTRAINT pk_users PRIMARY KEY (id)
   );

   -- Incorrect: unnamed constraints
   CREATE TABLE public.users (
       id INTEGER NOT NULL,
       email TEXT NOT NULL,
       PRIMARY KEY (id),                 -- ERROR: constraint must have explicit name
       UNIQUE (email),                   -- ERROR: constraint must have explicit name
       CHECK (email LIKE '%@%')          -- ERROR: constraint must have explicit name
   );
   ```

5. **Foreign Key References**: Foreign key references must use fully qualified table names:
   ```sql
   -- Correct: fully qualified reference
   CREATE TABLE public.orders (
       id INTEGER NOT NULL,
       user_id INTEGER NOT NULL,
       CONSTRAINT pk_orders PRIMARY KEY (id),
       CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
   );

   -- Incorrect: unqualified reference
   CREATE TABLE public.orders (
       id INTEGER NOT NULL,
       user_id INTEGER NOT NULL,
       CONSTRAINT pk_orders PRIMARY KEY (id),
       CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id)
   );
   ```

6. **Index Naming**: All indexes must have explicit names:
   ```sql
   -- Correct: named index
   CREATE INDEX idx_users_email ON public.users(email);

   -- Incorrect: unnamed index
   CREATE INDEX ON public.users(email);
   ```
