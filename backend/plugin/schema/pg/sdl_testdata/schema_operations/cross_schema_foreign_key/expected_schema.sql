CREATE SCHEMA auth;
CREATE SCHEMA app;

CREATE TABLE auth.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE app.posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    author_id INTEGER NOT NULL,
    CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES auth.users(id) ON DELETE CASCADE
);
