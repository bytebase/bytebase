CREATE SEQUENCE "public"."precision_test_test_id_seq" START WITH 1 INCREMENT BY 1;
ALTER TABLE "public"."data_samples" ALTER COLUMN "bit_flags" TYPE bit(32);
ALTER TABLE "public"."data_samples" ALTER COLUMN "bit_flags" DROP DEFAULT;
ALTER TABLE "public"."data_samples" ALTER COLUMN "bit_flags" SET DEFAULT B'00000000000000000000000000000000';
ALTER TABLE "public"."data_samples" ALTER COLUMN "fixed_char" TYPE character(20);
ALTER TABLE "public"."data_samples" ALTER COLUMN "long_text" TYPE character varying(150);
ALTER TABLE "public"."data_samples" ALTER COLUMN "precise_decimal" TYPE numeric(16,6);
ALTER TABLE "public"."data_samples" ALTER COLUMN "precise_decimal" DROP DEFAULT;
ALTER TABLE "public"."data_samples" ALTER COLUMN "precise_decimal" SET DEFAULT 0.000000;
ALTER TABLE "public"."data_samples" ALTER COLUMN "short_text" TYPE character varying(100);
ALTER TABLE "public"."data_samples" ALTER COLUMN "simple_decimal" TYPE numeric(5,1);
ALTER TABLE "public"."data_samples" ALTER COLUMN "simple_decimal" DROP DEFAULT;
ALTER TABLE "public"."data_samples" ALTER COLUMN "simple_decimal" SET DEFAULT 0.0;
ALTER TABLE "public"."data_samples" ALTER COLUMN "var_bits" TYPE bit varying(16);

ALTER TABLE "public"."measurements" ALTER COLUMN "device_name" TYPE character varying(150);
ALTER TABLE "public"."measurements" ALTER COLUMN "location_code" TYPE character(15);
ALTER TABLE "public"."measurements" ALTER COLUMN "metadata_bits" TYPE bit varying(32);
ALTER TABLE "public"."measurements" ALTER COLUMN "status_bits" TYPE bit(64);
ALTER TABLE "public"."measurements" ALTER COLUMN "value_high_precision" TYPE numeric(20,12);
ALTER TABLE "public"."measurements" ALTER COLUMN "value_low_precision" TYPE numeric(6,2);

CREATE TABLE "public"."precision_test" (
    "test_id" integer NOT NULL DEFAULT nextval('public.precision_test_test_id_seq'::regclass),
    "expand_varchar" character varying(500),
    "reduce_varchar" character varying(25),
    "expand_numeric" numeric(25,10),
    "reduce_numeric" numeric(4,1),
    "expand_char" character(50),
    "reduce_char" character(5),
    "expand_bits" bit(128),
    "reduce_bits" bit(4),
    "expand_varbits" bit varying(256),
    "reduce_varbits" bit varying(8)
);
ALTER TABLE "public"."precision_test" ADD CONSTRAINT "precision_test_pkey" PRIMARY KEY (test_id);

