CREATE TABLE "public"."authors" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "featured_book_id" integer,
  CONSTRAINT "authors_pkey" PRIMARY KEY ("id")
);
CREATE TABLE "public"."books" (
  "id" serial NOT NULL,
  "title" character varying(200) NOT NULL,
  "author_id" integer NOT NULL,
  "publication_year" integer,
  CONSTRAINT "books_pkey" PRIMARY KEY ("id")
);
ALTER TABLE "public"."authors" ADD CONSTRAINT "fk_featured_book" FOREIGN KEY ("featured_book_id") REFERENCES "public"."books"("id");
ALTER TABLE "public"."books" ADD CONSTRAINT "fk_book_author" FOREIGN KEY ("author_id") REFERENCES "public"."authors"("id");
CREATE VIEW "public"."author_book_summary" AS
SELECT
    a.id as author_id,
    a.name as author_name,
    b.id as book_id,
    b.title as book_title,
    b.publication_year
FROM authors a
JOIN books b ON a.id = b.author_id;
CREATE VIEW "public"."prolific_authors" AS
SELECT
    author_id,
    author_name,
    COUNT(*) as book_count
FROM author_book_summary
GROUP BY author_id, author_name
HAVING COUNT(*) > 5;

