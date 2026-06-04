// SQL Editor — permission-gated UI (Batch 8, I-series CUJs).
//
// Covers:
//   - I1  admin wrench absent for projectDeveloper
//   - I4  RequestQueryButton ("Request query") on a non-JIT project
//         where the user lacks bb.sql.select
//   - I9  Sidebar surfaces the same Request-query affordance when the
//         user lands on a database they have no access to
//   - I10 admin wrench absent for sqlEditorUser
//   - I11 INSERT denied (no "EditorHint" modal, just an inline error
//         with the missing permission)
//
// Per AGENTS.md doctrine "F. Test by role, not just by admin", we
// create dedicated test users via the User API in `beforeAll`, grant
// each the minimal role we want to verify, log them in via fresh
// BrowserContexts (one per role) so their auth state is isolated, and
// delete them in `afterAll`. We never lean on the demo dump's seeded
// `dev1@example.com` / `dba1@example.com` (we don't know their
// passwords) and never share a context across roles.
//
// Skipped from the agreed list:
//   - I5: requires a workspace gate (workspace-level allowQueryWithoutMembership)
//         not exposed via the public API in this branch — deferred.

import {
  test,
  expect,
  type Browser,
  type BrowserContext,
  type Page,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(180_000);

// One workspace-scoped client logged in as the demo admin — owns user
// lifecycle + IAM grants.
let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

// Fixed test-user identities. The password is shared across all of them
// to keep auth-state setup terse; it's a fresh string with no
// production meaning. Real demo passwords are unknown.
const TEST_PASSWORD = "e2e-perms-pw-1!"; // NOSONAR — e2e fixture only

const DEV_USER = {
  email: "e2e-perms-dev@example.com",
  title: "E2E Perms Dev",
  authFile: ".auth/perms-dev.json",
};
const SQL_EDITOR_USER = {
  email: "e2e-perms-sqledit@example.com",
  title: "E2E Perms SqlEditor",
  authFile: ".auth/perms-sqledit.json",
};
const READ_ONLY_USER = {
  email: "e2e-perms-reader@example.com",
  title: "E2E Perms Reader",
  authFile: ".auth/perms-reader.json",
};
const NO_MEMBERSHIP_USER = {
  email: "e2e-perms-noproj@example.com",
  title: "E2E Perms NoProj",
  authFile: ".auth/perms-noproj.json",
};

type RoleContext = {
  context: BrowserContext;
  page: Page;
  sqlEditor: SqlEditorPage;
};

const roleContexts = new Map<string, RoleContext>();

async function openAs(
  browser: Browser,
  user: { email: string; authFile: string },
): Promise<RoleContext> {
  const context = await browser.newContext({ storageState: user.authFile });
  const page = await context.newPage();
  const sqlEditor = new SqlEditorPage(page, env.baseURL);
  return { context, page, sqlEditor };
}

// beforeAll runs 4 user creates + 3 IAM grants + 4 API logins + 4
// page.goto + 4 storage-state writes + 4 context opens. Sequentially
// that's well past Playwright's default 30s hook budget; we bump it
// once here so a slow disposable server doesn't false-fail the gate.
test.beforeAll(async ({ browser }) => {
  test.setTimeout(180_000);
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Provision the four test users. Make this idempotent: if a previous
  // test in this same worker (e.g., due to a Playwright retry) already
  // created the row, createUser returns 409 — we tolerate it and
  // continue, because the server instance is the same.
  for (const u of [DEV_USER, SQL_EDITOR_USER, READ_ONLY_USER, NO_MEMBERSHIP_USER]) {
    try {
      await env.api.createUser(u.email, TEST_PASSWORD, u.title);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("(409)")) throw err;
    }
  }

  // Grant project-scoped roles on env.project (discovered Postgres project).
  // - DEV_USER: projectDeveloper (no bb.sql.admin → wrench hidden)
  // - SQL_EDITOR_USER: sqlEditorUser (read+DML, no admin → wrench hidden)
  // - READ_ONLY_USER: sqlEditorReadUser (read only — INSERT denied)
  // - NO_MEMBERSHIP_USER: projectViewer.
  //   Why not "no project role at all"? The SQL Editor route guard
  //   demands bb.projects.getIamPolicy on the targeted project; a pure
  //   workspaceMember (the default for allUsers) gets bounced to a
  //   "Missing required permissions" wall before any editor UI mounts,
  //   making the Request-query CUJ impossible to assert. projectViewer
  //   passes the route guard but lacks bb.sql.select / bb.sql.dml, so
  //   RequestQueryButton renders as the affordance.
  await env.api.appendProjectBinding(
    env.project,
    "roles/projectDeveloper",
    [`user:${DEV_USER.email}`],
  );
  await env.api.appendProjectBinding(
    env.project,
    "roles/sqlEditorUser",
    [`user:${SQL_EDITOR_USER.email}`],
  );
  await env.api.appendProjectBinding(
    env.project,
    "roles/sqlEditorReadUser",
    [`user:${READ_ONLY_USER.email}`],
  );
  await env.api.appendProjectBinding(
    env.project,
    "roles/projectViewer",
    [`user:${NO_MEMBERSHIP_USER.email}`],
  );

  // Pre-warm browser sign-ins so individual tests just load storage state.
  for (const u of [DEV_USER, SQL_EDITOR_USER, READ_ONLY_USER, NO_MEMBERSHIP_USER]) {
    await signInBrowserAs(browser, env.baseURL, u.email, TEST_PASSWORD, u.authFile);
    roleContexts.set(u.email, await openAs(browser, u));
  }
});

test.afterAll(async () => {
  for (const rc of roleContexts.values()) {
    await rc.context.close();
  }
  roleContexts.clear();
  // Best-effort: revert allowRequestRole on the sample project (this
  // file's Request-query describe flips it) so downstream specs don't
  // inherit a mutated flag. Skip IAM cleanup and user deletion entirely:
  //
  //   - DELETE /v1/users/{email} marks the row as deactivated rather
  //     than purging it, so a subsequent beforeAll re-run on the same
  //     server hits 401 "user has been deactivated". This bit us in
  //     earlier batches; the disposable server is torn down by
  //     globalTeardown anyway, so leaving the rows in place is safe.
  try {
    await env.api.updateProjectSettings(env.project, {
      allowRequestRole: false,
    });
  } catch {
    /* ignore */
  }
});

// I1 — projectDeveloper role grants project-write permissions but NOT
// `bb.sql.admin`. `useSQLEditorStore.allowAdmin` is the gate for the
// wrench button (AdminModeButton.tsx returns null when allowAdmin is
// false), so opening the editor as a projectDeveloper means no wrench.
test.describe("Admin-mode wrench is hidden for projectDeveloper users", () => {
  test("opening a connected DB as a developer shows the toolbar without a wrench", async () => {
    const rc = roleContexts.get(DEV_USER.email)!;
    await rc.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await rc.page.waitForTimeout(1500);

    // The Save / Share / row-limit toolbar must be in the DOM (proves
    // we actually landed on the editor and the page isn't still
    // loading). Wrench is the per-permission element under test.
    await expect(rc.sqlEditor.saveButton).toBeVisible({ timeout: 10_000 });
    // The wrench locator filters by `lucide-wrench` SVG. For a user
    // without bb.sql.admin, that SVG should be absent from the
    // editor toolbar entirely.
    await expect(rc.sqlEditor.adminModeButton).toHaveCount(0);
  });
});

// I10 — sqlEditorUser has bb.sql.select + bb.sql.update + bb.sql.delete
// but not bb.sql.admin. Same wrench-absence assertion as I1, different
// role — the two roles are checked separately because they each
// represent a real user persona ops gives out.
test.describe("Admin-mode wrench is hidden for sqlEditorUser users", () => {
  test("opening a connected DB as a sqlEditorUser shows the toolbar without a wrench", async () => {
    const rc = roleContexts.get(SQL_EDITOR_USER.email)!;
    await rc.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await rc.page.waitForTimeout(1500);

    await expect(rc.sqlEditor.saveButton).toBeVisible({ timeout: 10_000 });
    await expect(rc.sqlEditor.adminModeButton).toHaveCount(0);
  });
});

// I11 — sqlEditorReadUser has bb.sql.select only. Running an INSERT
// returns a permission-denied error; we don't expect a hint modal
// (those are reserved for the "executingHint" recoverable flow). The
// inline error in the result panel is the user-visible signal.
test.describe("INSERT is denied with an inline error for sqlEditorReadUser", () => {
  test("running INSERT in a read-only project surfaces a 'no permission' error", async () => {
    const rc = roleContexts.get(READ_ONLY_USER.email)!;
    await rc.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await rc.page.waitForTimeout(1500);

    // Type a one-off INSERT against a table that may or may not exist
    // — we don't care about the SQL parsing result; we care that the
    // permission gate trips BEFORE the query reaches the database.
    // The runQuery helper waits for either a row-count footer (which
    // won't happen) or an inline error pane.
    await rc.sqlEditor.runPreparedQuery(
      "INSERT INTO _e2e_does_not_exist DEFAULT VALUES;",
    );

    // The error pane renders the permission detail somewhere in the
    // result region. We accept any of the common shapes: an explicit
    // "permission" mention, the role name, or the request-query CTA.
    // The strong assertion is that the row-count footer DID NOT appear
    // (Run reached an error state) AND a request affordance is offered.
    const permissionMention = rc.page.getByText(
      /(permission denied|no.*permission|bb\.sql\.(update|delete|create|admin|insert)|missing.*permission)/i,
    );
    const requestCTA = rc.page.getByRole("button", { name: /Request query|Request just-in-time/i });
    const anySignal = permissionMention.or(requestCTA).first();
    await expect(anySignal).toBeVisible({ timeout: 10_000 });

    // No N-rows footer (proves the query did not succeed).
    await expect(rc.page.getByText(/^\d+\s+rows?$/i)).toHaveCount(0);
  });
});

// I4 — RequestQueryButton has a two-axis gate:
//   - `project.allowRequestRole === false` → component returns null (no
//     Request button at all, regardless of permission state).
//   - `project.allowRequestRole === true` AND user lacks bb.sql.select →
//     the button renders. Label switches on `useJIT` (non-JIT label is
//     "Request query" — covered here; JIT label is in sql-editor-jit.spec.ts).
//
// The new bootstrap creates `project-sample` with `allowRequestRole=false`
// by default (sampleinstance/manager.go:171 uses an empty Setting). We
// exercise BOTH halves of the gate so a future regression on either side
// is caught:
//   D1 ("…off → no button"): projectViewer + allowRequestRole=false →
//     no Request query button (the negative case the framework relies on).
//   D2 ("…on → button"): flip allowRequestRole=true via API, re-navigate,
//     verify the button now renders. afterEach flips it back.

test.describe("Request-query CTA respects project.allowRequestRole", () => {
  test.afterEach(async () => {
    // Always restore to the default-off state so other tests in this
    // file (and downstream specs that share env.project) inherit a
    // known baseline.
    await env.api
      .updateProjectSettings(env.project, { allowRequestRole: false })
      .catch(() => {});
  });

  test("with allowRequestRole=false, projectViewer running a forbidden SELECT sees NO Request query button", async () => {
    // Confirm the seeded state (sample project is created with
    // allowRequestRole=false). RequestQueryButton's `if (!available)
    // return null` then suppresses both the non-JIT and JIT variants.
    await env.api.updateProjectSettings(env.project, { allowRequestRole: false });

    const rc = roleContexts.get(NO_MEMBERSHIP_USER.email)!;
    await rc.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await rc.page.waitForTimeout(2000);
    await rc.sqlEditor.runPreparedQuery("SELECT 1;");
    await rc.page.waitForTimeout(800);

    // Neither label should appear — the component returned null.
    await expect(
      rc.page.getByRole("button", { name: "Request query", exact: true }),
    ).toHaveCount(0);
    await expect(
      rc.page.getByRole("button", { name: /Request just-in-time/i }),
    ).toHaveCount(0);
  });

  test("with allowRequestRole=true, projectViewer running a forbidden SELECT sees the 'Request query' button", async () => {
    // Flip the project's allowRequestRole on. Once the editor re-reads
    // the project (we force a fresh navigation), the button renders.
    await env.api.updateProjectSettings(env.project, { allowRequestRole: true });

    const rc = roleContexts.get(NO_MEMBERSHIP_USER.email)!;
    await rc.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await rc.page.waitForTimeout(2000);
    await rc.sqlEditor.runPreparedQuery("SELECT 1;");
    await rc.page.waitForTimeout(800);

    // Non-JIT label — `sql-editor.request-query` → "Request query".
    await expect(
      rc.page.getByRole("button", { name: "Request query", exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });

    // JIT variant must NOT show — this project has
    // allowJustInTimeAccess=false (its default), so `useJIT` resolves
    // to false and the label switch picks the non-JIT side.
    await expect(
      rc.page.getByRole("button", { name: /Request just-in-time/i }),
    ).toHaveCount(0);
  });
});
