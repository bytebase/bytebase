-- Covered objects:
-- - Table
-- - Index

CREATE DATABASE bytebase_test_todo;

-- Table and Index
CREATE TABLE bytebase_test_todo.author (
    id UUID,
    name String
) ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE bytebase_test_todo.todo (
    id UUID,
    name String COMMENT 'name of the todo',
    content String,
    author_id UUID,
    created_ts Int64 NOT NULL,
    updated_ts Int64 NOT NULL,
    INDEX index_name (name) TYPE minmax GRANULARITY 4,
    INDEX index_created_ts (created_ts) TYPE minmax GRANULARITY 4
) ENGINE = MergeTree()
ORDER BY id;
