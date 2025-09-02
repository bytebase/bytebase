-- Schema with potentially incompatible type conversions
CREATE TABLE public.risky_conversions (
    conversion_id SERIAL PRIMARY KEY,
    text_to_numeric NUMERIC(10,2) NOT NULL,       -- VARCHAR(100) → NUMERIC(10,2) (data validation required)
    large_int SMALLINT,                           -- BIGINT → SMALLINT (potential overflow)
    high_precision REAL,                          -- DOUBLE PRECISION → REAL (precision loss)
    timestamp_field DATE,                         -- TIMESTAMP WITH TIME ZONE → DATE (time component lost)
    json_field TEXT,                              -- JSONB → TEXT (structure lost)
    array_field TEXT,                             -- INTEGER[] → TEXT (array structure lost)
    numeric_text INTEGER                          -- TEXT → INTEGER (parsing required)
);

CREATE TABLE public.data_conflicts (
    conflict_id SERIAL PRIMARY KEY,
    wide_varchar VARCHAR(10) NOT NULL,            -- VARCHAR(500) → VARCHAR(10) (truncation risk)
    nullable_required VARCHAR(100) NOT NULL,      -- Added NOT NULL constraint (existing nulls problem)
    unique_field VARCHAR(100) UNIQUE,             -- Added UNIQUE constraint (duplicate values problem)
    default_change INTEGER DEFAULT 999,           -- Changed default: 100 → 999
    type_change INTEGER                           -- TEXT → INTEGER (parsing validation required)
);

-- These conversions represent various incompatibility scenarios:
-- 1. Data validation failures (text that can't convert to numeric)
-- 2. Data truncation (large values that don't fit in smaller types)
-- 3. Precision loss (high precision to low precision)
-- 4. Information loss (timestamp to date, structured to text)
-- 5. Constraint violations (nulls where NOT NULL required)
-- 6. Uniqueness violations (duplicates where UNIQUE required)
-- 7. Format validation failures (invalid integer strings)