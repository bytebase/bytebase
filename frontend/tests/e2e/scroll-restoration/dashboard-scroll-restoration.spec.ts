import {
  expect,
  test,
  type BrowserContext,
  type Locator,
  type Page,
} from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import {
  createDatabaseChangePlanViaUI,
  submitDraftForReviewViaUI,
} from "../framework/ui-create-plan";

test.setTimeout(240_000);

const ISSUE_COUNT = 18;
let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let issueListUrl: string;
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

// Build the initial list up to `expectedCount` rows by clicking "Load more",
// the way a user loads more of a paginated list. Each page is fetched
// asynchronously and can lag under full-suite server load, so this is patient:
// it waits for the row count to grow after each click and keeps clicking
// "Load more" whenever the affordance is present, bounded by an overall
// deadline, rather than assuming a click immediately yields the next page.
async function loadAllRows(rows: Locator, expectedCount: number): Promise<void> {
  const loadMore = page.getByRole("button", { name: /Load more/i }).first();
  const deadline = Date.now() + 90_000;
  while ((await rows.count()) < expectedCount && Date.now() < deadline) {
    const previousCount = await rows.count();
    if (await loadMore.isVisible().catch(() => false)) {
      await loadMore.click().catch(() => {});
      await expect
        .poll(() => rows.count(), { timeout: 15_000 })
        .toBeGreaterThan(previousCount)
        .catch(() => {});
    } else {
      // "Load more" not shown yet — a restore may still be re-fetching a page.
      await page.waitForTimeout(500);
    }
  }
  await expect(rows).toHaveCount(expectedCount, { timeout: 20_000 });
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

  // Mimic a real user dwelling on the loaded list before drilling into a row.
  // The app persists its paged-data snapshot in a passive effect that runs after
  // the rows render; a human's pause lets it finish, but Playwright otherwise
  // navigates away instantly and the snapshot is written stale (or not at all).
  // With the snapshot complete, the back-navigation below restores every row
  // from cache with NO refetch — which also sidesteps the POP-mount fetch/abort
  // race that otherwise leaves the list short or empty. Waiting for network idle
  // is the settle signal; the small margin covers the post-render effect.
  await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
  await page.waitForTimeout(1_000);

  const target = rows.last();
  await target.scrollIntoViewIfNeeded();
  const before = await readViewport(target);
  expect(before.scrollTop).toBeGreaterThan(100);

  await target.click();
  await expect(page).toHaveURL(detailUrlPattern, { timeout: 20_000 });
  await page.goBack();
  await expect(page).toHaveURL(listUrl);
  // The list is restored from the persisted snapshot (a no-refresh POP), so all
  // rows come back together — a simple wait, no "Load more" nudge needed.
  await expect(rows).toHaveCount(rowCount, { timeout: 20_000 });
  await expect(target).toBeVisible({ timeout: 20_000 });

  // The restore re-applies the saved scroll position asynchronously after the
  // rows settle — poll generously so a slow re-application under load doesn't
  // read a mid-restore frame.
  await expect
    .poll(
      async () => {
        const after = await readViewport(target);
        return Math.abs(after.scrollTop - before.scrollTop);
      },
      { timeout: 20_000 }
    )
    .toBeLessThanOrEqual(2);
  await expect
    .poll(
      async () => {
        const after = await readViewport(target);
        return Math.abs(after.anchorOffset - before.anchorOffset);
      },
      { timeout: 20_000 }
    )
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

  const setting = await env.api.getSetting("WORKSPACE_PROFILE");
  originalExternalUrl =
    (setting?.value as { workspaceProfile?: { externalUrl?: string } })
      ?.workspaceProfile?.externalUrl ?? "";

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
    // Below Tailwind's sm breakpoint, the banner CTA moves onto its own row.
    viewport: { width: 600, height: 720 },
  });
  page = await sharedContext.newPage();

  // Seed the way the UI does: each plan is created together with its draft
  // review issue (createPlanWithDraftReview) and then submitted for review, so
  // every one surfaces in the Issues list (draft issues are hidden there). An
  // issueless bare-API plan is not a reachable product state. This exercises
  // the real create + submit workflow and stays correct if it changes again.
  await page.setViewportSize({ width: 1280, height: 900 }); // the create page needs room
  for (let i = 0; i < ISSUE_COUNT; i++) {
    await createDatabaseChangePlanViaUI(page, {
      baseURL: env.baseURL,
      projectId,
      database: env.database,
      title: `${searchToken} issue ${i}`,
      sql: "SELECT 1;",
    });
    await submitDraftForReviewViaUI(page);
  }
  await page.setViewportSize({ width: 600, height: 720 });
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
