-- Initial schema with various precisions
CREATE TABLE public.data_samples (
    sample_id SERIAL PRIMARY KEY,
    short_text VARCHAR(50) NOT NULL,
    long_text VARCHAR(200),
    fixed_char CHAR(10),
    precise_decimal NUMERIC(12,4) DEFAULT 0.0000,
    simple_decimal NUMERIC(6,2) DEFAULT 0.00,
    bit_flags BIT(16) DEFAULT B'0000000000000000',
    var_bits VARBIT(32)
);

CREATE TABLE public.measurements (
    measurement_id SERIAL PRIMARY KEY,
    device_name VARCHAR(100) NOT NULL,
    location_code CHAR(20),
    value_high_precision NUMERIC(15,8),
    value_low_precision NUMERIC(8,3),
    status_bits BIT(32),
    metadata_bits VARBIT(64)
);