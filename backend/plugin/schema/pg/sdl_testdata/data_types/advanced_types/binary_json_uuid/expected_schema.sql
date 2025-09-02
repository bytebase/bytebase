-- Enhanced schema with full advanced type usage
CREATE TABLE public.documents (
    doc_id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content BYTEA,
    metadata JSONB,                    -- Changed from JSON to JSONB
    checksum BYTEA,                    -- New binary column
    file_info JSONB DEFAULT '{}',      -- New JSONB with default
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE public.sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Added default UUID generation
    user_id INTEGER NOT NULL,
    data JSONB,
    settings JSONB DEFAULT '{"theme": "default"}'::jsonb, -- New JSONB with default
    expires_at TIMESTAMP NOT NULL,
    last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP     -- New timestamp column
);

CREATE TABLE public.files (
    file_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    content BYTEA NOT NULL,
    mime_type VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    size_bytes INTEGER,
    upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for JSON queries
CREATE INDEX idx_documents_metadata ON public.documents USING GIN(metadata);
CREATE INDEX idx_sessions_data ON public.sessions USING GIN(data);
CREATE INDEX idx_sessions_settings ON public.sessions USING GIN(settings);
CREATE INDEX idx_files_metadata ON public.files USING GIN(metadata);
CREATE INDEX idx_files_mime_type ON public.files(mime_type);