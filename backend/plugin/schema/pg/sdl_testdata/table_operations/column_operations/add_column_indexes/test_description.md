# Add Column Indexes Test

This test validates the SDL migration handling of adding indexes on existing columns:

## Test Coverage

- **Simple Indexes**: Basic B-tree indexes on single columns
- **Composite Indexes**: Multi-column indexes
- **Partial Indexes**: Indexes with WHERE clauses
- **Specialized Indexes**: GIN indexes for arrays and text search
- **Unique Indexes**: Unique constraint indexes

## Validation Goals

- Verify accurate detection of new indexes
- Validate proper CREATE INDEX DDL generation
- Ensure index types and expressions are correct
- Test partial index conditions and composite column ordering

## Expected Behavior

The migration should generate proper CREATE INDEX statements with correct index types, expressions, and conditions.