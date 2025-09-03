CREATE TABLE public.products (
    id SERIAL PRIMARY KEY,
    old_name VARCHAR(50),
    price INTEGER,
    temp_field TEXT,
    category_name VARCHAR(30)
);

CREATE TABLE public.categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE INDEX idx_products_temp ON public.products(temp_field);
CREATE INDEX idx_products_category_name ON public.products(category_name);