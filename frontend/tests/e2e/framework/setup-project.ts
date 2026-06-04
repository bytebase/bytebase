import { test as setup, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { loadTestEnv, saveTestEnv } from "./env";
import { seedTestData } from "./seed-test-data";

const AUTH_FILE = path.join(__dirname, "../../../.auth/state.json");

setup("authenticate and discover", async ({ page }) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);

  // Provision baseline data on top of the new bootstrap's setupSample:
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

  // Discovery: find first Postgres instance, database, and project
  const { instances } = await env.api.listInstances();
  const pgInstance = instances?.find(
    (i: { engine: string; name: string }) =>
      i.engine === "POSTGRES" &&
      !i.name.includes("deleted") &&
      !i.name.includes("bytebase-meta")
  );
  if (!pgInstance) {
    throw new Error("Discovery failed: no Postgres instance found");
  }

  const instance = pgInstance.name;
  const instanceId = instance.split("/").pop()!;

  const { databases } = await env.api.listDatabases(instance);
  const db = databases?.find(
    (d: { name: string }) =>
      !d.name.includes("/postgres") &&
      !d.name.includes("/template") &&
      !d.name.includes("/bbdev")
  ) ?? databases?.[0];
  if (!db) {
    throw new Error(`Discovery failed: no suitable database in ${instance}`);
  }

  const database = db.name;
  const databaseId = database.split("/").pop()!;
  const project = (db as { project?: string }).project ?? "";

  saveTestEnv({ ...env, project, instance, instanceId, database, databaseId });

  // Pin the SQL editor's "last viewed project" by visiting the sample
  // project once. Without this pin, gotoHome() in later specs lands on
  // whatever `SQLEditorRouteShell.fallbackToFirstProject` returns —
  // `head(projects)` from a list sorted by created_time DESC. Any spec
  // that creates an additional project (e.g. connection.spec.ts's
  // project-switcher fixture) would silently shift the default project
  // for every later spec, breaking gotoHome-based worksheet/sidebar
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
