# String Data Types Test

This test validates the SDL migration handling of PostgreSQL string data types:
- CHAR, VARCHAR with different lengths
- TEXT and unlimited text
- CITEXT for case-insensitive text

Tests both creation and modification of string columns with length changes.