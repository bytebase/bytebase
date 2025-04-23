# About

`bytebase-action` helps to do common chores in database CI/CD with Bytebase.

## Inputs

This action requires the following inputs, configured as environment variables:

### Environment Variables

- **`BYTEBASE_URL`**: The Bytebase instance URL.

- **`BYTEBASE_SERVICE_ACCOUNT`**: The service account email.

- **`BYTEBASE_SERVICE_ACCOUNT_SECRET`**: The service account password.

- **`BYTEBASE_PROJECT`**: The target Bytebase project name.
  - Format: `projects/{project}`

- **`BYTEBASE_TARGETS`**: A comma-separated string listing the target databases or database groups.
  - Can specify a database group or individual databases.
  - Formats:
    - Database: `instances/{instance}/databases/{database}`
    - Database Group: `projects/{project}/databaseGroups/{databaseGroup}`

- **`FILE_PATTERN`**: A pattern used to glob SQL migration files.
