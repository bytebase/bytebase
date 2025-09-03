DROP INDEX IF EXISTS "public"."idx_products_legacy_data";
DROP INDEX IF EXISTS "public"."idx_products_old_price";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "old_price";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "deprecated_field1";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "deprecated_field2";
ALTER TABLE "public"."products" DROP COLUMN IF EXISTS "legacy_data";

