-- Enhanced schema with comprehensive array type usage
CREATE TABLE public.products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    tags TEXT[] DEFAULT ARRAY[]::TEXT[],        -- Added default empty array
    category_ids BIGINT[],                      -- Changed from INTEGER[] to BIGINT[]
    prices NUMERIC(12,4)[],                     -- Increased precision
    variants TEXT[][],                           -- New multidimensional array
    ratings INTEGER[] DEFAULT ARRAY[0,0,0,0,0], -- New array with default values
    features JSONB[]                            -- New JSONB array
);

CREATE TABLE public.events (
    event_id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    participant_ids UUID[],
    event_dates TIMESTAMP WITH TIME ZONE[],     -- Added timezone
    locations TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[], -- New required array with default
    capacities INTEGER[],                       -- New integer array
    schedules TIMESTAMP[][],                     -- New multidimensional timestamp array
    metadata JSONB[] DEFAULT ARRAY[]::JSONB[]   -- New JSONB array with default
);

CREATE TABLE public.analytics (
    analysis_id SERIAL PRIMARY KEY,
    data_points NUMERIC(15,6)[] NOT NULL,
    categories TEXT[] NOT NULL,
    timestamps TIMESTAMP WITH TIME ZONE[] NOT NULL,
    user_ids UUID[],
    scores REAL[] DEFAULT ARRAY[]::REAL[],
    flags BOOLEAN[] DEFAULT ARRAY[]::BOOLEAN[]
);

-- Add GIN indexes for array operations
CREATE INDEX idx_products_tags ON public.products USING GIN(tags);
CREATE INDEX idx_products_categories ON public.products USING GIN(category_ids);
CREATE INDEX idx_events_participants ON public.events USING GIN(participant_ids);
CREATE INDEX idx_events_locations ON public.events USING GIN(locations);
CREATE INDEX idx_analytics_categories ON public.analytics USING GIN(categories);