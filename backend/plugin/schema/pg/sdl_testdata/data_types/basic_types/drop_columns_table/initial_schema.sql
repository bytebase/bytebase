-- Initial schema with multiple tables and various data types
CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    age INTEGER,
    balance DECIMAL(10,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    metadata JSONB
);

CREATE TABLE public.user_sessions (
    session_id UUID PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    session_data JSONB,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table that will be completely dropped
CREATE TABLE public.temp_data (
    temp_id SERIAL PRIMARY KEY,
    temp_value TEXT,
    temp_number NUMERIC(15,5),
    temp_date DATE,
    temp_boolean BOOLEAN DEFAULT false
);

-- Indexes on columns that will be dropped
CREATE INDEX idx_users_first_last ON public.users(first_name, last_name);
CREATE INDEX idx_users_metadata ON public.users USING GIN(metadata);
CREATE INDEX idx_temp_value ON public.temp_data(temp_value);