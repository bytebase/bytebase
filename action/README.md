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
It uses global flags for connection and file discovery (unless a plan is specified), and specific flags like `--release-title` to name the created resources in Bytebase.

## Configuration

This action is configured via command-line flags. Global flags apply to all commands, while some commands have specific flags.

### Global Flags

These flags apply to the main `bytebase-action` command and its subcommands (`check`, `rollout`).

-   **`--output`**: The output file location. The output file is a JSON file with the created resource names and check results.
    -   Default: `""` (empty string)
    -   For `check` command: outputs detailed check results including advices, affected rows, and risk levels
    -   For `rollout` command: outputs created resource names (release, plan, rollout)

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

-   **`--release-title`**: The title of the release created in Bytebase.
    -   Default: The current timestamp in RFC3339 format (e.g., `2025-04-25T17:32:07+08:00`).

-   **`--target-stage`**: The target stage up to which the rollout should proceed. If not specified, the rollout will be created but will not wait for completion.
    -   Format: `environments/{environment}`
    -   Example: `environments/prod`

-   **`--plan`**: The specific plan to rollout.
    -   Format: `projects/{project}/plans/{plan}`
    -   If specified, this shadows the `--file-pattern` and `--targets` flags, meaning they will be ignored.

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
   - `CREATE TABLE`
   - `CREATE INDEX` / `CREATE UNIQUE INDEX`
   - `CREATE VIEW`
   - `CREATE SEQUENCE`
   - `CREATE FUNCTION`
   - `ALTER SEQUENCE`

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

