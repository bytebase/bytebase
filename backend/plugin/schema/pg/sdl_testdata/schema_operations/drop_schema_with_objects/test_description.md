# Drop Schema with Objects Test

This test validates the SDL migration handling of dropping schemas that contain objects:

## Test Coverage

- **Schema Deletion**: Removing a schema with all its objects
- **Cascading Drops**: Ensuring objects in schema are dropped first
- **Drop Ordering**: Proper ordering of DROP statements
- **Clean Removal**: Complete removal of schema and contents

## Validation Goals

- Verify accurate detection of removed schemas
- Validate proper ordering of DROP statements
- Ensure objects are dropped before schema
- Test CASCADE behavior for schema drops

## Expected Behavior

The migration should generate DROP statements in reverse dependency order, dropping all objects within the schema before dropping the schema itself, or using DROP SCHEMA CASCADE.
