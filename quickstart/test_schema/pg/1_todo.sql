-- Covered objects:
-- - Sequence
-- - Table
-- - Index

CREATE DATABASE bytebase_test_todo;

-- Reconnect to bytebase_test_todo before running following sql
-- Sequence, Table and Index
CREATE TABLE public.author (
	id SERIAL PRIMARY KEY,
	name VARCHAR(255)
);

COMMENT ON COLUMN public.author.name is 'name of the author';

CREATE TABLE public.todo (
	id SERIAL PRIMARY KEY,
	name VARCHAR(255) COMMENT 'name of the todo',
	content TEXT,
	author_id INTEGER,
	created_ts BIGINT NOT NULL,
	updated_ts BIGINT NOT NULL,
	FOREIGN KEY (author_id) REFERENCES author (id) ON DELETE CASCADE
);

CREATE INDEX bytebase_test_todo_todo_name ON public.todo(name);
CREATE INDEX bytebase_test_todo_todo_created_ts ON public.todo(created_ts);

COMMENT ON INDEX bytebase_test_todo_todo_name is 'index on todo.name';
COMMENT ON INDEX bytebase_test_todo_todo_created_ts is 'index on todo.created_ts';
