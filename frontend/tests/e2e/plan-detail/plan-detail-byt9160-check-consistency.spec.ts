import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let planId: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;
let shouldSkip = false;

// Counts captured by test 1, consumed by test 2
let spec1SuccessCount = 0;
let spec2SuccessCount = 0;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Permissive settings — no gates needed for this test
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  const ts = Date.now();

  // Create two sheets with distinct SQL
  const sheetA = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_9160a_${ts} TEXT;`
  );
  const sheetB = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_9160b_${ts} INTEGER NOT NULL DEFAULT 0;`
  );

  // Discover a second database on a different instance
  const { instances } = await env.api.listInstances();
  // env.instance is like "instances/test-sample-instance" — extract its name
  const primaryInstanceName = env.instance; // e.g. "instances/prod-sample-instance"

  let secondDatabase: string | null = null;

  for (const inst of instances) {
    if (inst.name === primaryInstanceName) continue;
    if (inst.name.includes("bytebase-meta")) continue;
    if (inst.engine !== "POSTGRES") continue;
    try {
      const { databases } = await env.api.listDatabases(inst.name);
      // Only pick databases that belong to our project
      const candidate = databases.find((db) => db.project === env.project);
      if (candidate) {
        secondDatabase = candidate.name;
        break;
      }
    } catch {
      // Instance unreachable — skip
    }
  }

  if (!secondDatabase) {
    // No second instance/database available — flag to skip all tests
    shouldSkip = true;
    return;
  }

  // Create plan with 2 specs:
  //   Spec A — single target (env.database), sheet A
  //   Spec B — two targets (env.database + secondDatabase, overlapping), sheet B
  const plan = await env.api.createPlan(
    env.project,
    `E2E BYT-9160 Check Consistency ${ts}`,
    [
      { id: `spec-a-${ts}`, targets: [env.database], sheet: sheetA },
      {
        id: `spec-b-${ts}`,
        targets: [env.database, secondDatabase],
        sheet: sheetB,
      },
    ]
  );
  planId = plan.name.split("/").pop()!;

  // Create issue so the plan gets a full lifecycle context
  await env.api.createIssue(env.project, "E2E BYT-9160 Check Consistency", plan.name);

  // Run plan checks
  await env.api.runPlanChecks(plan.name);

  // Poll until plan checks are DONE (max 60 s).
  // getPlanCheckRun returns 404 before the first run completes — catch and retry.
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    try {
      const checkRun = await env.api.getPlanCheckRun(plan.name);
      if (checkRun.status === "DONE") break;
    } catch {
      // Check run not created yet — retry
    }
    await new Promise((r) => setTimeout(r, 2000));
  }

  // Open the shared browser context
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("Plan Detail: Check Count Consistency (BYT-9160)", () => {
  test.describe.configure({ mode: "serial" });

  test("spec #2 has more check results than spec #1", async () => {
    test.skip(shouldSkip, "No second database available for multi-spec plan");
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();

    // Helper: read the success count digit next to "Success" text in the inline Checks h3 area.
    // Uses evaluate to find the count reliably regardless of nesting.
    const readInlineSuccessCount = async (): Promise<number> => {
      return page.evaluate(() => {
        // Find all elements with text "Success" that are near an h3 "Checks"
        const all = Array.from(document.querySelectorAll("*"));
        for (const el of all) {
          if (el.children.length > 0) continue; // leaf nodes only
          if (el.textContent?.trim() !== "Success") continue;
          // Walk siblings and parent siblings to find a digit
          const parent = el.parentElement;
          if (!parent) continue;
          const siblings = Array.from(parent.querySelectorAll("*"));
          for (const s of siblings) {
            if (s === el) continue;
            const t = s.textContent?.trim();
            if (t && /^\d+$/.test(t)) {
              // Check that this is near an h3 "Checks" (within 5 parents)
              let node: Element | null = parent;
              for (let i = 0; i < 5 && node; i++) {
                if (node.querySelector("h3")?.textContent?.includes("Checks")) {
                  return parseInt(t, 10);
                }
                node = node.parentElement;
              }
            }
          }
        }
        return 0;
      });
    };

    // Spec #1 is already selected
    await expect(page.getByText("Success").first()).toBeVisible({ timeout: 15_000 });
    spec1SuccessCount = await readInlineSuccessCount();
    expect(spec1SuccessCount).toBeGreaterThan(0);

    // Click spec #2 tab
    await page.getByText("#2").first().click();
    await page.waitForLoadState("networkidle");
    await expect(page.getByText("Success").first()).toBeVisible({ timeout: 10_000 });
    spec2SuccessCount = await readInlineSuccessCount();

    // Spec #2 targets two databases, so it must produce more check results
    expect(spec2SuccessCount).toBeGreaterThan(0);
    expect(spec2SuccessCount).toBeGreaterThan(spec1SuccessCount);
  });

  // BYT-9160: sidebar always shows the last spec's counts regardless of which
  // spec tab is selected. Mark fixme until the bug is resolved.
  test.fixme("sidebar check counts match selected spec #1 (BYT-9160)", async () => {
    // Select spec #1
    await planPage.specTab(1).click();
    await expect(planPage.inlineCheckCount("Success")).toBeVisible({ timeout: 10_000 });

    const sidebarRaw = await planPage.sidebarCheckCount("Success").textContent();
    const inlineRaw = await planPage.inlineCheckCount("Success").textContent();

    const sidebarCount = parseInt(sidebarRaw ?? "0", 10);
    const inlineCount = parseInt(inlineRaw ?? "0", 10);

    // Sidebar should reflect spec #1 — not spec #2's larger count.
    // This assertion FAILS because sidebar shows spec #2's count (the bug).
    expect(sidebarCount).toBe(inlineCount);
  });
});
