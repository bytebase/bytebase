import { test as setup, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { loadTestEnv, saveTestEnv } from "./env";
import { execSql, execSqlScript, getInstancePgPort, querySql } from "./psql";
import { seedTestData } from "./seed-test-data";

const AUTH_FILE = path.join(__dirname, "../../../.auth/state.json");
const SECONDARY_DATABASE_ID = "hr_prod";
const PROD_ENVIRONMENT = "environments/prod";

function loadSampleSeedData(): string {
  const seedDir = path.join(
    __dirname,
    "../../../../backend/component/sample/seed",
  );
  return fs
    .readdirSync(seedDir)
    .filter((file) => file.endsWith(".sql"))
    .sort()
    .map((file) => fs.readFileSync(path.join(seedDir, file), "utf8"))
    .join("\n");
}

// The secondary-database sync poll below allows 60s, plus login, discovery
// and psql seeding before it. Playwright's 30s default test timeout would
// cut that poll off before instance sync can complete on a slower machine,
// so give the setup step an explicit budget that covers its own waits.
setup.setTimeout(180_000);

setup("authenticate and discover", async ({ page }) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);

  // Provision baseline data on top of the project-scoped sample bootstrap:
  // silence the external-URL banner so its wrench-icon button doesn't
  // shadow the editor's admin wrench locator, and create a secondary
  // project so the project-switcher CUJ has an alternative target.
  // The previous demo dump pre-seeded these things; the post-demo
  // bootstrap leaves them off, so we own the seed here.
  await seedTestData(env.api);

  // Browser login for auth cookies
  await page.goto(`${env.baseURL}/auth/signin`);
  if (page.url().includes("/auth")) {
    await page.getByRole("textbox", { name: /email/i }).fill(env.adminEmail);
    await page.getByRole("textbox", { name: /password/i }).fill(env.adminPassword);
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page).not.toHaveURL(/\/auth/, { timeout: 60000 });
  }

  // Dismiss modals + suppress "New version" modal
  await page.keyboard.press("Escape").catch(() => {});
  await page.waitForTimeout(500);
  await page.evaluate(() => {
    localStorage.setItem(
      "bb.release",
      JSON.stringify({
        ignoreRemindModalTillNextRelease: true,
        nextCheckTs: Date.now() + 86400000,
      })
    );
  });

  // The global bootstrap records the project-scoped sample instance because
  // the workspace-level ListInstances collection intentionally omits it.
  const { project, instance, instanceId } = env;

  const { databases } = await env.api.listDatabases(instance);
  const db = databases?.find((item) => item.name.endsWith("/hr_test"));
  if (!db) {
    throw new Error(`Discovery failed: no hr_test database in ${instance}`);
  }

  const database = db.name;
  const databaseId = database.split("/").pop()!;
  if ((db as { project?: string }).project !== project) {
    throw new Error(`Discovery failed: database ${database} is not in ${project}`);
  }

  // Multi-database and multi-stage specs own an explicit second database
  // fixture. The product sample itself remains one project-scoped instance
  // with one hr_test database.
  const pgPort = await getInstancePgPort(env);
  if (
    querySql(
      databaseId,
      pgPort,
      `SELECT 1 FROM pg_database WHERE datname = '${SECONDARY_DATABASE_ID}'`,
    ) !== "1"
  ) {
    execSql(
      databaseId,
      pgPort,
      `CREATE DATABASE ${SECONDARY_DATABASE_ID}`,
    );
  }
  if (
    querySql(
      SECONDARY_DATABASE_ID,
      pgPort,
      "SELECT to_regclass('public.employee')",
    ) !== "employee"
  ) {
    execSqlScript(
      SECONDARY_DATABASE_ID,
      pgPort,
      loadSampleSeedData(),
    );
  }
  // The sample instance is registered with a sync allowlist of just hr_test;
  // the syncer skips any other discovered database. Widen it to include the
  // secondary database BEFORE syncing, or hr_prod is discovered but never
  // persisted and the poll below can never see it.
  await env.api.updateInstanceSyncDatabases(instance, [
    databaseId,
    SECONDARY_DATABASE_ID,
  ]);
  await env.api.syncInstance(instance);
  const secondaryDatabase = `${instance}/databases/${SECONDARY_DATABASE_ID}`;
  await expect
    .poll(
      async () => {
        const result = await env.api.listDatabases(instance);
        return result.databases?.some((item) => item.name === secondaryDatabase);
      },
      { timeout: 60_000, message: `${SECONDARY_DATABASE_ID} should be synced` },
    )
    .toBe(true);
  await env.api.transferDatabaseToProject(secondaryDatabase, project);
  await env.api.updateDatabaseEnvironment(
    secondaryDatabase,
    PROD_ENVIRONMENT,
  );

  saveTestEnv({ ...env, project, instance, instanceId, database, databaseId });

  // Pin the SQL editor's "last viewed project" by visiting the sample
  // project once. Without this pin, gotoHome() in later specs lands on
  // whatever `SQLEditorRouteShell.fallbackToFirstProject` returns —
  // `head(projects)` from a list sorted by created_time DESC. Any spec
  // that creates an additional project (e.g. connection.spec.ts's
  // project-switcher fixture) would silently shift the default project
  // for every later spec, breaking gotoHome-based saved query/sidebar
  // tests that assume `project-sample`. Visiting the URL exercises the
  // real `setProject()` path which writes to localStorage in whatever
  // shape `useLocalStorage` expects — safer than hand-rolling the JSON
  // format.
  const projectId = project.split("/").pop()!;
  await page.goto(
    `${env.baseURL}/sql-editor/projects/${projectId}/instances/${instanceId}/databases/${databaseId}`,
  );
  await page.waitForLoadState("networkidle").catch(() => {});
  await page.waitForTimeout(1500);

  // Save browser auth state (now includes the pinned last-project key)
  const authDir = path.dirname(AUTH_FILE);
  if (!fs.existsSync(authDir)) fs.mkdirSync(authDir, { recursive: true });
  await page.context().storageState({ path: AUTH_FILE });
});
