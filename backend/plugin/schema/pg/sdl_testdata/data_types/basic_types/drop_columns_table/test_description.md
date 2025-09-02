# DROP Columns and Table Test

This test validates the SDL migration handling of DROP operations:
- Dropping individual columns from existing tables
- Dropping entire tables with various data types
- Handling dependencies (indexes, constraints) when dropping columns
- Proper cleanup of sequences and related objects

Tests proper DDL generation for destructive schema operations and dependency management.