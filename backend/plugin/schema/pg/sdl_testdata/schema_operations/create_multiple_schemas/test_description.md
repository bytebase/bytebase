# Create Multiple Schemas Test

This test validates the SDL migration handling of creating multiple schemas:

## Test Coverage

- **Multiple Schema Creation**: Creating multiple schemas in one migration
- **Schema Ordering**: Proper ordering of CREATE SCHEMA statements
- **Tables in Different Schemas**: Tables created in different schemas
- **Schema Isolation**: Ensuring objects are properly scoped to their schemas

## Validation Goals

- Verify accurate detection of new schemas
- Validate proper CREATE SCHEMA DDL generation
- Ensure tables are created in correct schemas
- Test schema qualification in table DDL

## Expected Behavior

The migration should generate CREATE SCHEMA statements before creating any objects within those schemas, with proper schema qualification for all objects.
