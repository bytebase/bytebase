CREATE SEQUENCE "public"."analytics_record_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."analytics" (
    "record_id" bigint NOT NULL DEFAULT nextval('public.analytics_record_id_seq'::regclass),
    "metric_name" character varying(200) NOT NULL,
    "value_small" bigint,
    "value_int" numeric(20,6),
    "description" text,
    "measured_at" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP,
    "precision_value" double precision
);
ALTER TABLE "public"."analytics" ADD CONSTRAINT "analytics_pkey" PRIMARY KEY (record_id);

ALTER TABLE "public"."customers" ALTER COLUMN "credit_limit" TYPE numeric(12,4);
ALTER TABLE "public"."customers" ALTER COLUMN "credit_limit" DROP DEFAULT;
ALTER TABLE "public"."customers" ALTER COLUMN "credit_limit" SET DEFAULT 1000.0000;
ALTER TABLE "public"."customers" ALTER COLUMN "customer_id" TYPE bigint;
ALTER TABLE "public"."customers" ALTER COLUMN "email" TYPE text;
ALTER TABLE "public"."customers" ALTER COLUMN "last_login" TYPE timestamp(6) with time zone;
ALTER TABLE "public"."customers" ALTER COLUMN "signup_date" TYPE timestamp(6) with time zone;
ALTER TABLE "public"."customers" ALTER COLUMN "username" TYPE character varying(50);

ALTER TABLE "public"."products" ALTER COLUMN "created_date" TYPE timestamp(6) without time zone;
ALTER TABLE "public"."products" ALTER COLUMN "created_date" DROP DEFAULT;
ALTER TABLE "public"."products" ALTER COLUMN "created_date" SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."products" ALTER COLUMN "description" TYPE text;
ALTER TABLE "public"."products" ALTER COLUMN "name" TYPE character varying(100);
ALTER TABLE "public"."products" ALTER COLUMN "price" TYPE double precision;
ALTER TABLE "public"."products" ALTER COLUMN "product_id" TYPE bigint;
ALTER TABLE "public"."products" ALTER COLUMN "quantity" TYPE bigint;
ALTER TABLE "public"."products" ALTER COLUMN "rating" TYPE numeric(5,2);
ALTER TABLE "public"."products" ALTER COLUMN "updated_time" TYPE time(6) with time zone;

