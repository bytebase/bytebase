// Issue detail — markdown comments & description (link rendering).
//
// BYT-9664 (FIXED, #20537): since 3.17.1, links rendered in issue comments (and
// the issue description) navigated the CURRENT tab — a full-page redirect away
// from the issue detail page — instead of opening in a new tab as they did
// through 3.17.0. Root cause: the Vue→React migration replaced the old Vue
// markdown preview (a DOMPurify hook forced target="_blank" + a sandboxed
// iframe) with a React MarkdownEditor that emitted plain <a href> via
// dangerouslySetInnerHTML, so links defaulted to same-tab navigation. The old
// code also never set `rel`, leaving a reverse-tabnabbing gap.
//
// The fix's sanitizePreviewHtml (MarkdownEditor.tsx) sets target="_blank" AND
// rel="noopener noreferrer" on every rendered <a>. The same component renders
// BOTH comments and the issue description, so this file locks both surfaces
// (comment = the reported surface; description = the sibling site from the
// CUJ analysis).
//
// Seeds the comment + description via the API (CreateIssueComment / issue
// description) rather than driving the review popover — the bug lives in the
// read-only render path, which both surfaces share.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";

test.describe.configure({ mode: "serial" });

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;

const STAMP = Date.now();
// Distinct external URLs so the rendered <a> elements are unambiguous to locate.
const MD_LINK_URL = `https://example.com/byt9664-comment-${STAMP}`;
const BARE_URL = `https://example.com/byt9664-bare-${STAMP}`;
const DESC_LINK_URL = `https://example.com/byt9664-desc-${STAMP}`;

let issueName = "";
let issueUrl = "";

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);

  // Create a real plan + issue. The description carries a markdown link (sibling
  // surface); the comment is posted right after (the reported surface).
  const sheet = await env.api.createSheet(
    env.project,
    "SELECT 1; -- byt9664",
  );
  const plan = await env.api.createPlan(env.project, `BYT-9664 ${STAMP}`, [
    { id: `spec-${STAMP}`, targets: [env.database], sheet },
  ]);
  const issue = await env.api.createIssue(
    env.project,
    `BYT-9664 ${STAMP}`,
    plan.name,
    `Reference: [design doc](${DESC_LINK_URL})`,
  );
  issueName = issue.name;
  await env.api.createIssueComment(
    issueName,
    `[docs](${MD_LINK_URL}) and a bare link ${BARE_URL}`,
  );

  const projectId = env.project.split("/").pop()!;
  const issueId = issueName.split("/").pop()!;
  issueUrl = `${env.baseURL}/projects/${projectId}/issues/${issueId}`;

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  // Keep the run hermetic: never actually load the external sites. The popup /
  // navigation still fires (which is what we assert) — the request just aborts.
  await sharedContext.route(/example\.com/, (route) => route.abort());
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("Issue comment links open in a new tab with rel=noopener noreferrer (BYT-9664)", () => {
  test("rendered comment links carry target=_blank + rel, and clicking does not redirect the issue page", async () => {
    test.setTimeout(120_000);

    await page.goto(issueUrl);
    await page.keyboard.press("Escape").catch(() => {});
    await page.waitForLoadState("networkidle").catch(() => {});

    // The markdown link and the linkified bare URL both render as <a> in the
    // comment body. Locate by href (unique per run).
    const mdLink = page.locator(`a[href="${MD_LINK_URL}"]`).first();
    const bareLink = page.locator(`a[href="${BARE_URL}"]`).first();
    await expect(mdLink).toBeVisible({ timeout: 15_000 });
    await expect(bareLink).toBeVisible({ timeout: 10_000 });

    // Attribute oracle — exactly what the fix restored (and the unit test in
    // MarkdownEditor.test.tsx asserts). Pre-fix: no target, no rel.
    for (const link of [mdLink, bareLink]) {
      await expect(link).toHaveAttribute("target", "_blank");
      await expect(link).toHaveAttribute(
        "rel",
        /(?=.*noopener)(?=.*noreferrer)/,
      );
    }

    // Behavioral oracle — clicking opens a NEW tab while the issue page stays
    // put (the user-visible symptom was a full-page redirect away from the
    // issue). The external request is aborted by the context route, so the
    // popup opens but loads nothing — the page event is all we need.
    const issuePath = new URL(issueUrl).pathname;
    const [popup] = await Promise.all([
      sharedContext.waitForEvent("page"),
      mdLink.click(),
    ]);
    // The issue page stays put. (The app legitimately appends its own `?spec=`
    // query param on load, so compare the PATHNAME — a same-tab redirect to the
    // external link would change the pathname/host entirely.)
    expect(
      new URL(page.url()).pathname,
      "clicking a comment link must NOT redirect the issue page",
    ).toBe(issuePath);
    expect(page.url(), "the issue page must not navigate to the link target").not.toContain(
      "example.com",
    );
    await popup.close();
  });
});

test.describe("Issue description links open in a new tab (BYT-9664 sibling surface)", () => {
  test("the rendered description link carries target=_blank + rel", async () => {
    test.setTimeout(120_000);

    await page.goto(issueUrl);
    await page.keyboard.press("Escape").catch(() => {});
    await page.waitForLoadState("networkidle").catch(() => {});

    // The issue description renders through the same MarkdownEditor read-only
    // path as comments — so the same sanitize contract must apply.
    const descLink = page.locator(`a[href="${DESC_LINK_URL}"]`).first();
    await expect(descLink).toBeVisible({ timeout: 15_000 });
    await expect(descLink).toHaveAttribute("target", "_blank");
    await expect(descLink).toHaveAttribute(
      "rel",
      /(?=.*noopener)(?=.*noreferrer)/,
    );
  });
});
