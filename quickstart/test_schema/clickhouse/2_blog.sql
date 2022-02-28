-- Covered objects:
-- - Table
-- - Index
-- - View

CREATE DATABASE bytebase_test_blog;

-- Table and Index
CREATE TABLE bytebase_test_blog.author (
    id UUID,
    name String COMMENT 'name of the author',
    email String,
    coin Int32 DEFAULT 0 COMMENT 'coin can be earned by posting posts, comments'
) ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE bytebase_test_blog.post (
    id UUID,
    created_ts Int64 NOT NULL,
    updated_ts Int64 NOT NULL,
    author_id UUID NOT NULL,
    name String comment 'name of the post',
    content String,
    like_count Int32 DEFAULT 0,
    INDEX index_name (name) TYPE minmax GRANULARITY 4,
    INDEX index_created_ts (created_ts) TYPE minmax GRANULARITY 4
) ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE bytebase_test_blog.comment (
    id UUID,
    created_ts Int64 NOT NULL,
    updated_ts Int64 NOT NULL,
    post_id Int32,
    author_id Int32,
    content String,
    INDEX index_created_ts (created_ts) TYPE minmax GRANULARITY 4
) ENGINE = MergeTree()
ORDER BY id;

-- View
CREATE VIEW bytebase_test_blog.top_like_post AS
SELECT
    *
FROM
    bytebase_test_blog.post
ORDER BY
    like_count DESC
LIMIT 10;
