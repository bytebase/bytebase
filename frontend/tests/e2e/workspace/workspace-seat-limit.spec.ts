// Workspace seat limit — over-limit IAM edits (BYT-9633).
//
// BYT-9633 (FIXED, #20492 / #20497; cherry-picked to 3.18.1): on a seat-limited
// license, an over-limit workspace rejected EVERY IAM edit — including
// seat-neutral ones — so granting roles AND creating service accounts both
// failed with "workspace has N users, exceeding the limit of M". Two root causes:
//   (a) the seat count included soft-deleted (deactivated) principals, and
//   (b) SetIamPolicy rejected any edit where newCount >= oldCount (not just
//       edits that INCREASE the count).
// The fix excludes soft-deleted principals from the count and only rejects edits
// that strictly increase it — so a service account (a `serviceAccount:` member,
// seat-neutral) can be created over-limit, while adding another end user is still
// rejected. (Mirrors backend TestSeatLimitAllowsServiceAccountWhenOverLimit.)
//
// SAFETY: this is the only spec that needs the FREE plan, so it DROPS the license
// in beforeAll and RESTORES it in afterAll. It lives in workspace/ (the last test
// directory) so the drop happens only after every other spec has finished, and
// afterAll re-installs + verifies the license BEFORE anything else so a failure
// here cannot strand later runs. Never reorder this earlier.

import { test, expect } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient, IamPolicy } from "../framework/api-client";

test.describe.configure({ mode: "serial" });

const FREE_SEAT_LIMIT = 20;
const STAMP = Date.now();
const SEAT_PASSWORD = "e2e-seat-pw-1!"; // NOSONAR — e2e fixture only
const SA_PREFIX = `e2e-seat-bot-${STAMP}`;

let env: TestEnv & { api: BytebaseApiClient };
let workspace = "";
let policySnapshot: IamPolicy | undefined;
const createdUserEmails: string[] = [];
let createdSaEmail = "";
let licenseRestored = false;

test.beforeAll(async () => {
  test.setTimeout(240_000);
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  ({ workspace } = await env.api.getActuatorInfo());

  // Snapshot the workspace IAM policy so afterAll can restore it exactly.
  policySnapshot = await env.api.getWorkspaceIamPolicy(workspace);

  // Push the workspace over the FREE seat limit while still licensed. Create
  // enough ACTIVE users so the distinct user: count is FREE_SEAT_LIMIT + 1.
  const info = await env.api.getActuatorInfo();
  const baseline = info.userCountInIam ?? 1;
  const toCreate = Math.max(FREE_SEAT_LIMIT + 1 - baseline, 1);

  const newMembers: string[] = [];
  for (let i = 0; i < toCreate; i++) {
    const email = `e2e-seat-${STAMP}-${i}@example.com`;
    try {
      await env.api.createUser(email, SEAT_PASSWORD, `Seat ${i}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("(409)")) throw err;
    }
    createdUserEmails.push(email);
    newMembers.push(`user:${email}`);
  }

  // Bind all new users as workspaceMember in one write (over the limit now).
  const policy = await env.api.getWorkspaceIamPolicy(workspace);
  const memberBinding = policy.bindings.find(
    (b) => b.role === "roles/workspaceMember",
  );
  if (memberBinding) {
    memberBinding.members = [
      ...new Set([...(memberBinding.members ?? []), ...newMembers]),
    ];
  } else {
    policy.bindings.push({ role: "roles/workspaceMember", members: newMembers });
  }
  await env.api.setWorkspaceIamPolicy(workspace, policy);

  // Drop the license → FREE plan, 20-seat limit. Now over-limit.
  await env.api.uploadLicense("");
  await expect
    .poll(async () => (await env.api.getSubscription()).plan, {
      timeout: 30_000,
      message: "license drop should put the workspace on the FREE plan",
    })
    .toBe("FREE");
  // Confirm the seat count is genuinely over the FREE limit.
  const overLimit = await env.api.getActuatorInfo();
  expect(
    overLimit.userCountInIam ?? 0,
    "the workspace must be over the FREE seat limit for this test to be meaningful",
  ).toBeGreaterThan(FREE_SEAT_LIMIT);
});

test.afterAll(async () => {
  // RESTORE THE LICENSE FIRST — before any other teardown — so a failure
  // elsewhere can't leave the server on the FREE plan for a subsequent run.
  if (env?.api) {
    const license = process.env.BYTEBASE_E2E_LICENSE ?? "";
    for (let attempt = 0; attempt < 3 && !licenseRestored; attempt++) {
      try {
        await env.api.uploadLicense(license);
        const sub = await env.api.getSubscription();
        if (sub.plan && sub.plan !== "FREE") licenseRestored = true;
      } catch {
        /* retry */
      }
    }
    // Restore the original IAM policy (drops the seat-filler + service-account
    // bindings). Re-fetch first so we write with the CURRENT etag: the snapshot
    // was captured in beforeAll and its etag is stale after the intervening IAM
    // writes (the seat-filler bind here and the service-account bind in the
    // first test), and SetIamPolicy rejects a stale non-empty etag as a
    // concurrent update (CodeAborted) — which a bare `.catch` would silently
    // swallow, leaving the workspace polluted.
    if (policySnapshot) {
      const current = await env.api
        .getWorkspaceIamPolicy(workspace)
        .catch(() => undefined);
      if (current) {
        await env.api
          .setWorkspaceIamPolicy(workspace, {
            bindings: policySnapshot.bindings,
            etag: current.etag,
          })
          .catch(() => {});
      }
    }
    // Deactivate the seat-filler users (best-effort cleanup).
    for (const email of createdUserEmails) {
      await env.api.deleteUser(email).catch(() => {});
    }
    if (createdSaEmail) {
      await env.api.deleteServiceAccount(createdSaEmail).catch(() => {});
    }
  }
});

test.describe("Over-limit workspace can still create a service account (BYT-9633)", () => {
  // We assert against the service endpoints the UI calls (CreateServiceAccount +
  // SetIamPolicy) rather than driving the Settings UI: on the FREE plan the
  // frontend Service Accounts page is feature-gated, which is independent of —
  // and would mask — the seat-limit fix under test. The backend operation is the
  // exact one the customer hit ("could not create a service account / add roles").

  test("creating a service account with a workspace role succeeds over the seat limit", async () => {
    test.setTimeout(60_000);

    // Drive the exact path the page's handleCreate uses — create the service
    // account principal, then bind it into workspace IAM as a `serviceAccount:`
    // member (a SEAT-NEUTRAL member). Pre-fix, SetIamPolicy rejected ANY edit
    // when over-limit (newCount >= oldCount), so this binding popped
    // "exceeding the limit"; the fix only rejects edits that strictly increase
    // the (soft-deleted-excluded) seat count, so a service account is allowed.
    const sa = await env.api.createServiceAccount(workspace, SA_PREFIX, "Seat Bot");
    createdSaEmail = sa.email;

    const policy = await env.api.getWorkspaceIamPolicy(workspace);
    const binding = policy.bindings.find(
      (b) => b.role === "roles/workspaceMember",
    );
    const saMember = `serviceAccount:${sa.email}`;
    if (binding) binding.members = [...binding.members, saMember];
    else
      policy.bindings.push({
        role: "roles/workspaceMember",
        members: [saMember],
      });

    // Must NOT throw "exceeding the limit" — a seat-neutral member is allowed.
    await env.api.setWorkspaceIamPolicy(workspace, policy);
  });

  test("adding another end user over the limit is still rejected (vacuity: the limit is active)", async () => {
    test.setTimeout(60_000);

    // Proves the limit is genuinely enforced — so the SA success above reflects
    // the seat-neutral exemption, not a disabled limit. Adding a NEW user:
    // member strictly increases the count and must be rejected.
    const email = `e2e-seat-overflow-${STAMP}@example.com`;
    try {
      await env.api.createUser(email, SEAT_PASSWORD, "Seat Overflow");
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("(409)")) throw err;
    }
    createdUserEmails.push(email);

    const policy = await env.api.getWorkspaceIamPolicy(workspace);
    const binding = policy.bindings.find(
      (b) => b.role === "roles/workspaceMember",
    );
    if (binding) binding.members = [...binding.members, `user:${email}`];
    else
      policy.bindings.push({
        role: "roles/workspaceMember",
        members: [`user:${email}`],
      });

    await expect(
      env.api.setWorkspaceIamPolicy(workspace, policy),
      "adding another end user over the limit must be rejected",
    ).rejects.toThrow(/exceeding the limit/i);
  });
});
