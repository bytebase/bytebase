-- Schema with mixed precision changes (both expansions and reductions)
CREATE TABLE public.data_samples (
    sample_id SERIAL PRIMARY KEY,
    short_text VARCHAR(100) NOT NULL,              -- Expanded: VARCHAR(50) → VARCHAR(100)
    long_text VARCHAR(150),                        -- Reduced: VARCHAR(200) → VARCHAR(150) 
    fixed_char CHAR(20),                           -- Expanded: CHAR(10) → CHAR(20)
    precise_decimal NUMERIC(16,6) DEFAULT 0.000000, -- Expanded: NUMERIC(12,4) → NUMERIC(16,6)
    simple_decimal NUMERIC(5,1) DEFAULT 0.0,       -- Reduced: NUMERIC(6,2) → NUMERIC(5,1)
    bit_flags BIT(32) DEFAULT B'00000000000000000000000000000000', -- Expanded: BIT(16) → BIT(32)
    var_bits VARBIT(16)                            -- Reduced: VARBIT(32) → VARBIT(16)
);

CREATE TABLE public.measurements (
    measurement_id SERIAL PRIMARY KEY,
    device_name VARCHAR(150) NOT NULL,             -- Expanded: VARCHAR(100) → VARCHAR(150)
    location_code CHAR(15),                        -- Reduced: CHAR(20) → CHAR(15)
    value_high_precision NUMERIC(20,12),           -- Expanded: NUMERIC(15,8) → NUMERIC(20,12)
    value_low_precision NUMERIC(6,2),              -- Reduced: NUMERIC(8,3) → NUMERIC(6,2)
    status_bits BIT(64),                           -- Expanded: BIT(32) → BIT(64)
    metadata_bits VARBIT(32)                       -- Reduced: VARBIT(64) → VARBIT(32)
);

CREATE TABLE public.precision_test (
    test_id SERIAL PRIMARY KEY,
    expand_varchar VARCHAR(500),                   -- New expanded varchar
    reduce_varchar VARCHAR(25),                    -- New reduced varchar  
    expand_numeric NUMERIC(25,10),                 -- New high precision numeric
    reduce_numeric NUMERIC(4,1),                   -- New low precision numeric
    expand_char CHAR(50),                          -- New expanded char
    reduce_char CHAR(5),                           -- New reduced char
    expand_bits BIT(128),                          -- New expanded bit
    reduce_bits BIT(4),                            -- New reduced bit
    expand_varbits VARBIT(256),                    -- New expanded varbit
    reduce_varbits VARBIT(8)                       -- New reduced varbit
);