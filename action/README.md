# About

`bytebase-action` helps to do common chores in database CI/CD with Bytebase.

## Configuration

This action is configured via command-line flags.

### Global Flags

These flags apply to the main `bytebase-action` command and its subcommands (`check`, `rollout`).

-   **`--url`**: The Bytebase instance URL.
    -   Default: `https://demo.bytebase.com`

-   **`--service-account`**: The service account email.
    -   Default: `ci@service.bytebase.com`

-   **`--service-account-secret`**: The service account password.
    -   Default: Reads from the `BYTEBASE_SERVICE_ACCOUNT_SECRET` environment variable. You can override this by providing the flag directly.
    -   *Note: Setting the environment variable `BYTEBASE_SERVICE_ACCOUNT_SECRET` is the recommended way to handle the secret.*

-   **`--project`**: The target Bytebase project name.
    -   Format: `projects/{project}`
    -   Default: `projects/project-sample`

-   **`--targets`**: A comma-separated list or multiple uses of the flag specifying the target databases or database groups.
    -   Can specify a database group or individual databases.
    -   Formats:
        -   Database: `instances/{instance}/databases/{database}`
        -   Database Group: `projects/{project}/databaseGroups/{databaseGroup}`
    -   Default: `instances/test-sample-instance/databases/hr_test,instances/prod-sample-instance/databases/hr_prod`

-   **`--file-pattern`**: A glob pattern used to find SQL migration files.
    -   Used by subcommands like `check` and `rollout` to locate relevant files.
    -   Default: `""` (empty string)

### `check` Command

Checks the SQL migration files found using `--file-pattern`. (No specific flags for this subcommand itself).

Usage: `bytebase-action check [global flags]`

### `rollout` Command Flags

These flags are specific to the `rollout` subcommand (`bytebase-action rollout`). This command rolls out the migration files found using `--file-pattern`.

-   **`--release-title`**: The title of the release created in Bytebase.
    -   Default: The current timestamp in RFC3339 format (e.g., `2025-04-25T17:29:56+08:00`).

-   **`--rollout-title`**: The title of the rollout issue created in Bytebase.
    -   Default: The current timestamp in RFC3339 format (e.g., `2025-04-25T17:29:56+08:00`).

Usage: `bytebase-action rollout [global flags] [rollout flags]`
