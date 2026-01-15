# Create Extension Test

This test validates the SDL migration handling of creating PostgreSQL extensions:

## Test Coverage

- **Extension Creation**: Creating common PostgreSQL extensions
- **UUID Extension**: Creating uuid-ossp for UUID generation
- **Extension Dependencies**: Ensuring extensions are created before objects that use them

## Validation Goals

- Verify accurate detection of new extensions
- Validate proper CREATE EXTENSION DDL generation
- Ensure extension is created with correct schema
- Test extension availability for subsequent objects

## Expected Behavior

The migration should generate proper CREATE EXTENSION statements, typically near the beginning of the migration to ensure availability for other objects.
