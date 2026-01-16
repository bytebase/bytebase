# Bidirectional Foreign Key Test

This test validates the SDL migration handling of bidirectional foreign key relationships:

## Test Coverage

- **Bidirectional FKs**: Two tables with FKs to each other
- **Circular Dependencies**: Managing circular dependencies between tables
- **Deferred Constraints**: Creating tables first, then adding FKs
- **Dependency Resolution**: Proper ordering of CREATE and ALTER statements

## Validation Goals

- Verify accurate detection of bidirectional FK relationships
- Validate proper DDL generation to avoid circular dependency errors
- Ensure tables are created before FK constraints
- Test handling of mutual dependencies between tables

## Expected Behavior

The migration should create both tables without FK constraints first, then add FK constraints using ALTER TABLE statements to break the circular dependency.
