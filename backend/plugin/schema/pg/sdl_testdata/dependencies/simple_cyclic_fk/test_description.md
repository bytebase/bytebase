# Simple Cyclic Foreign Key Test

This test validates the SDL migration handling of self-referencing foreign keys:

## Test Coverage

- **Self-Referencing FK**: Table with FK to itself
- **Deferred Constraints**: Handling of constraint creation timing
- **Circular Dependencies**: Managing circular dependencies in DDL
- **ALTER TABLE Approach**: Using ALTER TABLE ADD CONSTRAINT for cyclic FKs

## Validation Goals

- Verify accurate detection of cyclic FK relationships
- Validate proper DDL generation using deferred constraints
- Ensure FK constraints are created after table
- Test handling of self-referencing relationships

## Expected Behavior

The migration should create the table first, then add the self-referencing FK constraint using ALTER TABLE ADD CONSTRAINT, avoiding circular dependency issues.
