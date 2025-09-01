-- Initial schema with basic array types
CREATE TABLE public.products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    tags TEXT[],
    category_ids INTEGER[],
    prices NUMERIC(10,2)[]
);

CREATE TABLE public.events (
    event_id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    participant_ids UUID[],
    event_dates TIMESTAMP[]
);