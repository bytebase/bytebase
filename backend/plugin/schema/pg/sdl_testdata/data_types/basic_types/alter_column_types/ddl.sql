ALTER TABLE "public"."users" ADD COLUMN "last_login" timestamp(6) without time zone;
ALTER TABLE "public"."users" ADD COLUMN "middle_name" character varying(50);
ALTER TABLE "public"."users" ALTER COLUMN "age" TYPE bigint;
ALTER TABLE "public"."users" ALTER COLUMN "balance" TYPE numeric(12,4);
ALTER TABLE "public"."users" ALTER COLUMN "balance" DROP DEFAULT;
ALTER TABLE "public"."users" ALTER COLUMN "balance" SET DEFAULT 0.0000;
ALTER TABLE "public"."users" ALTER COLUMN "email" SET NOT NULL;
ALTER TABLE "public"."users" ALTER COLUMN "username" TYPE character varying(100);

