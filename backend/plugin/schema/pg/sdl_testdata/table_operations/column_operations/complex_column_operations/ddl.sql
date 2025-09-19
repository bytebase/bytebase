DROP INDEX IF EXISTS "public"."idx_products_category_name";
DROP INDEX IF EXISTS "public"."idx_products_temp";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "old_name";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "temp_field";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "category_name";

ALTER TABLE "public"."products" ADD COLUMN "category_id" integer NOT NULL;
ALTER TABLE "public"."products" ADD COLUMN "created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."products" ADD COLUMN "description" text;
ALTER TABLE "public"."products" ADD COLUMN "is_active" boolean DEFAULT true;
ALTER TABLE "public"."products" ADD COLUMN "name" character varying(200) NOT NULL;
ALTER TABLE "public"."products" ADD COLUMN "updated_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."products" ALTER COLUMN "price" TYPE numeric(10,2);
ALTER TABLE "public"."products" ALTER COLUMN "price" SET NOT NULL;
CREATE INDEX idx_products_active ON public.products USING btree (is_active) WHERE is_active = true;
CREATE INDEX idx_products_category_id ON public.products USING btree (category_id);
CREATE INDEX idx_products_name ON public.products USING btree (name);
CREATE INDEX idx_products_price ON public.products USING btree (price);
ALTER TABLE "public"."products" ADD CONSTRAINT "products_price_check" CHECK (price > 0);

ALTER TABLE "public"."products" ADD CONSTRAINT "fk_products_category" FOREIGN KEY ("category_id") REFERENCES "public"."categories" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;

