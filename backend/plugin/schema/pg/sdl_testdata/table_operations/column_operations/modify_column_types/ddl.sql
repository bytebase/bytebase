ALTER TABLE "public"."customers" ALTER COLUMN "age" TYPE integer;
ALTER TABLE "public"."customers" ALTER COLUMN "balance" TYPE bigint;
ALTER TABLE "public"."customers" ALTER COLUMN "is_active" SET DEFAULT true;
ALTER TABLE "public"."customers" ALTER COLUMN "name" TYPE character varying(200);

