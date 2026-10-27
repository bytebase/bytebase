// Audit logs → the Actor filter (3.24: #21541, BYT-10291).
//
// The filter searches users, service accounts and workload identities on one
// keyword, whatever its case, and reads a typed resource name with any casing
// of its prefix. The lists a page can call reach only its own parents, so a
// project-owned service account is found by name from its project's audit log
// and, from the workspace audit log, only when its full email is typed.
//
// The actors are created per run; a service account makes one audited write
// (CreateSheet) so the filter has rows to narrow.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import enUS from "../../../src/locales/en-US.json";
import { BytebaseApiClient, type IamPolicy } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";

const ACTOR_SCOPE = enUS["audit-log"]["advanced-search"].scope.actor;
const ACTOR_COLUMN = enUS["audit-log"].table.actor;

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let originalProjectPolicy: IamPolicy;

const suffix = Date.now();
let workspaceAccount: { name: string; email: string };
let projectAccount: { name: string; email: string };
let workloadIdentity: { name: string; email: string };

// Signs in as a service account and makes one audited write in the project.
async function auditedWriteAs(account: { email: string; serviceKey: string }): Promise<void> {
  const client = new BytebaseApiClient({ baseURL: env.baseURL });
  await client.login(account.email, account.serviceKey);
  await client.createSheet(env.project, `SELECT ${suffix};`);
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();

  const workspaceSA = await env.api.createServiceAccount(workspace, `e2e-auditor-${suffix}`, "E2E Auditor");
  const projectSA = await env.api.createServiceAccount(env.project, `e2e-proj-bot-${suffix}`, "E2E Project Bot");
  workloadIdentity = await env.api.createWorkloadIdentity(workspace, `e2e-ci-${suffix}`, "E2E CI", {
    providerType: "GITHUB",
    issuerUrl: "https://token.actions.githubusercontent.com",
    allowedAudiences: ["bytebase"],
    subjectPattern: "repo:example/app:ref:refs/heads/main",
  });
  workspaceAccount = workspaceSA;
  projectAccount = projectSA;

  originalProjectPolicy = await env.api.getProjectIamPolicy(env.project);
  await env.api.appendProjectBinding(env.project, "roles/projectDeveloper", [
    `serviceAccount:${workspaceSA.email}`,
    `serviceAccount:${projectSA.email}`,
  ]);
  await auditedWriteAs(workspaceSA);
  await auditedWriteAs(projectSA);

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
  if (originalProjectPolicy) {
    const current = await env.api.getProjectIamPolicy(env.project);
    await env.api
      .setProjectIamPolicy(env.project, { ...originalProjectPolicy, etag: current.etag })
      .catch((e) => console.warn("restore project policy failed:", e));
  }
  if (workspaceAccount) await env.api.deleteServiceAccount(workspaceAccount.email);
  if (projectAccount) await env.api.deleteServiceAccount(projectAccount.email);
  if (workloadIdentity) await env.api.deleteWorkloadIdentity(workloadIdentity.email);
});

const workspaceAuditLog = () => `${env.baseURL}/audit-log`;
const projectAuditLog = () => `${env.baseURL}/projects/${env.project.split("/").pop()}/audit-logs`;

function menuOption(text: string) {
  return page.locator("[data-search-menu-option-index]").filter({ hasText: text });
}

// Opens the page, picks the Actor scope and types the keyword at a typing pace.
async function searchActor(url: string, keyword: string): Promise<void> {
  await page.goto(url);
  await page.getByPlaceholder(enUS.common.filter).click();
  await menuOption(ACTOR_SCOPE.description).click();
  await page.keyboard.type(keyword, { delay: 30 });
}

// Every Actor cell in the table, once the filtered search has settled.
async function actorCells(): Promise<string[]> {
  const headers = await page.getByRole("columnheader").allInnerTexts();
  const column = headers.findIndex((header) => header.trim() === ACTOR_COLUMN);
  expect(column).toBeGreaterThanOrEqual(0);
  return page
    .locator("tbody tr")
    .evaluateAll(
      (rows, index) => rows.map((row) => row.querySelectorAll("td")[index]?.textContent?.trim() ?? ""),
      column,
    );
}

function expectFilteredTo(actor: string) {
  return expect(page.locator('[data-search-scope-id="actor"]')).toContainText(actor);
}

test.describe("A service account is found by any casing of its name and narrows the log (A1)", () => {
  test("a mixed-case fragment offers the account; choosing it leaves only its rows", async () => {
    await searchActor(workspaceAuditLog(), `E2E-AUDITOR-${suffix}`);
    await menuOption(workspaceAccount.email).click();
    await expectFilteredTo(workspaceAccount.name);
    await expect.poll(async () => new Set(await actorCells()), { timeout: 15_000 }).toEqual(
      new Set([workspaceAccount.name]),
    );
  });
});

test.describe("A workload identity is found the same way (A2)", () => {
  test("a mixed-case fragment offers the identity; choosing it filters on its name", async () => {
    await searchActor(workspaceAuditLog(), `E2E-Ci-${suffix}`);
    await menuOption(workloadIdentity.email).click();
    await expectFilteredTo(workloadIdentity.name);
    // The identity never acted, so nothing is left.
    await expect(page.getByText(enUS.common["no-data"], { exact: true })).toBeVisible({ timeout: 15_000 });
  });
});

test.describe("A typed resource name filters whatever the casing of its prefix (A3)", () => {
  test("ServiceAccounts/<Email> committed with Enter filters on the canonical name", async () => {
    const [local, domain] = workspaceAccount.email.split("@");
    await searchActor(workspaceAuditLog(), `ServiceAccounts/${local.toUpperCase()}@${domain}`);
    await expect(menuOption(workspaceAccount.email)).toBeVisible();
    await page.keyboard.press("Enter");
    await expectFilteredTo(workspaceAccount.name);
    await expect.poll(async () => new Set(await actorCells()), { timeout: 15_000 }).toEqual(
      new Set([workspaceAccount.name]),
    );
  });
});

test.describe("A project-owned service account (A4)", () => {
  test("the workspace log cannot list it, but its full email is offered as typed and filters", async () => {
    await searchActor(workspaceAuditLog(), `E2E-PROJ-BOT-${suffix}`);
    // The account lists reach the workspace only; the search settles empty.
    await expect(page.getByText(enUS.common["search-no-result"], { exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(menuOption(projectAccount.email)).toHaveCount(0);

    await searchActor(workspaceAuditLog(), projectAccount.email.toUpperCase());
    await menuOption(projectAccount.email).click();
    await expectFilteredTo(projectAccount.name);
    await expect.poll(async () => new Set(await actorCells()), { timeout: 15_000 }).toEqual(
      new Set([projectAccount.name]),
    );
  });

  test("the project's own audit log finds it by a fragment", async () => {
    await searchActor(projectAuditLog(), `E2E-PROJ-BOT-${suffix}`);
    await menuOption(projectAccount.email).click();
    await expectFilteredTo(projectAccount.name);
    await expect.poll(async () => new Set(await actorCells()), { timeout: 15_000 }).toEqual(
      new Set([projectAccount.name]),
    );
  });
});
