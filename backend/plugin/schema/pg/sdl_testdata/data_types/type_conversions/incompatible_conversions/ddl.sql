ALTER TABLE "public"."data_conflicts" ALTER COLUMN "default_change" DROP DEFAULT;
ALTER TABLE "public"."data_conflicts" ALTER COLUMN "default_change" SET DEFAULT 999;
ALTER TABLE "public"."data_conflicts" ALTER COLUMN "nullable_required" SET NOT NULL;
ALTER TABLE "public"."data_conflicts" ALTER COLUMN "type_change" TYPE integer USING "type_change"::integer;
ALTER TABLE "public"."data_conflicts" ALTER COLUMN "wide_varchar" TYPE character varying(10) USING "wide_varchar"::character varying(10);
ALTER TABLE "public"."data_conflicts" ADD CONSTRAINT "data_conflicts_unique_field_key" UNIQUE (unique_field);

ALTER TABLE "public"."risky_conversions" ALTER COLUMN "array_field" TYPE text;
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "high_precision" TYPE real USING "high_precision"::real;
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "json_field" TYPE text USING "json_field"::text;
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "large_int" TYPE smallint USING "large_int"::smallint;
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "numeric_text" TYPE integer USING "numeric_text"::integer;
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "text_to_numeric" TYPE numeric(10,2) USING "text_to_numeric"::numeric(10,2);
ALTER TABLE "public"."risky_conversions" ALTER COLUMN "timestamp_field" TYPE date;

