CREATE TABLE "public"."departments" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "manager_id" integer,
  CONSTRAINT "departments_pkey" PRIMARY KEY ("id")
);
CREATE TABLE "public"."employees" (
  "id" serial NOT NULL,
  "name" character varying(100) NOT NULL,
  "department_id" integer NOT NULL,
  CONSTRAINT "employees_pkey" PRIMARY KEY ("id")
);
ALTER TABLE "public"."departments" ADD CONSTRAINT "fk_department_manager" FOREIGN KEY ("manager_id") REFERENCES "public"."employees"("id");
ALTER TABLE "public"."employees" ADD CONSTRAINT "fk_employee_department" FOREIGN KEY ("department_id") REFERENCES "public"."departments"("id");

