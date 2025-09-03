CREATE INDEX idx_posts_author_id ON public.posts USING btree (author_id);
CREATE INDEX idx_posts_author_status ON public.posts USING btree (author_id, status);
CREATE INDEX idx_posts_category ON public.posts USING btree (category);
CREATE INDEX idx_posts_created_at ON public.posts USING btree (created_at);
CREATE INDEX idx_posts_published ON public.posts USING btree (created_at) WHERE ((status)::text = 'published'::text);
CREATE INDEX idx_posts_status ON public.posts USING btree (status);
CREATE INDEX idx_posts_tags ON public.posts USING gin (tags);

