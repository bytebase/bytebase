# Incompatible Data Type Conversions Test

This test validates the SDL migration handling of potentially incompatible type conversions that may require data validation or cause data loss:

**High Risk Conversions**:
- VARCHAR(100) → NUMERIC(10,2) - Text to numeric conversion requiring validation
- BIGINT → SMALLINT - Potential integer overflow
- DOUBLE PRECISION → REAL - Precision loss in floating point
- TIMESTAMP WITH TIME ZONE → DATE - Time component loss

**Data Structure Loss**:
- JSONB → TEXT - Structured data becomes plain text
- INTEGER[] → TEXT - Array structure becomes string representation
- TEXT → INTEGER - String parsing with potential failures

**Constraint Conflicts**:
- VARCHAR(500) → VARCHAR(10) - Potential data truncation
- NULL allowed → NOT NULL - Existing null values become invalid
- No constraint → UNIQUE - Existing duplicates become invalid
- Default value changes - May affect application behavior

**Format Validation**:
- TEXT → INTEGER - String parsing validation required

These conversions typically require careful data migration planning and validation to ensure no data loss or constraint violations occur during the schema evolution.