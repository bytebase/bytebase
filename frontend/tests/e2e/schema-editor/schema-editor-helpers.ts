import type { Page } from "@playwright/test";
import type { BytebaseApiClient } from "../framework/api-client";
import type { TestEnv } from "../framework/env";
import { createDatabaseChangePlanViaUI } from "../framework/ui-create-plan";

// Create a DATABASE_CHANGE plan whose spec targets the Postgres `hr_test`
// sample database, so the plan's Statement section renders the "Schema editor"
// button (schemaEditorEligible = target engine is MySQL/TiDB/Postgres). The
// editor only generates SQL for insertion — it never applies to the DB — so
// creating throwaway tables against hr_test in the editor is side-effect free.
//
// Created through the UI (plan + draft review issue together), the same way a
// user starts a change — no bare api.createPlan. The plan stays a draft (not
// submitted): the schema editor operates on the spec while the plan is a draft.
export async function createSchemaEditorPlan(
  env: TestEnv & { api: BytebaseApiClient },
  page: Page,
  prefix: string
): Promise<{ projectId: string; planId: string; planName: string }> {
  const pg = await env.api.findDatabaseByShortName("hr_test", env.project);
  if (!pg) throw new Error("hr_test Postgres database not found");

  const projectId = env.project.split("/").pop()!;
  const { planId } = await createDatabaseChangePlanViaUI(page, {
    baseURL: env.baseURL,
    projectId,
    database: pg.database,
    title: `${prefix} ${Date.now()}`,
    sql: "SELECT 1;",
  });
  return {
    projectId,
    planId,
    planName: `${env.project}/plans/${planId}`,
  };
}
