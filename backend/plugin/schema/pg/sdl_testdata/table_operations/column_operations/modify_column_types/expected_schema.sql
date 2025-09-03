CREATE TABLE public.customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    age INTEGER,
    balance BIGINT,
    is_active BOOLEAN DEFAULT true,
    notes TEXT
);