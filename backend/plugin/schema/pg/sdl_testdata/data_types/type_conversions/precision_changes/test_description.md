# Data Type Precision Changes Test

This test validates the SDL migration handling of precision modifications:

**Expanding Precision (Safe)**:
- VARCHAR(50) → VARCHAR(100)
- NUMERIC(8,2) → NUMERIC(12,4)
- CHAR(10) → CHAR(20)
- BIT(8) → BIT(16)
- VARBIT(16) → VARBIT(32)

**Reducing Precision (Potentially Unsafe)**:
- VARCHAR(100) → VARCHAR(50) 
- NUMERIC(12,4) → NUMERIC(8,2)
- CHAR(20) → CHAR(10)
- BIT(16) → BIT(8)

Tests both safe precision expansions and potentially risky precision reductions that may require data validation.