# Numeric Data Types Test

This test validates the SDL migration handling of all PostgreSQL numeric data types:
- SMALLINT, INTEGER, BIGINT
- DECIMAL, NUMERIC with various precision/scale
- REAL, DOUBLE PRECISION
- SERIAL, BIGSERIAL

Tests both creation and modification of numeric columns with different constraints.