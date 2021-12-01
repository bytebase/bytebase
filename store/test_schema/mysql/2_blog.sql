-- Covered objects:
-- - Table
-- - Index
-- - View
-- - Procedure
-- - Function
-- - Event
-- - Trigger

CREATE DATABASE bytebase_test_blog;

-- Table and Index
CREATE TABLE bytebase_test_blog.author (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(255) COMMENT 'name of the author',
	email VARCHAR(255),
    coin INTEGER DEFAULT 0 COMMENT 'coin can be earned by posting posts, comments'
);

CREATE TABLE bytebase_test_blog.post (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	created_ts BIGINT NOT NULL,
	updated_ts BIGINT NOT NULL,
	author_id INTEGER NOT NULL,
	name VARCHAR(255) COMMENT 'name of the post',
	content TEXT,
	like_count INTEGER DEFAULT 0,
	FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE,
	INDEX (name),
	INDEX (created_ts)
);

CREATE TABLE bytebase_test_blog.comment (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	created_ts BIGINT NOT NULL,
	updated_ts BIGINT NOT NULL,
	post_id INTEGER,
	author_id INTEGER,
	content TEXT,
	FOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,
	FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE,
	INDEX (created_ts)
);

-- View
CREATE VIEW bytebase_test_blog.top_like_post AS
SELECT
	*
FROM
	bytebase_test_blog.post
ORDER BY
	like_count DESC
LIMIT 10;

-- Procedure
CREATE PROCEDURE bytebase_test_blog.author_post_count (IN author_id INT, OUT post_count INT)
BEGIN
	SELECT
		COUNT(*) INTO post_count
		FROM
			author, post
		WHERE
			author.id = post.author_id AND author.id = author_id;
END;

-- Function
CREATE FUNCTION bytebase_test_blog.author_comment_count (author_id INT)
	RETURNS INT DETERMINISTIC
BEGIN
DECLARE
	comment_count INT DEFAULT 0;
	SELECT
		COUNT(*) INTO comment_count
	FROM
		author,
		COMMENT
	WHERE
		author.id = comment.author_id
		AND author.id = author_id;
	RETURN comment_count;
END;

-- Event
CREATE EVENT bytebase_test_blog.increase_author_coin_daily ON SCHEDULE EVERY 1 DAY DO UPDATE author
SET coin = coin + 1;

-- Trigger
CREATE TRIGGER bytebase_test_blog.update_post_updated_ts
	AFTER UPDATE ON post
	FOR EACH ROW
BEGIN
	UPDATE
		post
	SET
		updated_ts = NOW()
	WHERE
		id = OLD.id;
END;
