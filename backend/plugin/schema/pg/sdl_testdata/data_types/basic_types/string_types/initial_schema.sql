-- Basic table with some string types
CREATE TABLE public.basic_strings (
    id SERIAL PRIMARY KEY,
    short_name VARCHAR(20) NOT NULL,
    description TEXT
);