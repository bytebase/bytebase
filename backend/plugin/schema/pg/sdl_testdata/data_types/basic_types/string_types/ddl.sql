CREATE SEQUENCE "public"."string_types_test_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."string_types_test" (
    "id" integer NOT NULL DEFAULT nextval('public.string_types_test_id_seq'::regclass),
    "char_col" character(10),
    "char_default" character(1) DEFAULT 'A',
    "varchar_short" character varying(50) NOT NULL,
    "varchar_long" character varying(500),
    "varchar_no_limit" character varying,
    "text_col" text,
    "text_with_default" text DEFAULT 'default text',
    CONSTRAINT "string_types_test_check_1" CHECK (LENGTH(text_col) > 0 OR text_col IS NULL)
);
ALTER TABLE "public"."string_types_test" ADD CONSTRAINT "string_types_test_pkey" PRIMARY KEY (id);
ALTER TABLE "public"."string_types_test" ADD CONSTRAINT "string_types_test_varchar_short_key" UNIQUE (varchar_short);
CREATE INDEX idx_string_varchar_short ON public.string_types_test USING btree (varchar_short);
CREATE INDEX idx_string_text_prefix ON public.string_types_test USING btree (LEFT(text_col, 50)) WHERE text_col IS NOT NULL;
CREATE INDEX idx_string_char_col ON public.string_types_test USING btree (char_col) WHERE char_col IS NOT NULL;

