CREATE TABLE public.customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    age SMALLINT,
    balance INTEGER,
    is_active BOOLEAN,
    notes TEXT
);