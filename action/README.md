# About

`bytebase-action` helps to do common chores in database CI/CD with Bytebase.

## Commands

### check

`bytebase-action check` checks the migration files. This is typically done in the CI phase.

## Configuration

This action is configured via command-line flags.

### Global Flags

These flags apply to the main `bytebase-action` command and its subcommands.

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

### `check` Command Flags

These flags are specific to the `check` subcommand (`bytebase-action check`).

-   **`--file-pattern`**: A glob pattern used to find SQL migration files to check.
    -   Default: `""` (empty string)
    -   *Required for the `check` command to find files.*
