-- Simple table with basic types
CREATE TABLE public.simple_test (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    age INTEGER,
    active BOOLEAN DEFAULT true
);