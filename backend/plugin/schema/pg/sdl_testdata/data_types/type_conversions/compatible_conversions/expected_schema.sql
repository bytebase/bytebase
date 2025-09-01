-- Enhanced schema with compatible type promotions
CREATE TABLE public.products (
    product_id BIGINT PRIMARY KEY,                 -- SMALLINT → BIGINT
    name VARCHAR(100) NOT NULL,                    -- CHAR(50) → VARCHAR(100)
    description TEXT,                              -- VARCHAR(200) → TEXT
    price DOUBLE PRECISION NOT NULL,               -- REAL → DOUBLE PRECISION
    quantity BIGINT DEFAULT 0,                     -- INTEGER → BIGINT
    rating NUMERIC(5,2),                          -- NUMERIC(3,1) → NUMERIC(5,2) expanded precision
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- DATE → TIMESTAMP
    updated_time TIME WITH TIME ZONE DEFAULT CURRENT_TIME -- TIME → TIME WITH TIME ZONE
);

CREATE TABLE public.customers (
    customer_id BIGINT PRIMARY KEY,               -- INTEGER → BIGINT
    username VARCHAR(50) UNIQUE NOT NULL,        -- VARCHAR(30) → VARCHAR(50)
    email TEXT NOT NULL,                         -- VARCHAR(100) → TEXT
    credit_limit NUMERIC(12,4) DEFAULT 1000.0000, -- NUMERIC(8,2) → NUMERIC(12,4)
    signup_date TIMESTAMP WITH TIME ZONE NOT NULL, -- DATE → TIMESTAMP WITH TIME ZONE
    last_login TIMESTAMP WITH TIME ZONE          -- TIMESTAMP → TIMESTAMP WITH TIME ZONE
);

CREATE TABLE public.analytics (
    record_id BIGSERIAL PRIMARY KEY,
    metric_name VARCHAR(200) NOT NULL,
    value_small BIGINT,                          -- For SMALLINT → BIGINT conversion
    value_int NUMERIC(20,6),                     -- For INTEGER → NUMERIC conversion
    description TEXT,                            -- For various string → TEXT conversions
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    precision_value DOUBLE PRECISION             -- For REAL → DOUBLE PRECISION
);