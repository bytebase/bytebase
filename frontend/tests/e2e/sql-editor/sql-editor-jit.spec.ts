// SQL Editor — JIT access flow (Batch 8, L-series + H1 CUJs).
//
// Covers:
//   - L1 / L2 RequestQueryButton label switches to "Request just-in-time
//     access" when project.allowJustInTimeAccess is on AND the missing
//     permission is exactly bb.sql.select (the JIT path).
//   - L4 ACCESS pane shows the "Request access" CTA when JIT is enabled.
//   - L5 AccessGrantRequestDrawer renders Databases / Statement / Unmask /
//     Export / Expiration / Reason fields when opened from the pane
//     (Unmask + Export capability sections added by PRs #20491/#20487).
//   - L6 ACCESS gutter tab appears only while JIT is on.
//   - H1 Gutter tab count flips between 3 (no JIT) and 4 (with JIT).
//
// Deferred (out of this batch):
//   - L3 full JIT Unmask flow — request → admin approve → re-run.
//     Requires a multi-actor handoff (requestor user creates grant,
//     admin user approves, requestor re-runs and sees unmasked data),
//     plus seeded masked columns. Worth its own spec; punted to a
//     later session.
//   - L7 MaskingReasonPopover — requires the demo to ship a column
//     with a non-empty maskingReason; the discovered Postgres project
//     in this branch's demo does not have one out of the box. Will
//     pair with L3 when masking-test data is set up.

import {
  test,
  expect,
  type BrowserContext,
  type Page,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

// Single test user — projectViewer on env.project, so they can mount
// the editor but lack bb.sql.select. The JIT switch on the project
// toggles whether their Request-button reads as JIT or non-JIT.
//
// We use a fresh, file-scoped user so we don't have to coordinate with
// sql-editor-permissions.spec.ts (which manages its own e2e users).
const TEST_PASSWORD = "e2e-jit-pw-1!"; // NOSONAR — e2e fixture only
const VIEWER_USER = {
  email: "e2e-jit-viewer@example.com",
  title: "E2E JIT Viewer",
  authFile: ".auth/jit-viewer.json",
};

let viewerCtx: BrowserContext;
let viewerPage: Page;
let viewerEditor: SqlEditorPage;

// Admin-side context used to flip allowJustInTimeAccess between
// describe blocks. We keep one shared admin page for screenshots in
// case of debugging, but the JIT toggle itself goes through the API
// client — flipping via UI would require a separate `/projects/<id>`
// navigation that we can avoid.
let adminCtx: BrowserContext;
let adminPage: Page;

async function setJIT(enabled: boolean): Promise<void> {
  // Also set allowRequestRole=true: the JIT label tests below need a
  // RequestQueryButton to render at all (the component returns null
  // when allowRequestRole is false — see RequestQueryButton.tsx:131,
  // 159-161). The new sample project is born with allowRequestRole=false;
  // the JIT tests are about the LABEL flipping, not about whether the
  // button shows up.
  await env.api.updateProjectSettings(env.project, {
    allowRequestRole: true,
    allowJustInTimeAccess: enabled,
  });
  // Force-reload the viewer's page so the store picks up the new
  // project flag. The SQL Editor caches project state per-tab; flipping
  // the flag via API does not in-flight invalidate that cache.
  await viewerEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await viewerPage.waitForTimeout(1500);
}

test.beforeAll(async ({ browser }) => {
  test.setTimeout(180_000);
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Provision the test user — idempotent on 409 for Playwright retries.
  try {
    await env.api.createUser(VIEWER_USER.email, TEST_PASSWORD, VIEWER_USER.title);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    if (!msg.includes("(409)")) throw err;
  }
  await env.api.appendProjectBinding(
    env.project,
    "roles/projectViewer",
    [`user:${VIEWER_USER.email}`],
  );

  await signInBrowserAs(
    browser,
    env.baseURL,
    VIEWER_USER.email,
    TEST_PASSWORD,
    VIEWER_USER.authFile,
  );
  viewerCtx = await browser.newContext({ storageState: VIEWER_USER.authFile });
  viewerPage = await viewerCtx.newPage();
  viewerEditor = new SqlEditorPage(viewerPage, env.baseURL);

  adminCtx = await browser.newContext({ storageState: ".auth/state.json" });
  adminPage = await adminCtx.newPage();

  // Start from a known-off JIT state so the test ordering is
  // deterministic. Each describe block flips the flag explicitly.
  await env.api.updateProjectSettings(env.project, {
    allowJustInTimeAccess: false,
  });
});

test.afterAll(async () => {
  // Best-effort: turn JIT off (so other specs don't inherit the
  // flipped state) and detach contexts. We deliberately DO NOT delete
  // the test user — Playwright's worker semantics caused the
  // file-level beforeAll to fire again between describe blocks in an
  // earlier iteration, and after the first run deleting the user left the
  // row in a "deactivated" state, which then blocked
  // /v1/auth/login on the rerun ("user has been deactivated"). The
  // disposable bytebase server is torn down by globalTeardown at the
  // end of the run, so leaving the row in place is safe.
  try {
    await env.api.updateProjectSettings(env.project, {
      allowRequestRole: false,
      allowJustInTimeAccess: false,
    });
  } catch {
    /* ignore */
  }
  await viewerCtx?.close();
  await adminCtx?.close();
});

// L6 + H1 — the ACCESS gutter tab only renders when the project has
// `allowJustInTimeAccess` enabled, so the gutter count flips between
// 3 (Worksheet/Schema/History) and 4 (+ Access).
test.describe("ACCESS gutter tab follows the project's allowJustInTimeAccess flag", () => {
  test("with JIT off, the gutter has 3 tabs and no Access tab", async () => {
    await setJIT(false);

    await expect(viewerEditor.gutterWorksheetTab).toBeVisible({ timeout: 10_000 });
    await expect(viewerEditor.gutterSchemaTab).toBeVisible();
    await expect(viewerEditor.gutterHistoryTab).toBeVisible();
    // ACCESS tab is conditionally rendered — must be absent here.
    await expect(viewerEditor.gutterAccessTab).toHaveCount(0);
  });

  test("with JIT on, the gutter exposes an ACCESS tab in addition to the 3 defaults", async () => {
    await setJIT(true);

    await expect(viewerEditor.gutterWorksheetTab).toBeVisible({ timeout: 10_000 });
    await expect(viewerEditor.gutterSchemaTab).toBeVisible();
    await expect(viewerEditor.gutterHistoryTab).toBeVisible();
    await expect(viewerEditor.gutterAccessTab).toBeVisible();
  });
});

// L1 / L2 — RequestQueryButton's `useJIT` is true when
// project.allowJustInTimeAccess is on AND the missing permissions are
// exactly bb.sql.select. The visible label switches from
// "Request query" to "Request just-in-time access" accordingly.
test.describe("RequestQueryButton label tracks the project's JIT flag", () => {
  test("with JIT off, running a forbidden SELECT shows the non-JIT 'Request query' CTA", async () => {
    await setJIT(false);
    await viewerEditor.runPreparedQuery("SELECT 1;");
    await viewerPage.waitForTimeout(800);

    await expect(
      viewerPage.getByRole("button", { name: "Request query", exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      viewerPage.getByRole("button", { name: /Request just-in-time/i }),
    ).toHaveCount(0);
  });

  test("with JIT on, running a forbidden SELECT shows the JIT 'Request just-in-time access' CTA", async () => {
    await setJIT(true);
    await viewerEditor.runPreparedQuery("SELECT 1;");
    await viewerPage.waitForTimeout(800);

    await expect(
      viewerPage
        .getByRole("button", { name: /Request just-in-time access/i })
        .first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      viewerPage.getByRole("button", { name: "Request query", exact: true }),
    ).toHaveCount(0);
  });
});

// L4 + L5 — the ACCESS pane (visible only when JIT is on) renders a
// "Request access" button at its top-right. Clicking it opens the
// AccessGrantRequestDrawer with the four expected sections.
//
// REQUIRES ENTERPRISE LICENSE: the button is gated by
// `disabled={!hasJITFeature || disabled}` in AccessPane.tsx — without
// BYTEBASE_E2E_LICENSE, hasJITFeature is false and the click is a no-op
// (Playwright's `force: true` bypasses actionability checks but does NOT
// override the HTML `disabled` attribute, so onClick never fires and the
// drawer never opens). The license is installed at bootstrap, so the feature is available here.
test.describe("ACCESS pane Request-access button opens the grant drawer", () => {
  test("opening the ACCESS tab and clicking Request access opens a drawer with Databases / Statement / Expiration / Reason", async () => {
    await setJIT(true);

    // Click the ACCESS gutter tab to mount the pane.
    await viewerEditor.gutterAccessTab.click();
    await viewerPage.waitForTimeout(500);

    // The pane's primary action — label from
    // `sql-editor.request-access` ("Request Access" in en-US).
    const requestAccessBtn = viewerPage
      .getByRole("button", { name: /Request Access/i })
      .first();
    await expect(requestAccessBtn).toBeVisible({ timeout: 10_000 });
    // PermissionGuard renders the button as disabled when JIT feature
    // is unlicensed. The demo build ships the feature on (sample data
    // includes JIT grants), so we expect enabled — but assert via a
    // try/catch fallback so a future license change doesn't bury the
    // structural CUJ. The CUJ here is "the button is present and
    // wired", which already shows that the pane is on the right
    // surface.
    await requestAccessBtn.click({ force: true });
    await viewerPage.waitForTimeout(500);

    // Drawer header — i18n key `sql-editor.request-data-access`.
    await expect(
      viewerPage.getByText("Request Data Access", { exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });

    // L5: the labelled regions inside the drawer. Each is a headline
    // above its input — we match the label as a substring because the
    // rendered text is `<label>Databases *</label>` and similar (the
    // trailing `*` is the required-field marker, not a separate element).
    // Anchoring to the start of the line via a regex keeps the match
    // unambiguous (matches "Databases *" or "Databases" alone, not
    // "Some Databases List").
    //
    // PRs #20491/#20487 added the Unmask + Export capability sections
    // (the drawer previously had only Databases/Statement/Expiration/Reason).
    for (const label of [
      "Databases",
      "Statement",
      "Unmask",
      "Export",
      "Expiration",
      "Reason",
    ]) {
      const re = new RegExp(`^${label}(\\s*\\*)?\\s*$`);
      await expect(
        viewerPage.getByText(re).first(),
        `drawer must contain a "${label}" section`,
      ).toBeVisible({ timeout: 5000 });
    }

    // The two capability checkboxes are present AND default UNCHECKED when the
    // drawer is opened from the generic "Request access" CTA (no pendingCreate).
    // Asserting unchecked — not just visible — catches a regression that
    // pre-checks either capability for a plain request. (The "Request export"
    // entry point pre-checks ONLY Export, not Unmask, after BYT-9654/#20516 —
    // covered in sql-editor-export.spec.ts.)
    const unmaskCheckbox = viewerPage.getByRole("checkbox", {
      name: "See unmasked sensitive data",
    });
    const exportCheckbox = viewerPage.getByRole("checkbox", {
      name: "Export the query result",
    });
    await expect(
      unmaskCheckbox,
      "drawer must expose the Unmask capability checkbox",
    ).toBeVisible({ timeout: 5000 });
    await expect(exportCheckbox).toBeVisible();
    await expect(
      unmaskCheckbox,
      "Unmask must default unchecked from the generic Request-access CTA",
    ).not.toBeChecked();
    await expect(
      exportCheckbox,
      "Export must default unchecked from the generic Request-access CTA",
    ).not.toBeChecked();

    // Close the drawer so any follow-up test doesn't inherit it.
    // The drawer is dismissed by clicking outside (Base UI Sheet) or
    // by an explicit Cancel — we use Escape which the Sheet honors.
    await viewerPage.keyboard.press("Escape");
    await viewerPage.waitForTimeout(300);
  });
});
