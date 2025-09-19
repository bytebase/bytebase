ALTER TABLE "public"."orders" ADD COLUMN "customer_phone" character varying(20);
ALTER TABLE "public"."orders" ADD COLUMN "notes" text;
ALTER TABLE "public"."orders" ADD COLUMN "shipping_address" jsonb;
ALTER TABLE "public"."orders" ADD COLUMN "tax_amount" numeric(15,4) DEFAULT 0.0000;
ALTER TABLE "public"."orders" ALTER COLUMN "customer_email" TYPE character varying(150);
ALTER TABLE "public"."orders" ALTER COLUMN "order_date" TYPE timestamp(6) with time zone;
ALTER TABLE "public"."orders" ALTER COLUMN "order_date" DROP DEFAULT;
ALTER TABLE "public"."orders" ALTER COLUMN "order_date" SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."orders" ALTER COLUMN "status" TYPE character varying(30);
ALTER TABLE "public"."orders" ALTER COLUMN "status" DROP DEFAULT;
ALTER TABLE "public"."orders" ALTER COLUMN "status" SET DEFAULT 'pending';
ALTER TABLE "public"."orders" ALTER COLUMN "total_amount" TYPE numeric(15,4);
CREATE INDEX idx_orders_status ON public.orders USING btree (status);

ALTER TABLE "public"."products" ADD COLUMN "dimensions" jsonb;
ALTER TABLE "public"."products" ADD COLUMN "tags" _text;
ALTER TABLE "public"."products" ADD COLUMN "updated_at" timestamp(6) with time zone;
ALTER TABLE "public"."products" ADD COLUMN "weight" numeric(6,3);
ALTER TABLE "public"."products" ALTER COLUMN "category_ids" TYPE _int8;
ALTER TABLE "public"."products" ALTER COLUMN "created_at" TYPE timestamp(6) with time zone;
ALTER TABLE "public"."products" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."products" ALTER COLUMN "name" TYPE character varying(200);
ALTER TABLE "public"."products" ALTER COLUMN "price" TYPE numeric(12,4);
CREATE INDEX idx_products_tags ON public.products USING gin (tags);
CREATE INDEX idx_products_weight ON public.products USING btree (weight);

