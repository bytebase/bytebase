-- Modified table with altered column types  
CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,          -- Keep SERIAL to avoid sequence dependency issues
    username VARCHAR(100) NOT NULL, -- Expanded from VARCHAR(50) to VARCHAR(100)
    email VARCHAR(100) NOT NULL,    -- Changed from nullable to NOT NULL
    age BIGINT,                     -- Changed from INTEGER to BIGINT
    balance DECIMAL(12,4) DEFAULT 0.0000, -- Expanded precision from (10,2) to (12,4)
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,           -- New nullable column
    middle_name VARCHAR(50)         -- New nullable column
);

-- Keep existing indexes
CREATE UNIQUE INDEX idx_users_username ON public.users(username);
CREATE INDEX idx_users_email ON public.users(email);