CREATE SCHEMA "app";
CREATE SCHEMA "auth";
CREATE TABLE "auth"."users" (
  "id" serial NOT NULL,
  "username" character varying(50) NOT NULL,
  "email" character varying(255) NOT NULL,
  CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "users_username_key" UNIQUE ("username"),
  CONSTRAINT "users_email_key" UNIQUE ("email")
);
CREATE TABLE "app"."posts" (
  "id" serial NOT NULL,
  "title" character varying(200) NOT NULL,
  "content" text,
  "author_id" integer NOT NULL,
  CONSTRAINT "posts_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "fk_author" FOREIGN KEY ("author_id") REFERENCES "auth"."users"("id") ON DELETE CASCADE
);

