// Workspace → Instances → Add instance: the connection form (3.23: #21396,
// #21419, #21449, #21454).
//
// The form validates before it submits (the hostname starts empty and is
// required; Test Connection and Create stay disabled until the form is valid),
// offers an explicit Test Connection with inline recovery guidance on failure,
// asks before creating an instance whose connection failed, and edits secrets
// through SecretInput: "Unchanged" until touched, an explicit clear, and the
// form's Cancel to discard.
//
// The connection target is the suite's own sample Postgres, reached over its
// Unix socket directory (the sample instance registers `/tmp` as its host), so
// the success path needs no network and no extra database.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import enUS from "../../../src/locales/en-US.json";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";

const INSTANCE = enUS.instance;
const RECOVERY = enUS.instance["connection-recovery"];
const COMMON = enUS.common;

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
// The sample Postgres' admin data source: host (socket dir), port, user.
let sample: { host: string; port: string; username: string };
const createdInstances: string[] = [];

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const instance = await env.api.getInstance(env.instance);
  const admin = instance.dataSources.find((ds) => ds.type === "ADMIN") ?? instance.dataSources[0];
  if (!admin?.host || !admin.port || !admin.username) {
    throw new Error(
      `sample instance ${env.instance} has no usable admin data source: ${JSON.stringify(instance.dataSources)}`,
    );
  }
  sample = { host: admin.host, port: admin.port, username: admin.username };
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  for (const name of createdInstances) {
    await env.api.deleteInstance(name).catch((e) => console.warn("cleanup failed:", name, e));
  }
  await sharedContext?.close();
});

const hostname = () => page.getByRole("textbox", { name: "Hostname *" });
const port = () => page.getByRole("textbox", { name: "Port", exact: true });
const username = () => page.getByRole("textbox", { name: "Username", exact: true });
const testConnection = () => page.getByRole("button", { name: INSTANCE["test-connection"], exact: true });
const create = () => page.getByRole("button", { name: "Create", exact: true });

async function gotoNewPostgresInstance(): Promise<void> {
  await page.goto(`${env.baseURL}/instances/new`);
  await page.getByRole("button", { name: "PostgreSQL", exact: true }).click();
  await expect(hostname()).toBeVisible();
  await expect(hostname()).toHaveValue("");
}

// The connection-test result arrives as a toast that stays until dismissed and
// sits over the footer's actions; the user closes it before the next click, so
// the journeys do too. Every toast's × is labelled "Close"
// (components/ui/toast.tsx); the visible card is aria-hidden behind its live
// region, hence includeHidden.
async function closeToasts(): Promise<void> {
  const closes = page.getByRole("button", { name: "Close", exact: true, includeHidden: true });
  for (let attempt = 0; attempt < 5 && (await closes.count()) > 0; attempt++) {
    await closes.first().click({ timeout: 2_000 }).catch(() => {});
    await expect(closes).toHaveCount(0, { timeout: 2_000 }).catch(() => {});
  }
}

test.describe("An empty hostname is refused before anything is sent (I1)", () => {
  test("the required error shows on blur and both actions stay disabled", async () => {
    await gotoNewPostgresInstance();
    await expect(testConnection()).toBeDisabled();
    await expect(create()).toBeDisabled();

    await hostname().click();
    await page.keyboard.press("Tab");
    await expect(hostname()).toHaveAttribute("aria-invalid", "true");
    await expect(page.getByRole("alert").filter({ hasText: INSTANCE.validation.required })).toBeVisible();
    await expect(testConnection()).toBeDisabled();
    await expect(create()).toBeDisabled();
  });
});

test.describe("Test Connection explains an unreachable host; dismissing the error gives the retry back (I2)", () => {
  test("the network recovery card renders with the engine's error, and the retry works once the error is closed", async () => {
    await gotoNewPostgresInstance();
    await hostname().fill("127.0.0.1");
    await port().fill("1");
    await expect(testConnection()).toBeEnabled();
    await testConnection().click();

    await expect(page.getByRole("heading", { name: RECOVERY.network.title })).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText(/connection refused/).first()).toBeVisible();
    // Still on the form: nothing was created.
    await expect(page).toHaveURL(/\/instances\/new$/);

    // The retry runs a second connection check once the error is closed.
    await closeToasts();
    const retried = page.waitForResponse((response) =>
      response.url().endsWith("/bytebase.v1.InstanceService/CreateInstance"),
    );
    await testConnection().click();
    await retried;
    await expect(page.getByRole("heading", { name: RECOVERY.network.title })).toBeVisible();
    await closeToasts();
  });
});

test.describe("Create asks before saving a connection that failed (I4)", () => {
  test("Cancel keeps the form; nothing is created", async () => {
    await gotoNewPostgresInstance();
    await hostname().fill("127.0.0.1");
    await port().fill("1");
    await create().click();

    const dialog = page.getByRole("alertdialog");
    await expect(dialog.getByRole("heading", { name: INSTANCE["failed-to-connect-instance"] })).toBeVisible({ timeout: 20_000 });
    await expect(dialog.getByRole("heading", { name: RECOVERY.network.title })).toBeVisible();
    await expect(dialog.getByRole("button", { name: COMMON["continue-anyway"] })).toBeVisible();
    await dialog.getByRole("button", { name: COMMON.cancel, exact: true }).click();
    await expect(dialog).toBeHidden();
    await expect(page).toHaveURL(/\/instances\/new$/);
    await expect(hostname()).toHaveValue("127.0.0.1");
    await closeToasts();
  });
});

test.describe("Test, create, then edit the secret (I3)", () => {
  const title = `E2E Instance Form ${Date.now()}`;
  let instanceId = "";

  test("a tested connection creates the instance, whose stored password then edits through SecretInput", async () => {
    test.setTimeout(180_000);
    await test.step("a reachable sample Postgres tests successfully and Create lands on the instance", async () => {
      await gotoNewPostgresInstance();
      await page.getByRole("textbox", { name: "Instance Name *" }).fill(title);
      await hostname().fill(sample.host);
      await port().fill(sample.port);
      await username().fill(sample.username);
      await testConnection().click();
      await expect(page.getByText(INSTANCE["successfully-connected-instance"])).toBeVisible({ timeout: 20_000 });

      await closeToasts();
      await create().click();
      await page.waitForURL(/\/instances\/(?!new$)[^/?#]+/, { timeout: 60_000 });
      instanceId = page.url().match(/\/instances\/([^/?#]+)/)?.[1] ?? "";
      expect(instanceId).not.toBe("");
      createdInstances.push(`instances/${instanceId}`);
      // Exact and level 1: the creation toast's heading also contains the title.
      await expect(page.getByRole("heading", { level: 1, name: title, exact: true })).toBeVisible({
        timeout: 20_000,
      });
    });

    await test.step("the stored password reads Unchanged; typing, clearing and Cancel round-trip", async () => {
      // Re-enter the instance without the post-create ?intro= query: that product
      // intro dims the page once the database list loads and swallows clicks.
      expect(instanceId).not.toBe("");
      await page.goto(`${env.baseURL}/instances/${instanceId}`);
      await page.getByRole("tab", { name: COMMON.overview, exact: true }).click();

      const password = page.getByRole("textbox", { name: "Password", exact: true });
      const clear = page.getByRole("button", { name: COMMON.clear, exact: true });
      const reveal = page.getByRole("button", { name: COMMON["toggle-password-visibility"] });
      const update = page.getByRole("button", { name: COMMON.update, exact: true });

      // Untouched: a placeholder, an explicit clear, no pending change.
      await expect(password).toHaveAttribute("placeholder", COMMON["secret-input"].unchanged);
      await expect(password).toHaveValue("");
      await expect(clear).toBeVisible();
      await expect(reveal).toHaveCount(0);
      await expect(update).toHaveCount(0);

      // Typing replaces: the eye reveals only the new input, the form is dirty.
      await password.fill("e2e-new-secret");
      await expect(password).toHaveValue("e2e-new-secret");
      await expect(reveal).toBeVisible();
      await expect(update).toBeEnabled();

      // × clears explicitly — an empty value, no "Unchanged" placeholder, still dirty.
      await clear.click();
      await expect(password).toHaveValue("");
      await expect(password).not.toHaveAttribute("placeholder", COMMON["secret-input"].unchanged);
      await expect(update).toBeEnabled();

      // The form's Cancel discards: back to Unchanged, nothing pending.
      await page.getByRole("button", { name: COMMON.cancel, exact: true }).click();
      await expect(password).toHaveAttribute("placeholder", COMMON["secret-input"].unchanged);
      await expect(update).toHaveCount(0);
    });
  });
});
