# Cross-Schema Foreign Key Test

This test validates the SDL migration handling of foreign keys across schemas:

## Test Coverage

- **Cross-Schema References**: Foreign key from one schema to another
- **Dependency Ordering**: Ensuring referenced table is created first
- **Schema Qualification**: Proper schema qualification in FK constraints
- **Multi-Schema Coordination**: Multiple schemas with dependencies

## Validation Goals

- Verify accurate detection of cross-schema foreign keys
- Validate proper ordering of DDL statements
- Ensure FK constraints are schema-qualified
- Test dependency resolution across schemas

## Expected Behavior

The migration should create schemas and tables in the correct order, ensuring referenced tables exist before creating foreign keys that reference them, with proper schema qualification in all FK DDL.
