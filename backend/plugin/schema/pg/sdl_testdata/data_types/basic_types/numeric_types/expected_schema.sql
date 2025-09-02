-- Test all PostgreSQL numeric data types
CREATE TABLE public.numeric_types_test (
    id SERIAL PRIMARY KEY,
    small_int_col SMALLINT NOT NULL,
    int_col INTEGER DEFAULT 0,
    big_int_col BIGINT,
    
    -- Decimal with different precision and scale
    decimal_col DECIMAL(10,2) NOT NULL,
    numeric_col NUMERIC(15,3),
    numeric_no_scale NUMERIC(8),
    
    -- Floating point types
    real_col REAL,
    double_col DOUBLE PRECISION,
    
    -- Auto-increment columns
    big_serial_col BIGSERIAL NOT NULL,
    
    -- Constraints
    CHECK (small_int_col > 0),
    CHECK (decimal_col >= 0),
    UNIQUE (int_col, big_int_col)
);

-- Indexes on numeric columns
CREATE INDEX idx_numeric_decimal ON public.numeric_types_test(decimal_col);
CREATE INDEX idx_numeric_big_int ON public.numeric_types_test(big_int_col) WHERE big_int_col IS NOT NULL;
CREATE INDEX idx_numeric_composite ON public.numeric_types_test(int_col, real_col);