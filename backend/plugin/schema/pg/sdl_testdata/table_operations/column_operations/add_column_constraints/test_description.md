# Add Column Constraints Test

This test validates the SDL migration handling of adding various constraints to columns:

## Test Coverage

- **NOT NULL Constraints**: Adding NOT NULL to nullable columns
- **UNIQUE Constraints**: Adding uniqueness constraints  
- **CHECK Constraints**: Adding value validation constraints
- **Default Values**: Adding default value constraints
- **Combined Constraints**: Multiple constraint types on single columns

## Validation Goals

- Verify accurate detection of new column constraints
- Validate proper DDL generation for constraint additions
- Ensure constraint names are generated correctly
- Test constraint validation logic

## Expected Behavior

The migration should generate proper ALTER TABLE statements to add constraints, including ALTER COLUMN SET NOT NULL and ADD CONSTRAINT statements.