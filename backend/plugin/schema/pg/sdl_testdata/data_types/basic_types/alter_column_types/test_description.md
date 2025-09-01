# ALTER Column Types Test

This test validates the SDL migration handling of column type alterations:
- Expanding varchar length (VARCHAR(50) → VARCHAR(100))
- Compatible type conversions (INTEGER → BIGINT)
- Adding nullable columns to existing tables
- Modifying column default values
- Changing column nullability constraints

Tests proper handling of data-preserving schema alterations and their DDL generation.