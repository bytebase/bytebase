CREATE TABLE public.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    old_price DECIMAL(10,2),
    deprecated_field1 TEXT,
    description TEXT,
    deprecated_field2 INTEGER,
    category VARCHAR(50),
    legacy_data JSONB
);

CREATE INDEX idx_products_old_price ON public.products(old_price);
CREATE INDEX idx_products_legacy_data ON public.products USING gin(legacy_data);