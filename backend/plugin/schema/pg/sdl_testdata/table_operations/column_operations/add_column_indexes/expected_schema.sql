CREATE TABLE public.posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    author_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    category VARCHAR(50),
    tags TEXT[],
    status VARCHAR(20) DEFAULT 'draft'
);

CREATE INDEX idx_posts_author_id ON public.posts(author_id);
CREATE INDEX idx_posts_created_at ON public.posts(created_at);
CREATE INDEX idx_posts_category ON public.posts(category);
CREATE INDEX idx_posts_status ON public.posts(status);
CREATE INDEX idx_posts_author_status ON public.posts(author_id, status);
CREATE INDEX idx_posts_published ON public.posts(created_at) WHERE ((status)::text = 'published'::text);
CREATE INDEX idx_posts_tags ON public.posts USING gin(tags);