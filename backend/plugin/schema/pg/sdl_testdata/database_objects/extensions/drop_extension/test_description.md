# Drop Extension Test

This test validates the SDL migration handling of dropping PostgreSQL extensions:

## Test Coverage

- **Extension Deletion**: Removing an extension from schema
- **Clean Removal**: Dropping extension that is not used by any objects
- **DDL Generation**: Proper DROP EXTENSION statement generation

## Validation Goals

- Verify accurate detection of removed extensions
- Validate proper DROP EXTENSION DDL generation
- Ensure extension can only be dropped when not in use
- Test cleanup of unused extensions

## Expected Behavior

The migration should generate proper DROP EXTENSION statement when extension is removed from SDL and is not used by any objects.
