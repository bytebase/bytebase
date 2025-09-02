-- Expected schema with complex alterations applied
CREATE TABLE public.products (
    product_id SERIAL PRIMARY KEY, -- Keep SERIAL to avoid sequence issues
    name VARCHAR(200) NOT NULL,     -- Expanded from VARCHAR(100) to VARCHAR(200)
    description TEXT,
    price DECIMAL(12,4) NOT NULL,   -- Expanded from DECIMAL(8,2) to DECIMAL(12,4)
    category_ids BIGINT[],          -- Changed from INTEGER[] to BIGINT[]
    attributes JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, -- Added timezone and default
    updated_at TIMESTAMP WITH TIME ZONE, -- New column
    is_featured BOOLEAN DEFAULT false,
    weight DECIMAL(6,3),            -- New column for product weight
    dimensions JSONB,               -- New column for product dimensions
    tags TEXT[]                     -- New column for product tags
);

CREATE TABLE public.orders (
    order_id BIGSERIAL PRIMARY KEY, -- Keep BIGSERIAL to avoid sequence issues
    customer_email VARCHAR(150) NOT NULL, -- Expanded from VARCHAR(100)
    customer_phone VARCHAR(20),    -- New nullable column
    order_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Changed from DATE to TIMESTAMPTZ
    total_amount NUMERIC(15,4),     -- Expanded precision from NUMERIC(10,2)
    tax_amount NUMERIC(15,4) DEFAULT 0.0000, -- New column
    status VARCHAR(30) DEFAULT 'pending', -- Expanded from VARCHAR(20)
    shipping_address JSONB,         -- New column for shipping details
    notes TEXT                      -- New column for order notes
);

-- Keep existing indexes with potentially updated column types
CREATE INDEX idx_products_name ON public.products(name);
CREATE INDEX idx_products_price ON public.products(price);
CREATE INDEX idx_products_attributes ON public.products USING GIN(attributes);
CREATE INDEX idx_orders_customer ON public.orders(customer_email);
CREATE INDEX idx_orders_date ON public.orders(order_date);

-- New indexes for new columns
CREATE INDEX idx_products_weight ON public.products(weight);
CREATE INDEX idx_products_tags ON public.products USING GIN(tags);
CREATE INDEX idx_orders_status ON public.orders(status);