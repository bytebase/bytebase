import {
  expect,
  test,
  type BrowserContext,
  type Locator,
  type Page,
} from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";

test.setTimeout(240_000);

const ISSUE_COUNT = 18;
const PLAN_COUNT = 55;
let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let issueListUrl: string;
let planListUrl: string;
let originalExternalUrl = "";

type SavedViewport = {
  scrollTop: number;
  anchorOffset: number;
};

const readViewport = (row: Locator): Promise<SavedViewport> =>
  row.evaluate((element) => {
    const main = document.querySelector<HTMLElement>("#bb-layout-main");
    if (!main) throw new Error("Missing #bb-layout-main");
    return {
      scrollTop: main.scrollTop,
      anchorOffset:
        element.getBoundingClientRect().top - main.getBoundingClientRect().top,
    };
  });

async function loadAllRows(rows: Locator, expectedCount: number): Promise<void> {
  while ((await rows.count()) < expectedCount) {
    const previousCount = await rows.count();
    await page.getByRole("button", { name: /Load more/i }).click();
    await expect
      .poll(() => rows.count())
      .toBeGreaterThan(previousCount);
  }
  await expect(rows).toHaveCount(expectedCount);
}

async function expectBackRestored({
  detailUrlPattern,
  listUrl,
  rowTestId,
  rowCount,
}: {
  detailUrlPattern: RegExp;
  listUrl: string;
  rowTestId: "issue-list-item" | "plan-list-item";
  rowCount: number;
}): Promise<void> {
  await page.goto(listUrl);
  const rows = page.getByTestId(rowTestId);
  await expect(rows.first()).toBeVisible({ timeout: 20_000 });
  await loadAllRows(rows, rowCount);

  const target = rows.last();
  await target.scrollIntoViewIfNeeded();
  const before = await readViewport(target);
  expect(before.scrollTop).toBeGreaterThan(100);

  await target.click();
  await expect(page).toHaveURL(detailUrlPattern, { timeout: 20_000 });
  await page.goBack();
  await expect(page).toHaveURL(listUrl);
  await expect(rows).toHaveCount(rowCount, { timeout: 20_000 });
  await expect(target).toBeVisible({ timeout: 20_000 });

  await expect
    .poll(async () => {
      const after = await readViewport(target);
      return Math.abs(after.scrollTop - before.scrollTop);
    })
    .toBeLessThanOrEqual(2);
  await expect
    .poll(async () => {
      const after = await readViewport(target);
      return Math.abs(after.anchorOffset - before.anchorOffset);
    })
    .toBeLessThanOrEqual(2);
  expect(await page.evaluate(() => window.scrollY)).toBe(0);
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const projectId = env.project.split("/").pop()!;
  const stamp = Date.now();
  const searchToken = `scroll-restoration-${stamp}`;
  issueListUrl = `${env.baseURL}/projects/${projectId}/issues?q=${encodeURIComponent(
    searchToken
  )}`;
  planListUrl = `${env.baseURL}/projects/${projectId}/plans?q=${encodeURIComponent(
    searchToken
  )}`;

  const setting = await env.api.getSetting("WORKSPACE_PROFILE");
  originalExternalUrl =
    (setting?.value as { workspaceProfile?: { externalUrl?: string } })
      ?.workspaceProfile?.externalUrl ?? "";

  const sheet = await env.api.createSheet(env.project, "SELECT 1;");
  for (let i = 0; i < PLAN_COUNT; i++) {
    const title = `${searchToken} issue ${i}`;
    const plan = await env.api.createPlan(env.project, title, [
      {
        id: `scroll-restoration-${stamp}-${i}`,
        targets: [env.database],
        sheet,
      },
    ]);
    const repeatedDescription = Array.from(
      { length: (i % 4) + 1 },
      () => `${searchToken} variable-height content`
    ).join("\n");
    if (i < ISSUE_COUNT) {
      await env.api.createIssue(
        env.project,
        title,
        plan.name,
        repeatedDescription
      );
    }
  }

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
    // Below Tailwind's sm breakpoint, the banner CTA moves onto its own row.
    viewport: { width: 600, height: 720 },
  });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await env.api.setWorkspaceExternalUrl(originalExternalUrl).catch(() => {});
  await sharedContext?.close();
});

test.describe("dashboard scroll restoration", () => {
  test.describe.configure({ mode: "serial" });

  test("restores the nested main pane without a banner", async () => {
    await env.api.setWorkspaceExternalUrl(env.baseURL);
    await page.goto(issueListUrl);
    await expect(
      page.getByRole("link", { name: /Configure now/i })
    ).toHaveCount(0);
    await expectBackRestored({
      detailUrlPattern: /\/projects\/[^/]+\/plans\//,
      listUrl: issueListUrl,
      rowTestId: "issue-list-item",
      rowCount: ISSUE_COUNT,
    });
  });

  test("restores every loaded page in the paginated Plans view", async () => {
    await env.api.setWorkspaceExternalUrl(env.baseURL);
    await expectBackRestored({
      detailUrlPattern: /\/projects\/[^/]+\/plans\//,
      listUrl: planListUrl,
      rowTestId: "plan-list-item",
      rowCount: PLAN_COUNT,
    });
  });

  test("restores the nested main pane below a wrapped banner", async () => {
    await env.api.setWorkspaceExternalUrl("");
    await page.goto(issueListUrl);
    await expect(
      page.getByRole("link", { name: /Configure now/i }).first()
    ).toBeVisible({ timeout: 10_000 });
    await expectBackRestored({
      detailUrlPattern: /\/projects\/[^/]+\/plans\//,
      listUrl: issueListUrl,
      rowTestId: "issue-list-item",
      rowCount: ISSUE_COUNT,
    });
  });
});
