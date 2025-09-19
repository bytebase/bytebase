ALTER TABLE "public"."posts" ALTER COLUMN "author_id" SET NOT NULL;
CREATE INDEX idx_posts_author_id ON public.posts USING btree (author_id);
CREATE INDEX idx_posts_category_id ON public.posts USING btree (category_id);

ALTER TABLE "public"."posts" ADD CONSTRAINT "fk_posts_author" FOREIGN KEY ("author_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "public"."posts" ADD CONSTRAINT "fk_posts_category" FOREIGN KEY ("category_id") REFERENCES "public"."categories" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;

