// Project service account and workload identity lists, and the workload
// identity wizard's sheet. The suite creates its own accounts through the API
// and checks what the lists say about them.
import { expect, test } from "@playwright/test";
import type { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
const suffix = Date.now().toString(36);
const SA_ID = `e2e-list-bot-${suffix}`;
const WI_ID = `e2e-list-ci-${suffix}`;
const created: { serviceAccount?: string; workloadIdentity?: string } = {};
const SHOTS = process.env.BYTEBASE_E2E_SHOTS;

test.beforeAll(async () => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  const sa = await env.api.createServiceAccount(env.project, SA_ID, "E2E list bot");
  created.serviceAccount = sa.email;
  const wi = await env.api.createWorkloadIdentity(env.project, WI_ID, "E2E list CI", {
    providerType: "GITHUB",
    issuerUrl: "https://token.actions.githubusercontent.com",
    allowedAudiences: ["bytebase"],
    subjectPattern: "repo:example-org/billing-api:environment:prod",
  });
  created.workloadIdentity = wi.email;
  await env.api.appendProjectBinding(env.project, "roles/projectDeveloper", [
    `serviceAccount:${sa.email}`,
    `workloadIdentity:${wi.email}`,
  ]);
});

test.afterAll(async () => {
  const policy = await env.api.getProjectIamPolicy(env.project);
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter(
      (m) =>
        m !== `serviceAccount:${created.serviceAccount}` &&
        m !== `workloadIdentity:${created.workloadIdentity}`,
    );
  }
  policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
  await env.api.setProjectIamPolicy(env.project, policy);
  if (created.serviceAccount) await env.api.deleteServiceAccount(created.serviceAccount);
  if (created.workloadIdentity) await env.api.deleteWorkloadIdentity(created.workloadIdentity);
});

test("service account row shows its role and creation time", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/service-accounts`);
  const row = page.getByRole("row").filter({ hasText: `${SA_ID}@` });
  await expect(row).toContainText("E2E list bot");
  await expect(row).toContainText("Project Developer");
  await expect(row.getByRole("cell").nth(2)).not.toHaveText("-");
  await expect(page.getByRole("columnheader")).toHaveText(["Name", "Roles", "Created", "Actions"]);
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/service-accounts.png`, fullPage: true });

  await row.getByRole("button", { name: "More actions for E2E list bot" }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Reset Service Key", "Edit", "Deactivate"]);
  await page.getByRole("menuitem", { name: "Deactivate" }).click();
  await expect(page.getByRole("alertdialog")).toContainText("Deactivate E2E list bot?");
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/service-account-deactivate.png` });
  await page.getByRole("alertdialog").getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByRole("alertdialog")).toBeHidden();
});

test("workload identity row says what it trusts in the form's words", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/workload-identities`);
  const row = page.getByRole("row").filter({ hasText: `${WI_ID}@` });
  await expect(row).toContainText("example-org/billing-api");
  await expect(row).toContainText("Only jobs with environment: prod");
  await expect(row).toContainText("Project Developer");
  await expect(row.getByRole("img", { name: "GitHub Actions" })).toBeVisible();
  await expect(page.getByRole("columnheader")).toHaveText([
    "Name",
    "Trusts",
    "Roles",
    "Created",
    "Actions",
  ]);
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/workload-identities.png`, fullPage: true });

  await page.getByRole("button", { name: "Create" }).first().click();
  const sheet = page.getByRole("dialog", { name: "Create workload identity" });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByRole("navigation", { name: "Steps" })).toBeVisible();
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/workload-identity-sheet.png` });
  await sheet.getByRole("button", { name: "Continue" }).click();
  await expect(sheet).toContainText("Which repositories can sign in?");
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/workload-identity-sheet-step2.png` });
});
