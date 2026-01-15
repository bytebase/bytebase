CREATE TYPE public.order_status AS ENUM (
    'pending',
    'confirmed',
    'shipped',
    'delivered'
);

CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) NOT NULL,
    status order_status DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
