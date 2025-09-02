-- Initial schema with types that can be safely converted
CREATE TABLE public.products (
    product_id SMALLINT PRIMARY KEY,
    name CHAR(50) NOT NULL,
    description VARCHAR(200),
    price REAL NOT NULL,
    quantity INTEGER DEFAULT 0,
    rating NUMERIC(3,1),
    created_date DATE DEFAULT CURRENT_DATE,
    updated_time TIME DEFAULT CURRENT_TIME
);

CREATE TABLE public.customers (
    customer_id INTEGER PRIMARY KEY,
    username VARCHAR(30) UNIQUE NOT NULL,
    email VARCHAR(100) NOT NULL,
    credit_limit NUMERIC(8,2) DEFAULT 1000.00,
    signup_date DATE NOT NULL,
    last_login TIMESTAMP
);