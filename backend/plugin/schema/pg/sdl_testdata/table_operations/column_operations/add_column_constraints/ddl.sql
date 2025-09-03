ALTER TABLE "public"."employees" ALTER COLUMN "department" SET NOT NULL;
ALTER TABLE "public"."employees" ALTER COLUMN "email" SET NOT NULL;
ALTER TABLE "public"."employees" ALTER COLUMN "name" SET NOT NULL;
ALTER TABLE "public"."employees" ADD CONSTRAINT "employees_email_key" UNIQUE (email);
ALTER TABLE "public"."employees" ADD CONSTRAINT "employees_age_check" CHECK (age >= 18 AND age <= 100);
ALTER TABLE "public"."employees" ADD CONSTRAINT "employees_salary_check" CHECK (salary > 0);

