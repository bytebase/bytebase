-- Initial schema with types that will be converted incompatibly
CREATE TABLE public.risky_conversions (
    conversion_id SERIAL PRIMARY KEY,
    text_to_numeric VARCHAR(100) NOT NULL,        -- Will convert to NUMERIC
    large_int BIGINT,                             -- Will convert to SMALLINT (lossy)
    high_precision DOUBLE PRECISION,              -- Will convert to REAL (precision loss)
    timestamp_field TIMESTAMP WITH TIME ZONE,     -- Will convert to DATE (time loss)
    json_field JSONB,                             -- Will convert to TEXT
    array_field INTEGER[],                        -- Will convert to TEXT
    numeric_text TEXT                             -- Contains numeric data, will convert to INTEGER
);

CREATE TABLE public.data_conflicts (
    conflict_id SERIAL PRIMARY KEY,
    wide_varchar VARCHAR(500) NOT NULL,           -- Will reduce to VARCHAR(10)
    nullable_required VARCHAR(100),               -- Will become NOT NULL
    unique_field VARCHAR(100),                    -- Will add UNIQUE constraint
    default_change INTEGER DEFAULT 100,           -- Will change default value
    type_change TEXT                              -- Will change to completely different type
);