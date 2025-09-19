ALTER TABLE "public"."products" ADD COLUMN "category" character varying(50) DEFAULT 'general';
ALTER TABLE "public"."products" ADD COLUMN "created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."products" ADD COLUMN "description" text;
ALTER TABLE "public"."products" ADD COLUMN "is_featured" boolean DEFAULT false;

