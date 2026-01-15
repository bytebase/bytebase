# Cyclic FK with Views Test

This test validates the SDL migration handling of views depending on tables with cyclic foreign keys:

## Test Coverage

- **Complex Dependencies**: Cyclic FKs combined with view dependencies
- **Multi-Layer Dependencies**: Views that depend on tables with circular FKs
- **Dependency Resolution**: Proper ordering across multiple dependency types
- **Mixed Object Types**: Tables, FKs, and views in correct order

## Validation Goals

- Verify accurate detection of complex dependency chains
- Validate proper ordering of CREATE TABLE, ALTER TABLE, and CREATE VIEW
- Ensure cyclic FKs don't block view creation
- Test handling of multi-dimensional dependencies

## Expected Behavior

The migration should create tables first, then add FK constraints, then create views that depend on those tables, ensuring all dependencies are satisfied in the correct order.
