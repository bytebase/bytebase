# Compatible Data Type Conversions Test

This test validates the SDL migration handling of safe, compatible data type conversions:

**Numeric Type Promotions**:
- SMALLINT → INTEGER → BIGINT
- INTEGER → NUMERIC/DECIMAL  
- REAL → DOUBLE PRECISION
- NUMERIC precision expansion

**String Type Extensions**:
- CHAR(n) → VARCHAR(m) where m >= n
- VARCHAR(n) → VARCHAR(m) where m > n
- VARCHAR(n) → TEXT
- TEXT → CITEXT

**Temporal Type Enhancements**:
- DATE → TIMESTAMP
- TIMESTAMP → TIMESTAMP WITH TIME ZONE
- TIME → TIME WITH TIME ZONE

Tests data-preserving conversions that should execute without data loss.