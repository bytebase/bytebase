# Drop Columns Test

This test validates the SDL migration handling of dropping columns from existing tables:

## Test Coverage

- **Single Column Drop**: Removing individual columns
- **Multiple Column Drops**: Removing several columns at once
- **Various Data Types**: Dropping columns with different data types
- **Dependency Handling**: Ensuring constraints and indexes are properly handled

## Validation Goals

- Verify accurate detection of dropped columns
- Validate proper DDL generation for DROP COLUMN statements  
- Ensure dependent indexes and constraints are dropped
- Test table structure integrity after column removal

## Expected Behavior

The migration should generate proper ALTER TABLE DROP COLUMN statements while automatically handling dependent objects like indexes and constraints.