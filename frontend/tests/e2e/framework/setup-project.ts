import { test as setup, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";
import { loadTestEnv, saveTestEnv, type TestEnv } from "./env";
import { checkCrashRecovery } from "./mode-use-local-bytebase";

const AUTH_FILE = path.join(__dirname, "../../.auth/state.json");

setup("authenticate and discover", async ({ page }) => {
  // 1. Read partial TestEnv from globalSetup
  const env = loadTestEnv();
  const api = new BytebaseApiClient({
    baseURL: env.baseURL,
    credentials: env.adminPassword
      ? { email: env.adminEmail, password: env.adminPassword }
      : undefined,
  });

  // 2. Auth
  if (env.mode === "new") {
    // Mode B: credentials from globalSetup (signup already done)
    await api.login(env.adminEmail, env.adminPassword!);
  } else if (env.adminEmail && env.adminPassword) {
    // Mode A: fully automated API login
    await api.login(env.adminEmail, env.adminPassword);
  } else {
    // Mode A: headed browser login
    await page.goto(`${env.baseURL}/auth/signin`);
    if (env.adminEmail) {
      await page.getByRole("textbox", { name: /email/i }).fill(env.adminEmail);
    }
    // User fills remaining fields manually
    await page.waitForURL("**/landing**", { timeout: 120000 });
  }

  // Browser auth: navigate and save state
  if (env.adminEmail && env.adminPassword) {
    await page.goto(`${env.baseURL}/auth/signin`);
    await page.getByRole("textbox", { name: /email/i }).fill(env.adminEmail);
    await page.getByRole("textbox", { name: /password/i }).fill(env.adminPassword);
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page).not.toHaveURL(/\/auth/);
  }

  // Dismiss modals
  await page.keyboard.press("Escape").catch(() => {});
  await page.waitForTimeout(500);

  // 3. Suppress "New version" modal
  await page.evaluate(() => {
    localStorage.setItem(
      "bb.release",
      JSON.stringify({
        ignoreRemindModalTillNextRelease: true,
        nextCheckTs: Date.now() + 86400000,
      })
    );
  });

  // 4. Discovery: find first Postgres instance, database, and project
  let project = "";
  let instance = "";
  let instanceId = "";
  let database = "";
  let databaseId = "";

  try {
    const { instances } = await api.listInstances();
    const pgInstance = instances?.find(
      (i: { engine: string; name: string }) =>
        i.engine === "POSTGRES" && !i.name.includes("deleted")
    );
    if (pgInstance) {
      instance = pgInstance.name;
      instanceId = instance.split("/").pop()!;

      const { databases } = await api.listDatabases(instance);
      const db = databases?.find(
        (d: { name: string }) =>
          !d.name.includes("/postgres") &&
          !d.name.includes("/template") &&
          !d.name.includes("/bbdev")
      ) ?? databases?.[0];
      if (db) {
        database = db.name;
        databaseId = database.split("/").pop()!;
        // Find owning project
        project = (db as { project?: string }).project ?? "";
      }
    }
  } catch (err) {
    console.warn("Discovery failed (tests may still work with defaults):", err);
  }

  // 5. Mode A crash recovery check
  if (env.mode === "local") {
    await checkCrashRecovery(api);
  }

  // 6. Update env with discovered data
  saveTestEnv({
    ...env,
    project,
    instance,
    instanceId,
    database,
    databaseId,
  });

  // 7. Save browser auth state
  const authDir = path.dirname(AUTH_FILE);
  if (!fs.existsSync(authDir)) fs.mkdirSync(authDir, { recursive: true });
  await page.context().storageState({ path: AUTH_FILE });
});
