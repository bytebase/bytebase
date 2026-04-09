import { test as setup, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";
import { loadTestEnv, saveTestEnv } from "./env";

const AUTH_FILE = path.join(__dirname, "../../../.auth/state.json");

setup("authenticate and discover", async ({ page }) => {
  const env = loadTestEnv();
  const api = new BytebaseApiClient({
    baseURL: env.baseURL,
    credentials: { email: env.adminEmail, password: env.adminPassword },
  });

  // API login
  await api.login(env.adminEmail, env.adminPassword);

  // Browser login for auth cookies
  await page.goto(`${env.baseURL}/auth/signin`);
  if (page.url().includes("/auth")) {
    const emailField = page.getByRole("textbox", { name: /email/i });
    const hasDemoLogin = await emailField.count() === 0;

    if (hasDemoLogin) {
      await page.getByRole("button", { name: "Sign in", exact: true }).click();
    } else {
      await emailField.fill(env.adminEmail);
      await page.getByRole("textbox", { name: /password/i }).fill(env.adminPassword);
      await page.getByRole("button", { name: "Sign in", exact: true }).click();
    }
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
  let project = "";
  let instance = "";
  let instanceId = "";
  let database = "";
  let databaseId = "";

  const { instances } = await api.listInstances();
  const pgInstance = instances?.find(
    (i: { engine: string; name: string }) =>
      i.engine === "POSTGRES" &&
      !i.name.includes("deleted") &&
      !i.name.includes("bytebase-meta")
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
      project = (db as { project?: string }).project ?? "";
    }
  }

  if (!database) {
    throw new Error("Discovery failed: no suitable Postgres database found");
  }

  saveTestEnv({ ...env, project, instance, instanceId, database, databaseId });

  // Save browser auth state
  const authDir = path.dirname(AUTH_FILE);
  if (!fs.existsSync(authDir)) fs.mkdirSync(authDir, { recursive: true });
  await page.context().storageState({ path: AUTH_FILE });
});
