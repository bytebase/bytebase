# About

`bytebase-action` helps to do common chores in database CI/CD with Bytebase.

## Commands

This action provides several subcommands to interact with Bytebase.

### `check`

Usage: `bytebase-action check [global flags]`

Checks the SQL files matching the `--file-pattern`. This is typically used for linting or pre-deployment validation within a CI pipeline. It utilizes global flags like `--url`, `--service-account`, `--service-account-secret`, `--file-pattern`, and `--declarative`.

### `rollout`

Usage: `bytebase-action rollout [global flags] [rollout flags]`

Creates a new release and initiates a rollout issue in the specified Bytebase `--project`.
If a `--plan` is specified, it rolls out that specific plan.
Otherwise, it applies the SQL files matching the `--file-pattern` to the defined `--targets`.
The rollout will proceed up to the specified `--target-stage`.
It uses global flags for connection and file discovery (unless a plan is specified), and specific flags like `--release-title` to name the created resources in Bytebase.

## Configuration

This action is configured via command-line flags. Global flags apply to all commands, while some commands have specific flags.

### Global Flags

These flags apply to the main `bytebase-action` command and its subcommands (`check`, `rollout`).

-   **`--output`**: The output file location. The output file is a JSON file with the created resource names.
    -   Default: `""` (empty string)

-   **`--url`**: The Bytebase instance URL.
    -   Default: `https://demo.bytebase.com`

-   **`--service-account`**: The service account email.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_SERVICE_ACCOUNT` environment variable.

-   **`--service-account-secret`**: The service account password.
    -   Default: `""` (empty string). If not provided via flag, reads from the `BYTEBASE_SERVICE_ACCOUNT_SECRET` environment variable.
    -   *Note: Setting the environment variable `BYTEBASE_SERVICE_ACCOUNT_SECRET` is the recommended way to handle the secret.*

-   **`--project`**: The target Bytebase project name.
    -   Format: `projects/{project}`
    -   Default: `projects/hr`

-   **`--targets`**: A comma-separated list or multiple uses of the flag specifying the target databases or database groups.
    -   Used when `--plan` is not specified for the `rollout` command.
    -   Can specify a database group or individual databases.
    -   Formats:
        -   Database: `instances/{instance}/databases/{database}`
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
    -   **File Type Detection** (applies to versioned mode only):
        -   **DDL (default)**: Standard schema change files (e.g., `v1.0_create_table.sql`)
        -   **DML**: Data manipulation files with base filename ending with `dml` (e.g., `v1.0_insert_data_dml.sql`)
        -   **DDL Ghost**: Schema changes using gh-ost with base filename ending with `ghost` (e.g., `v1.0_alter_table_ghost.sql`)

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

-   **`--release-title`**: The title of the release created in Bytebase.
    -   Default: The current timestamp in RFC3339 format (e.g., `2025-04-25T17:32:07+08:00`).

-   **`--check-plan`**: Determines whether to run plan checks and how to handle failures.
    -   Valid values:
        -   `SKIP`: Do not run plan checks.
        -   `FAIL_ON_WARNING`: Run plan checks and fail if there are warnings or errors.
        -   `FAIL_ON_ERROR`: Run plan checks and fail only if there are errors.
    -   Default: `SKIP`

-   **`--target-stage`**: The target stage up to which the rollout should proceed. If not specified, the rollout will be created but will not wait for completion.
    -   Format: `environments/{environment}`
    -   Example: `environments/prod`

-   **`--plan`**: The specific plan to rollout.
    -   Format: `projects/{project}/plans/{plan}`
    -   If specified, this shadows the `--file-pattern` and `--targets` flags, meaning they will be ignored.

## Using Declarative Mode (Experimental)

Declarative mode is an experimental feature currently in development that allows you to manage database schemas as desired state definitions rather than versioned migrations.

### How to Use

To enable declarative mode, add the `--declarative` flag to your command:

```bash
bytebase-action rollout --declarative --file-pattern="schema/*.sql" [other flags]
```

In declarative mode, your SQL files represent the complete desired state of your database schema. The system will automatically:
- Compare the desired state with the current database state
- Generate the necessary changes to transform the current state into the desired state

### Important Limitations

1. **Database Support**: Currently only MySQL and PostgreSQL are supported.

2. **Supported SQL Statements**: Only the following CREATE statements are supported:
   - `CREATE TABLE`
   - `CREATE INDEX`
   - `CREATE FUNCTION`
   - `CREATE PROCEDURE`
   - `CREATE VIEW`
   - `CREATE SEQUENCE`

3. **PostgreSQL Specific Requirements**: When using PostgreSQL, you must use fully qualified names for all database objects in your schema files:
   ```sql
   -- Correct: fully qualified name
   CREATE TABLE myschema.users (
       id SERIAL PRIMARY KEY,
       name VARCHAR(100)
   );
   
   -- Incorrect: unqualified name
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       name VARCHAR(100)
   );
   ```

### Example File Organization

With declarative mode, you can organize your schema across multiple files:

```
schema/
├── tables.sql      # All table definitions
├── indexes.sql     # All index definitions
├── views.sql       # All view definitions
└── functions.sql   # All function/procedure definitions
```

Each file should contain the complete desired state for its respective database objects.
