CREATE SCHEMA sales;
CREATE SCHEMA inventory;

CREATE TABLE sales.customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE
);

CREATE TABLE inventory.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    sku VARCHAR(50) UNIQUE,
    quantity INTEGER DEFAULT 0
);
