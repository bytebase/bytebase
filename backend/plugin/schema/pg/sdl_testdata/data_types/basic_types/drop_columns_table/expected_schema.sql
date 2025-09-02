-- Expected schema after dropping columns and table
CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    -- first_name and last_name columns dropped
    age INTEGER,
    balance DECIMAL(10,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
    -- metadata column dropped (along with its GIN index)
);

CREATE TABLE public.user_sessions (
    session_id UUID PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    session_data JSONB,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- temp_data table completely dropped
-- All related indexes and sequences should be cleaned up automatically