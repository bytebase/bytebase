CREATE SEQUENCE "public"."datetime_types_test_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."datetime_types_test" (
    "id" integer NOT NULL DEFAULT nextval('public.datetime_types_test_id_seq'::regclass),
    "date_col" date DEFAULT CURRENT_DATE,
    "date_birth" date NOT NULL,
    "time_col" time(6) without time zone,
    "time_with_tz" time(6) with time zone,
    "time_precise" time(6) without time zone DEFAULT CURRENT_TIME,
    "timestamp_col" timestamp(6) without time zone,
    "timestamp_with_tz" timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP,
    "timestamp_created" timestamp(6) without time zone DEFAULT NOW(),
    "timestamp_precise" timestamp(3) without time zone NOT NULL,
    "interval_col" interval,
    "interval_default" interval DEFAULT '1 day',
    CONSTRAINT "datetime_types_test_check_1" CHECK (date_birth <= CURRENT_DATE),
    CONSTRAINT "datetime_types_test_check_2" CHECK (timestamp_precise >= '2020-01-01'::timestamp),
    CONSTRAINT "datetime_types_test_check_3" CHECK (interval_col IS NULL OR interval_col >= '0 seconds'::interval)
);
ALTER TABLE "public"."datetime_types_test" ADD CONSTRAINT "datetime_types_test_pkey" PRIMARY KEY (id);
CREATE INDEX idx_datetime_date ON public.datetime_types_test USING btree (date_col);
CREATE INDEX idx_datetime_timestamp_tz ON public.datetime_types_test USING btree (timestamp_with_tz);
CREATE INDEX idx_datetime_created_desc ON public.datetime_types_test USING btree (timestamp_created DESC);
CREATE INDEX idx_datetime_birth_year ON public.datetime_types_test USING btree (EXTRACT(YEAR FROM date_birth));

