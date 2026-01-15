CREATE TABLE "public"."orders" (
  "id" serial NOT NULL,
  "customer_name" character varying(100) NOT NULL,
  "total_amount" numeric(10,2) NOT NULL,
  "status" character varying(20) NOT NULL,
  "created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "orders_pkey" PRIMARY KEY ("id")
);
CREATE VIEW "public"."active_orders" AS
SELECT id, customer_name, total_amount, created_at
FROM orders
WHERE status = 'active';
CREATE VIEW "public"."high_value_orders" AS
SELECT id, customer_name, total_amount, created_at
FROM active_orders
WHERE total_amount > 1000;
CREATE VIEW "public"."recent_high_value_orders" AS
SELECT id, customer_name, total_amount, created_at
FROM high_value_orders
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '30 days';

