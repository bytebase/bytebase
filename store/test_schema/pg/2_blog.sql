-- Covered objects:
-- - Sequence
-- - Table
-- - Index
-- - View
-- - Procedure
-- - Function
-- - Trigger
-- - Event Trigger

CREATE DATABASE bytebase_test_blog;

-- Reconnect to bytebase_test_todo before running following sql
-- Sequence, Table and Index
CREATE TABLE public.author (
	id SERIAL PRIMARY KEY,
	name VARCHAR(255),
	email VARCHAR(255),
    coin INTEGER DEFAULT 0
);

COMMENT ON COLUMN public.author.coin is 'coin can be earned by posting posts, comments';

CREATE TABLE public.post (
	id SERIAL PRIMARY KEY,
	created_ts BIGINT NOT NULL,
	updated_ts BIGINT NOT NULL,
	author_id INTEGER NOT NULL,
	name VARCHAR(255),
	content TEXT,
	like_count INTEGER DEFAULT 0,
	FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE
);

CREATE INDEX bytebase_test_blog_post_name ON public.post(name);
CREATE INDEX bytebase_test_blog_post_created_ts ON public.post(created_ts);

COMMENT ON COLUMN public.post.name is 'name of the post';
COMMENT ON INDEX bytebase_test_blog_post_name is 'name index of post';

CREATE TABLE public.comment (
	id SERIAL PRIMARY KEY,
	created_ts BIGINT NOT NULL,
	updated_ts BIGINT NOT NULL,
	post_id INTEGER,
	author_id INTEGER,
	content TEXT,
	FOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,
	FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE
);

CREATE INDEX bytebase_test_blog_comment_created_ts ON public.comment(created_ts);

-- View
CREATE VIEW public.top_like_post AS
SELECT
	*
FROM
	public.post
ORDER BY
	like_count DESC
LIMIT 10;

COMMENT ON VIEW top_like_post is 'view to select top 10 most liked posts';

-- Procedure
CREATE PROCEDURE public.add_coin (author_id INT, count INT)
LANGUAGE SQL
AS $$
	UPDATE
		author SET
			coin = coin + count
		WHERE
			author.id = author_id;
$$;

-- Function
CREATE FUNCTION public.author_post_count (author_id INT)
	RETURNS BIGINT
	AS 'SELECT
			COUNT(*)
  		FROM
  			author, post
  		WHERE
  			author.id = post.author_id AND author.id = author_id;'
	LANGUAGE SQL
	IMMUTABLE
		RETURNS NULL ON NULL INPUT;

-- Trigger
CREATE OR REPLACE FUNCTION update_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_ts = now();
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_post_updated_ts
	BEFORE UPDATE ON post
	FOR EACH ROW
	EXECUTE PROCEDURE update_updated_ts ();

-- Event Trigger
CREATE OR REPLACE FUNCTION abort_any_command()
  RETURNS event_trigger
 LANGUAGE plpgsql
  AS $$
BEGIN
  RAISE EXCEPTION 'command % is disabled', tg_tag;
END;
$$;

CREATE EVENT TRIGGER abort_drop ON sql_drop
   EXECUTE FUNCTION abort_any_command();
