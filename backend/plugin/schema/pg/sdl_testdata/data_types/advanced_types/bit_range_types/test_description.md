# Bit and Range Data Types Test

This test validates the SDL migration handling of bit manipulation and range data types:

**Bit Types**:
- **BIT**: Fixed-length bit strings
- **BIT VARYING (VARBIT)**: Variable-length bit strings

**Range Types**:
- **INT4RANGE**: Integer ranges
- **INT8RANGE**: Bigint ranges
- **NUMRANGE**: Numeric ranges  
- **TSRANGE**: Timestamp ranges
- **TSTZRANGE**: Timestamp with timezone ranges
- **DATERANGE**: Date ranges

Tests creation, constraints, indexing, and range operations for these specialized data types.