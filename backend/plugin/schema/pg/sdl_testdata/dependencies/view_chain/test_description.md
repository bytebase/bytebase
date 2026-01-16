# View Chain Test

This test validates the SDL migration handling of views that depend on other views:

## Test Coverage

- **View Dependencies**: Views that depend on other views
- **Dependency Ordering**: Creating views in correct order
- **View Chains**: Multiple levels of view dependencies
- **DDL Ordering**: Ensuring base views are created before dependent views

## Validation Goals

- Verify accurate detection of view dependency chains
- Validate proper ordering of CREATE VIEW statements
- Ensure base tables and views are created before dependent views
- Test handling of multi-level view dependencies

## Expected Behavior

The migration should create tables first, then base views, then dependent views in the correct dependency order.
