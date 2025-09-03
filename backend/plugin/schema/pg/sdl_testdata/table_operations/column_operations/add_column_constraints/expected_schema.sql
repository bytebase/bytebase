CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) NOT NULL UNIQUE,
    age INTEGER CHECK (age >= 18 AND age <= 100),
    salary DECIMAL(10,2) CHECK (salary > 0),
    department VARCHAR(50) NOT NULL
);