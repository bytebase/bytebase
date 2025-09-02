CREATE SEQUENCE "public"."numeric_types_test_big_serial_col_seq" START WITH 1 INCREMENT BY 1;
CREATE SEQUENCE "public"."numeric_types_test_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."numeric_types_test" (
    "id" integer NOT NULL DEFAULT nextval('public.numeric_types_test_id_seq'::regclass),
    "small_int_col" smallint NOT NULL,
    "int_col" integer DEFAULT 0,
    "big_int_col" bigint,
    "decimal_col" numeric(10,2) NOT NULL,
    "numeric_col" numeric(15,3),
    "numeric_no_scale" numeric(8),
    "real_col" real,
    "double_col" double precision,
    "big_serial_col" bigint NOT NULL DEFAULT nextval('public.numeric_types_test_big_serial_col_seq'::regclass),
    CONSTRAINT "numeric_types_test_check_1" CHECK (small_int_col > 0),
    CONSTRAINT "numeric_types_test_check_2" CHECK (decimal_col >= 0)
);
ALTER TABLE "public"."numeric_types_test" ADD CONSTRAINT "numeric_types_test_pkey" PRIMARY KEY (id);
ALTER TABLE "public"."numeric_types_test" ADD CONSTRAINT "numeric_types_test_int_col_big_int_col_key" UNIQUE (int_col, big_int_col);
CREATE INDEX idx_numeric_decimal ON public.numeric_types_test USING btree (decimal_col);
CREATE INDEX idx_numeric_big_int ON public.numeric_types_test USING btree (big_int_col) WHERE big_int_col IS NOT NULL;
CREATE INDEX idx_numeric_composite ON public.numeric_types_test USING btree (int_col, real_col);

