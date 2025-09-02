-- Initial schema with basic versions of advanced types
CREATE TABLE public.documents (
    doc_id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content BYTEA,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.sessions (
    session_id UUID PRIMARY KEY,
    user_id INTEGER NOT NULL,
    data JSONB,
    expires_at TIMESTAMP NOT NULL
);