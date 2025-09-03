# Add Column Foreign Keys Test

This test validates the SDL migration handling of adding foreign key constraints to columns:

## Test Coverage

- **Basic Foreign Keys**: Simple references to primary keys
- **CASCADE Actions**: ON DELETE CASCADE, ON UPDATE CASCADE
- **SET NULL Actions**: ON DELETE SET NULL, ON UPDATE SET NULL
- **Multiple Foreign Keys**: Multiple references in same table
- **Self-References**: Foreign keys referencing the same table

## Validation Goals

- Verify accurate detection of new foreign key constraints
- Validate proper ALTER TABLE ADD CONSTRAINT DDL generation
- Ensure referential actions are correctly specified
- Test constraint naming and dependency resolution

## Expected Behavior

The migration should generate proper ALTER TABLE ADD CONSTRAINT statements with correct references and cascade actions.