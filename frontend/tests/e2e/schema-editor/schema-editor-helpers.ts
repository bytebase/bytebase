import type { BytebaseApiClient } from "../framework/api-client";
import type { TestEnv } from "../framework/env";

// Create a DATABASE_CHANGE plan whose spec targets the Postgres `hr_test`
// sample database, so the plan's Statement section renders the "Schema editor"
// button (schemaEditorEligible = target engine is MySQL/TiDB/Postgres). The
// editor only generates SQL for insertion — it never applies to the DB — so
// creating throwaway tables against hr_test in the editor is side-effect free.
export async function createSchemaEditorPlan(
  env: TestEnv & { api: BytebaseApiClient },
  prefix: string
): Promise<{ projectId: string; planId: string; planName: string }> {
  const pg = await env.api.findDatabaseByShortName("hr_test");
  if (!pg) throw new Error("hr_test Postgres database not found");

  const ts = Date.now();
  const sheet = await env.api.createSheet(
    env.project,
    "-- edit via schema editor\n"
  );
  const plan = await env.api.createPlan(env.project, `${prefix} ${ts}`, [
    { id: `spec-${ts}`, targets: [pg.database], sheet },
  ]);
  return {
    projectId: env.project.split("/").pop()!,
    planId: plan.name.split("/").pop()!,
    planName: plan.name,
  };
}
