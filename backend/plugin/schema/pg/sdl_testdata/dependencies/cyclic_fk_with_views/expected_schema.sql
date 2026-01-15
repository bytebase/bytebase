CREATE TABLE public.authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    featured_book_id INTEGER,
    CONSTRAINT fk_featured_book FOREIGN KEY (featured_book_id) REFERENCES books(id)
);

CREATE TABLE public.books (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    author_id INTEGER NOT NULL,
    publication_year INTEGER,
    CONSTRAINT fk_book_author FOREIGN KEY (author_id) REFERENCES authors(id)
);

CREATE VIEW public.author_book_summary AS
SELECT
    a.id as author_id,
    a.name as author_name,
    b.id as book_id,
    b.title as book_title,
    b.publication_year
FROM authors a
JOIN books b ON a.id = b.author_id;

CREATE VIEW public.prolific_authors AS
SELECT
    author_id,
    author_name,
    COUNT(*) as book_count
FROM author_book_summary
GROUP BY author_id, author_name
HAVING COUNT(*) > 5;
