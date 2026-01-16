CREATE TABLE "public"."employees" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "manager_id" integer,
  CONSTRAINT "employees_pkey" PRIMARY KEY ("id")
);
ALTER TABLE "public"."employees" ADD CONSTRAINT "fk_manager" FOREIGN KEY ("manager_id") REFERENCES "public"."employees"("id") ON DELETE SET NULL;

