// The release-backed plan page (docs/design/release-plan-page.md): the Changes
// list, each task's file spine before and after its run, a failed file in the
// spine and the viewer, Skip with a required reason, and the old release URL.
// Releases, plans and runs are set up through the API, as bytebase-action
// creates them; a project Developer reads the page and the admin, an Owner,
// skips.

import { type Browser, expect, type Page, test } from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { execSql, getInstancePgPort } from "../framework/psql";
import { signInBrowserAs } from "../framework/sign-in";
import { setPermissiveGates } from "./plan-helpers";

test.setTimeout(240_000);

const SCREENSHOT_DIR = process.env.BYTEBASE_E2E_SCREENSHOT_DIR;
const PASSWORD = "12345678";
const DEVELOPER = "release-plan-dev@example.com";
const DEVELOPER_MEMBER = `user:${DEVELOPER}`;
const DEVELOPER_AUTH = `${process.cwd()}/.auth/release-plan-dev.json`;
const COMMIT = "0a1b2c3d4e5f60718293a4b5c6d7e8f901234567";

type Rollout = {
  stages?: { name: string; tasks?: { name: string; status?: string }[] }[];
};

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let databaseName: string;
let schema: string;
let pgPort: string | undefined;
// Versions are counters unique to this run, so a retry on the same server
// applies its own and never finds them already applied.
let version: (n: number) => string;
let originalProjectSettings: Record<string, boolean> = {};
let originalApprovalSetting: unknown = null;
const createdPlans: string[] = [];
const createdReleases: string[] = [];
let successPlan: { id: string; name: string };
let failedPlan: { id: string; name: string; releaseId: string };

const lastSegment = (name: string) => name.split("/").pop()!;

async function screenshot(page: Page, name: string) {
  if (SCREENSHOT_DIR) {
    await page.screenshot({ path: `${SCREENSHOT_DIR}/${name}.png`, fullPage: true });
  }
}

async function createReleasePlan(
  label: string,
  files: { n: number; statement: string }[],
): Promise<{ id: string; name: string; releaseId: string }> {
  const release = await env.api.createRelease(env.project, {
    files: files.map(({ n, statement }) => ({
      path: `migrations/${version(n)}_${label}.sql`,
      version: version(n),
      statement,
    })),
    vcsSource: {
      vcsType: "GITHUB",
      url: `https://github.com/example/app/commit/${COMMIT}`,
    },
  });
  createdReleases.push(release.name);
  const plan = await env.api.createReleasePlan(env.project, lastSegment(release.name), {
    id: `release-plan-${label}-${Date.now()}`,
    targets: [env.database],
    release: release.name,
  });
  createdPlans.push(plan.name);
  await env.api.createRollout(plan.name);
  return { id: lastSegment(plan.name), name: plan.name, releaseId: lastSegment(release.name) };
}

async function runAndWait(planName: string, status: "DONE" | "FAILED") {
  const rollout = (await env.api.getRollout(planName)) as Rollout;
  for (const stage of rollout.stages ?? []) {
    await env.api.batchRunTasks(stage.name, (stage.tasks ?? []).map((task) => task.name));
  }
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    const current = (await env.api.getRollout(planName)) as Rollout;
    const statuses = (current.stages ?? []).flatMap((stage) =>
      (stage.tasks ?? []).map((task) => task.status),
    );
    if (statuses.length > 0 && statuses.every((s) => s === status)) return;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`${planName} did not reach ${status} within 120s`);
}

async function developerPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext({ storageState: DEVELOPER_AUTH });
  return context.newPage();
}

async function editPolicy(edit: (policy: { bindings: { role: string; members: string[] }[] }) => void) {
  const policy = await env.api.getProjectIamPolicy(env.project);
  edit(policy as unknown as { bindings: { role: string; members: string[] }[] });
  await env.api.setProjectIamPolicy(env.project, policy);
}

const dropDeveloper = (policy: { bindings: { role: string; members: string[] }[] }) => {
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter((member) => member !== DEVELOPER_MEMBER);
  }
  policy.bindings = policy.bindings.filter((binding) => binding.members.length > 0);
};

// The task card's identity button toggles it; it names the database.
const taskCard = (page: Page) =>
  page.locator("#plan-phase-deploy").getByRole("button", { name: new RegExp(databaseName) }).first();

test.beforeAll(async ({ browser }) => {
  test.setTimeout(240_000);
  env = loadTestEnv();
  projectId = lastSegment(env.project);
  databaseName = lastSegment(env.database);
  await env.api.login(env.adminEmail, env.adminPassword);
  const ts = `${Date.now()}`;
  schema = `release_plan_${ts}`;
  version = (n: number) => `${ts}${String(n).padStart(2, "0")}`;

  const project = await env.api.getProject(env.project);
  originalProjectSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
    enforceSqlReview: !!project.enforceSqlReview,
    forceIssueLabels: !!project.forceIssueLabels,
  };
  originalApprovalSetting = (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value ?? null;
  await setPermissiveGates(env.api, env.project);

  await env.api.createUser(DEVELOPER, PASSWORD, "Release plan developer").catch((err: unknown) => {
    const msg = err instanceof Error ? err.message : String(err);
    if (!msg.includes("(409)") && !msg.includes("already exists")) throw err;
  });
  await editPolicy((policy) => {
    dropDeveloper(policy);
    policy.bindings.push({ role: "roles/projectDeveloper", members: [DEVELOPER_MEMBER] });
  });
  await signInBrowserAs(browser, env.baseURL, DEVELOPER, PASSWORD, DEVELOPER_AUTH);

  // The target already carries version 01.
  const baseline = await createReleasePlan("baseline", [
    { n: 1, statement: `CREATE SCHEMA ${schema}; CREATE TABLE ${schema}.t1 (id INT);` },
  ]);
  await runAndWait(baseline.name, "DONE");

  // Three files, the first already applied: the planned spine, then a done run.
  successPlan = await createReleasePlan("orders", [
    { n: 1, statement: `CREATE SCHEMA ${schema}; CREATE TABLE ${schema}.t1 (id INT);` },
    { n: 2, statement: `CREATE TABLE ${schema}.orders (id INT);` },
    { n: 3, statement: `CREATE INDEX orders_id ON ${schema}.orders (id);` },
  ]);

  // Three new files whose second fails: the table it alters does not exist.
  failedPlan = await createReleasePlan("broken", [
    { n: 1, statement: `CREATE SCHEMA ${schema}; CREATE TABLE ${schema}.t1 (id INT);` },
    { n: 4, statement: `CREATE TABLE ${schema}.items (id INT);` },
    { n: 5, statement: `ALTER TABLE ${schema}.missing ADD COLUMN c INT;` },
    { n: 6, statement: `CREATE TABLE ${schema}.later (id INT);` },
  ]);
  // Failed here, not in a test, so each test that reads the failure stands
  // on its own.
  await runAndWait(failedPlan.name, "FAILED");
  pgPort = await getInstancePgPort(env);
});

test.afterAll(async () => {
  if (!env) return;
  await env.api.updateProjectSettings(env.project, originalProjectSettings);
  if (originalApprovalSetting) {
    await env.api.upsertSetting("WORKSPACE_APPROVAL", originalApprovalSetting, "value.workspace_approval");
  }
  await editPolicy(dropDeveloper);
  if (pgPort && /^release_plan_\d+$/.test(schema)) {
    execSql(databaseName, pgPort, `DROP SCHEMA IF EXISTS ${schema} CASCADE`);
  }
  for (const plan of createdPlans) await env.api.closePlan(plan).catch(() => {});
  for (const release of createdReleases) await env.api.deleteRelease(release).catch(() => {});
});

test("before the run, Changes lists the release newest first and the task shows what it will apply", async ({ browser }) => {
  const page = await developerPage(browser);
  await page.goto(`${env.baseURL}/projects/${projectId}/plans/${successPlan.id}`);

  const changes = page.getByTestId("release-changes");
  if (!(await changes.isVisible().catch(() => false))) {
    await page.locator("#plan-phase-changes").getByText("Show details", { exact: true }).click();
  }
  await expect(changes).toBeVisible();
  await expect(page.getByTestId("release-change-row")).toHaveCount(3);
  await expect(page.getByTestId("release-change-row").first()).toHaveAttribute("data-version", version(3));
  await screenshot(page, "release-changes");

  await changes.getByRole("button", { name: `migrations/${version(2)}_orders.sql` }).click();
  const viewer = page.getByTestId("release-file-viewer");
  await expect(viewer.getByRole("heading", { name: version(2), exact: true })).toBeVisible();
  await expect(viewer.getByTestId("release-file-path")).toHaveText(`migrations/${version(2)}_orders.sql`);
  await expect(viewer.locator(".view-lines")).toContainText("CREATE TABLE");
  await expect(viewer.getByRole("link", { name: /Open in GitHub at 0a1b2c3/ })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(viewer).toHaveCount(0);

  // The folded card: what this database will get, before any log exists.
  await expect(page.getByTestId("folded-description").first()).toHaveText(
    /2 to apply · 1 applied earlier/,
  );
  await taskCard(page).click();
  const spine = page.getByTestId("release-spine");
  await expect(spine).toBeVisible();
  await expect(spine.getByTestId("spine-count")).toHaveText("3 files · 2 pending · 1 applied earlier");
  await expect(spine.getByRole("button", { name: "Load 1 applied earlier" })).toBeVisible();
  await expect(spine.getByTestId(`spine-file-${version(2)}`)).toContainText("Will apply");
  await expect(spine.getByTestId(`spine-file-${version(3)}`)).toContainText("Will apply");
  await expect(page.getByText(/As of rollout creation/)).toBeVisible();
  await screenshot(page, "release-planned-spine");
  await page.context().close();
});

test("after the run, the task lists the version it passed over as applied earlier and links to the database's revisions", async ({ browser }) => {
  await runAndWait(successPlan.name, "DONE");
  const page = await developerPage(browser);
  await page.goto(`${env.baseURL}/projects/${projectId}/plans/${successPlan.id}`);

  await expect(page.getByTestId("folded-description").first()).toHaveText(/All 3 applied/);
  await taskCard(page).click();
  const spine = page.getByTestId("release-spine");
  await expect(spine.getByTestId("spine-count")).toHaveText("3 files · 2 applied · 1 applied earlier");
  await spine.getByRole("button", { name: "Load 1 applied earlier" }).click();
  await expect(spine.getByTestId("spine-applied-earlier-row")).toContainText(`migrations/${version(1)}_orders.sql`);
  await expect(spine.getByTestId(`spine-file-${version(2)}`)).toContainText("Applied");
  await expect(spine.getByTestId(`spine-file-${version(3)}`)).toContainText("Applied");
  await screenshot(page, "release-done-spine");
  // The sample instance is a project instance, so the link must carry it.
  await page.getByRole("link", { name: `Revisions on ${databaseName}` }).click();
  await expect(page).toHaveURL(new RegExp(`/databases/${databaseName}\\?parent=.*#revision$`));
  await expect(page.getByRole("tab", { name: "Revision" })).toHaveAttribute("aria-selected", "true");
  await page.context().close();
});

test("a failed file shows its error on the row, in the viewer and on the folded card, and nothing after it", async ({ browser }) => {
  const page = await developerPage(browser);
  await page.goto(`${env.baseURL}/projects/${projectId}/plans/${failedPlan.id}`);

  const folded = page.getByTestId("folded-description").first();
  await expect(folded).toContainText(version(5));
  await expect(folded).toContainText("does not exist");
  await expect(folded).not.toContainText("failed to execute release file");
  await screenshot(page, "release-failed-folded");

  await taskCard(page).click();
  const spine = page.getByTestId("release-spine");
  await expect(spine.getByTestId(`spine-file-${version(4)}`)).toContainText("Applied");
  const failed = spine.getByTestId(`spine-file-${version(5)}`);
  await expect(failed).toHaveAttribute("aria-expanded", "true");
  await expect(failed).toContainText("Failed");
  await expect(spine.getByTestId(`spine-file-${version(5)}-content`)).toContainText("does not exist");
  await expect(spine.getByTestId(`spine-file-${version(6)}`)).toHaveCount(0);
  await screenshot(page, "release-failed-spine");

  await spine.getByRole("button", { name: `migrations/${version(5)}_broken.sql` }).click();
  const viewer = page.getByTestId("release-file-viewer");
  await expect(viewer).toContainText(/Failed on/);
  await expect(viewer.locator(".squiggly-error").first()).toBeAttached();
  await screenshot(page, "release-failed-viewer");
  await page.context().close();
});

test("an Owner's skip needs a reason, then the task says who skipped it and the plan reads Deployed, 1 task skipped", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/plans/${failedPlan.id}`);
  await taskCard(page).click();
  // The task's own Skip; the selection toolbar's stays disabled with nothing selected.
  await page
    .locator("#plan-phase-deploy")
    .getByRole("button", { name: "Skip", exact: true, disabled: false })
    .first()
    .click();

  const dialog = page.getByRole("dialog");
  const confirm = dialog.getByRole("button", { name: "Skip", exact: true });
  await expect(dialog.getByTestId("task-error-line")).toContainText(version(5));
  await expect(confirm).toBeDisabled();
  await dialog.getByRole("textbox").fill("Fixed in git; the next release carries it");
  await expect(confirm).toBeEnabled();
  await screenshot(page, "release-skip-dialog");
  await confirm.click();

  await expect(page.getByText(/^Skipped by .+ · /).first()).toBeVisible();
  await expect(page.getByText("Fixed in git; the next release carries it").first()).toBeVisible();
  await expect(page.getByText("Deployed, 1 task skipped")).toBeVisible();
  await screenshot(page, "release-skipped");
});

test("the release URL opens its plan on Deploy with Changes folded", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/releases/${failedPlan.releaseId}`);
  await expect(page).toHaveURL(new RegExp(`/plans/${failedPlan.id}/rollout`));
  await expect(page.getByTestId("release-spine").or(page.getByTestId("folded-description")).first()).toBeVisible();
  await expect(page.getByTestId("release-changes")).toHaveCount(0);
});
