CREATE TYPE "public"."order_status" AS ENUM ('pending', 'confirmed', 'shipped', 'delivered');
CREATE TABLE "public"."orders" (
  "id" serial NOT NULL,
  "order_number" character varying(50) NOT NULL,
  "status" "public"."order_status" DEFAULT 'pending',
  "created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "orders_pkey" PRIMARY KEY ("id")
);

