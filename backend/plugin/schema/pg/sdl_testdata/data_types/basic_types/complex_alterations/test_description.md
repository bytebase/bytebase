# Complex Data Type Alterations Test

This test validates the SDL migration handling of complex column alterations:
- Multiple column type changes in a single table
- Adding new columns with constraints and defaults
- Modifying existing column constraints and defaults
- Handling array data types and JSONB modifications
- Date/time precision changes and timezone conversions

Tests comprehensive schema evolution scenarios that combine multiple alteration types.