-- Test all PostgreSQL date and time data types
CREATE TABLE public.datetime_types_test (
    id SERIAL PRIMARY KEY,
    
    -- Date types
    date_col DATE DEFAULT CURRENT_DATE,
    date_birth DATE NOT NULL,
    
    -- Time types
    time_col TIME,
    time_with_tz TIME WITH TIME ZONE,
    time_precise TIME(6) DEFAULT CURRENT_TIME,
    
    -- Timestamp types
    timestamp_col TIMESTAMP,
    timestamp_with_tz TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    timestamp_created TIMESTAMP DEFAULT NOW(),
    timestamp_precise TIMESTAMP(3) NOT NULL,
    
    -- Interval type
    interval_col INTERVAL,
    interval_default INTERVAL DEFAULT '1 day',
    
    -- Constraints
    CHECK (date_birth <= CURRENT_DATE),
    CHECK (timestamp_precise >= '2020-01-01'::timestamp),
    CHECK (interval_col IS NULL OR interval_col >= '0 seconds'::interval)
);

-- Indexes on datetime columns
CREATE INDEX idx_datetime_date ON public.datetime_types_test(date_col);
CREATE INDEX idx_datetime_timestamp_tz ON public.datetime_types_test(timestamp_with_tz);
CREATE INDEX idx_datetime_created_desc ON public.datetime_types_test(timestamp_created DESC);
CREATE INDEX idx_datetime_birth_year ON public.datetime_types_test(EXTRACT(YEAR FROM date_birth));