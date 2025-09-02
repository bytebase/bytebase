CREATE SEQUENCE "public"."simple_test_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."simple_test" (
    "id" integer NOT NULL DEFAULT nextval('public.simple_test_id_seq'::regclass),
    "name" character varying(50) NOT NULL,
    "age" integer,
    "active" boolean DEFAULT true
);
ALTER TABLE "public"."simple_test" ADD CONSTRAINT "simple_test_pkey" PRIMARY KEY (id);

