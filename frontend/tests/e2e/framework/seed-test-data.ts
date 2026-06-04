import { BytebaseApiClient } from "./api-client";

// External URL we set on the workspace_profile setting so the
// "Bytebase has not configured --external-url" banner stops rendering.
// The frontend's BannersWrapper.tsx checks `serverInfo.externalUrl`
// truthiness; any non-empty URL silences the banner.
const TEST_EXTERNAL_URL = "https://e2e.bytebase.test";

/**
 * Provisions WORKSPACE-LEVEL baseline data on top of `api.setupSample()`.
 * Called once per disposable server boot from setup-project.ts.
 *
 * Scope is intentionally narrow: only changes that EVERY spec wants. Any
 * per-spec test data (extra projects, feature flag flips, masking config
 * etc.) lives in the spec's own beforeAll so one spec's setup can't
 * shift another spec's expectations.
 *
 * Goal:
 *   - Silence the external-URL banner (so its wrench-icon "Configure now"
 *     button doesn't shadow the SQL editor's admin-mode wrench locator).
 *
 * Specifically does NOT:
 *   - Create extra projects (would change the SQL editor's default
 *     landing project; see the worksheet specs that gotoHome() and
 *     expect to land on `project-sample`).
 *   - Flip per-project feature flags (allowRequestRole,
 *     allowJustInTimeAccess) — those are SUT for individual specs.
 */
export async function seedTestData(api: BytebaseApiClient): Promise<void> {
  try {
    await api.setWorkspaceExternalUrl(TEST_EXTERNAL_URL);
  } catch (err) {
    // The setting may not exist yet on a fresh server; updateMask
    // semantics + allowMissing on the gateway should make PATCH a
    // create-or-update. Surface the error so a test infra change
    // doesn't silently drop the banner-suppression.
    throw new Error(
      `seedTestData: failed to set external URL — ${err instanceof Error ? err.message : err}`,
    );
  }

  // Enterprise license auto-activates a built-in workspace approval
  // "Fallback Rule" (Moderate · Requires project owner approval when
  // no other rules match) that fires for every CHANGE_DATABASE plan.
  // Without it, requireIssueApproval=false projects auto-create the
  // rollout — that's the baseline tests assume. Free plan has no rule
  // at all, so clearing here is a no-op there but makes the licensed
  // baseline match the free-plan one. Tests that DO want approval set
  // their own rule in beforeAll and restore on afterAll.
  try {
    await api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    );
  } catch {
    // Free plan: setting won't accept rules without license; rules
    // would already be empty on that path. Swallow silently.
  }
}

// Re-exported identifiers used by tests that need their own additional
// project. Created on-demand by `ensureSecondaryProject()` — kept here
// so specs share a single canonical name/title.
export const SECONDARY_PROJECT_ID = "e2e-secondary";
export const SECONDARY_PROJECT_TITLE = "E2E Secondary";

/**
 * Creates the secondary project on demand. Idempotent (tolerates "already
 * exists" on rerun). Use from spec beforeAll when the test depends on a
 * 2nd project being present (e.g. the project-switcher CUJ).
 *
 * Why this is on-demand rather than in seedTestData: creating a 2nd
 * project at global setup time alters the SQL editor's default landing
 * project on /sql-editor (no URL params), which silently breaks any
 * spec that uses gotoHome() and expects `project-sample`. Per-spec
 * creation avoids that side effect.
 */
export async function ensureSecondaryProject(api: BytebaseApiClient): Promise<void> {
  try {
    await api.createProject(SECONDARY_PROJECT_ID, SECONDARY_PROJECT_TITLE);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    if (!msg.includes("(409)") && !msg.includes("already exists")) throw err;
  }
}
