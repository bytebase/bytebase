DROP TABLE IF EXISTS "public"."temp_data";
DROP INDEX IF EXISTS "public"."idx_users_first_last";
DROP INDEX IF EXISTS "public"."idx_users_metadata";
ALTER TABLE "public"."users" DROP COLUMN IF EXISTS "first_name";
ALTER TABLE "public"."users" DROP COLUMN IF EXISTS "last_name";
ALTER TABLE "public"."users" DROP COLUMN IF EXISTS "metadata";
DROP SEQUENCE IF EXISTS "public"."temp_data_temp_id_seq";

