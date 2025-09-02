-- Initial schema with various data types
CREATE TABLE public.products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(8,2) NOT NULL,
    category_ids INTEGER[],
    attributes JSONB,
    created_at TIMESTAMP,
    is_featured BOOLEAN DEFAULT false
);

CREATE TABLE public.orders (
    order_id BIGSERIAL PRIMARY KEY,
    customer_email VARCHAR(100) NOT NULL,
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_amount NUMERIC(10,2),
    status VARCHAR(20) DEFAULT 'pending'
);

-- Indexes that may be affected by column changes
CREATE INDEX idx_products_name ON public.products(name);
CREATE INDEX idx_products_price ON public.products(price);
CREATE INDEX idx_products_attributes ON public.products USING GIN(attributes);
CREATE INDEX idx_orders_customer ON public.orders(customer_email);
CREATE INDEX idx_orders_date ON public.orders(order_date);