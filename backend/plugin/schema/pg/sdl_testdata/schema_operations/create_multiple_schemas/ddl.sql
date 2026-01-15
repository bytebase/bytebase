CREATE SCHEMA "inventory";
CREATE SCHEMA "sales";
CREATE TABLE "inventory"."products" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "sku" character varying(50),
  "quantity" integer DEFAULT 0,
  CONSTRAINT "products_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "products_sku_key" UNIQUE ("sku")
);
CREATE TABLE "sales"."customers" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "email" character varying(255),
  CONSTRAINT "customers_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "customers_email_key" UNIQUE ("email")
);

