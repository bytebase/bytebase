# Enum Used by Table Test

This test validates the SDL migration handling of creating tables that use enum types:

## Test Coverage

- **Enum Type Creation**: Creating enum type before table
- **Table with Enum Column**: Table column using the enum type
- **Dependency Ordering**: Ensuring enum is created before table
- **Default Values**: Enum column with default value

## Validation Goals

- Verify proper ordering of CREATE TYPE before CREATE TABLE
- Validate table column correctly references enum type
- Ensure default enum values are handled correctly
- Test dependency resolution between enum and table

## Expected Behavior

The migration should generate CREATE TYPE statement first, followed by CREATE TABLE statement that references the enum type.
