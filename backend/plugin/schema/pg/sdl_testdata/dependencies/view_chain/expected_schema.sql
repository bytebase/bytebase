CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    customer_name VARCHAR(100) NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE VIEW public.active_orders AS
SELECT id, customer_name, total_amount, created_at
FROM orders
WHERE status = 'active';

CREATE VIEW public.high_value_orders AS
SELECT id, customer_name, total_amount, created_at
FROM active_orders
WHERE total_amount > 1000;

CREATE VIEW public.recent_high_value_orders AS
SELECT id, customer_name, total_amount, created_at
FROM high_value_orders
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '30 days';
