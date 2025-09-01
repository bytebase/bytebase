-- Initial table with basic column types
CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100),
    age INTEGER,
    balance DECIMAL(10,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sample data constraints
CREATE UNIQUE INDEX idx_users_username ON public.users(username);
CREATE INDEX idx_users_email ON public.users(email);