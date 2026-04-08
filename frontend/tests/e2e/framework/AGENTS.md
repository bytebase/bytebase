# E2E Framework — AI Agent Conventions

This file helps AI coding assistants understand the e2e framework structure, conventions, and how to add new feature test suites correctly.

---

## File Structure and Responsibilities

All framework files live in `frontend/tests/e2e/framework/`.

### `api-client.ts`

`BytebaseApiClient` — typed wrapper around the Bytebase v1 REST API.

- Constructor takes `{ baseURL, credentials? }`.
- `private request<T>()` handles all HTTP calls and automatically retries with a fresh token on 401 responses (token refresh pattern — no manual re-login needed in tests).
- Public methods: `login`, `signup`, `healthCheck`, `listInstances`, `listDatabases`, `listProjects`, `getPolicy`, `upsertPolicy`, `deletePolicy`, `getCatalog`, `updateCatalog`, `query`.
- Add new methods here when tests need them — follow the typed-response pattern and let errors bubble as thrown `Error`s.

### `env.ts`

`TestEnv` interface and serialization helpers.

- `TestEnv` holds: `baseURL`, `adminEmail`, `adminPassword?`, `mode` (`"local" | "new"`), `project`, `instance`, `instanceId`, `database`, `databaseId`.
- `detectMode()` — returns `"local"` if `BYTEBASE_URL` is set, `"new"` otherwise.
- `getBaseURL()` — returns `BYTEBASE_URL` or `http://localhost:18234`.
- `saveTestEnv(env)` / `loadTestEnv()` — serialize/deserialize `TestEnv` to/from `frontend/.e2e-env.json`. `loadTestEnv()` also constructs and attaches a `BytebaseApiClient`, so callers get `TestEnv & { api: BytebaseApiClient }`.
- `cleanupEnvFile()` — removes `.e2e-env.json` on teardown.

### `snapshot.ts`

Snapshot/restore for Mode A crash recovery.

- `SnapshotScope` — declares what to capture: `policies` (policy resource paths), `catalogs` (database resource paths), and optional `instanceData` (raw SQL round-trips).
- `createSnapshot(api, scope)` — reads each item in scope from the live server and writes the result to `frontend/.e2e-snapshot.json`. Call this in `beforeAll` when `env.mode === "local"`.
- `restoreSnapshot(api, snapshot)` — writes captured state back to the server via `upsertPolicy` / `updateCatalog`. Call this in `afterAll`.
- `loadPersistedSnapshot()` / `deletePersistedSnapshot()` — used by `global-setup.ts` to detect and handle stale snapshots left by a crashed run.

### `mode-start-new-bytebase.ts`

Mode B: start a disposable Bytebase server.

- `cleanupOrphans()` — kills any leftover process from a previous crashed run by reading `/tmp/bytebase-e2e-pid`.
- `findAvailablePort()` — finds a free port pair starting from `18234` (main + embedded-PG offset).
- `startServer()` — spawns the binary (from `BYTEBASE_BIN` or `./bytebase-build/bytebase`), waits for `/healthz`, calls `signup` to create the admin account, and returns `{ baseURL, adminEmail, adminPassword }`.
- `stopServer()` — sends `SIGTERM` to the process group and deletes the temp data directory.

### `mode-use-local-bytebase.ts`

Mode A: verify reachability and handle crash recovery.

- `verifyReachable(baseURL)` — calls `/healthz` and throws a descriptive error if unreachable, including the command to start Bytebase locally.
- `checkCrashRecovery(api)` — checks for a stale `.e2e-snapshot.json`. In CI it restores automatically; locally it prompts the user before restoring.

### `global-setup.ts`

Playwright `globalSetup`: runs once before any test worker starts.

- Detects mode via `detectMode()`.
- Mode B: calls `cleanupOrphans()` then `startServer()`.
- Mode A: calls `verifyReachable()` then `checkCrashRecovery()`.
- In both modes: logs in to obtain a token, discovers default project/instance/database, and calls `saveTestEnv()`.

### `global-teardown.ts`

Playwright `globalTeardown`: runs once after all test workers finish.

- Mode B: calls `stopServer()` to kill the process and delete temp data.
- Both modes: calls `cleanupEnvFile()` to remove `.e2e-env.json`.

### `setup-project.ts`

Playwright setup project (runs as a named Playwright project, not a worker).

- Authenticates via the browser (navigates to `/landing`, logs in if redirected to `/auth`, suppresses the "New version available" modal).
- Saves browser storage state to `frontend/.auth/state.json` for all subsequent tests.
- Credentials come from `BYTEBASE_USER` (default: `admin@bytebase.com`) and `BYTEBASE_PASS` env vars.

---

## How to Add a New Feature Test Suite

### Step 1: Create the feature directory

```
frontend/tests/e2e/<feature-name>/
```

### Step 2: Create page objects (if needed)

Place them in the feature directory, not in `framework/`. Name them `<feature-name>.page.ts`.

```typescript
// frontend/tests/e2e/my-feature/my-feature.page.ts
import { type Page, type Locator } from "@playwright/test";

export class MyFeaturePage {
  readonly page: Page;
  readonly heading: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: /my feature/i });
  }

  async goto(projectId: string) {
    await this.page.goto(`/projects/${projectId}/my-feature`);
  }
}
```

### Step 3: Create the spec file

```typescript
// frontend/tests/e2e/my-feature/my-feature.spec.ts
import { test, expect } from "@playwright/test";
import { loadTestEnv } from "../framework/env";
import {
  createSnapshot,
  restoreSnapshot,
  deletePersistedSnapshot,
  type SnapshotScope,
  type Snapshot,
} from "../framework/snapshot";
import { MyFeaturePage } from "./my-feature.page";

test.describe("My Feature", () => {
  let env: ReturnType<typeof loadTestEnv>;
  let snapshot: Snapshot | undefined;

  // Declare what state this suite will modify
  const scope: SnapshotScope = {
    policies: ["projects/my-project/policies/some_policy"],
    catalogs: ["instances/my-instance/databases/my-db"],
  };

  test.beforeAll(async () => {
    env = loadTestEnv();

    // Feature-specific discovery: find the resources this suite needs
    // (do NOT hardcode project/instance/database names — discover via API)
    const { projects } = await env.api.listProjects();
    const project = projects.find((p) => p.title === "My Project");
    if (!project) throw new Error("My Project not found");

    // For Mode A: capture state before modifying anything
    if (env.mode === "local") {
      snapshot = await createSnapshot(env.api, scope);
    }

    // API-based setup (faster than browser interactions)
    await env.api.upsertPolicy(project.name, "some_policy", { /* ... */ });
  });

  test.afterAll(async () => {
    if (env?.mode === "local" && snapshot) {
      await restoreSnapshot(env.api, snapshot);
      deletePersistedSnapshot();
    }
  });

  test("verifies the feature in the browser", async ({ page }) => {
    const featurePage = new MyFeaturePage(page);
    await featurePage.goto("my-project-id");
    await expect(featurePage.heading).toBeVisible();
  });
});
```

### Step 4: Register the spec in Playwright config

Add the new spec file to the test match pattern, or rely on the glob if it already covers `tests/e2e/**/*.spec.ts`.

---

## Conventions

### API for setup, browser for verification

- Use `env.api` for all setup and teardown: creating projects, configuring policies, seeding data.
- Use `page` only to verify that the UI reflects the expected state and that interactions work.
- This keeps tests fast and makes failures easier to diagnose — a broken API call is a different problem than a broken UI render.

### Never hardcode resource names

Never write strings like `"projects/production"` or `"instances/prod-db"` directly in test code. Always discover via API:

```typescript
const { instances } = await env.api.listInstances();
const pg = instances.find((i) => i.engine === "POSTGRES");
```

### Always use `loadTestEnv()` for shared state

`loadTestEnv()` returns the base URL, mode, and a ready-to-use `BytebaseApiClient`. Do not construct `BytebaseApiClient` directly in spec files.

### Feature-specific data stays in the feature spec

The `framework/` directory contains only generic infrastructure. Any feature-specific discovery logic (finding the right instance, reading a catalog, determining a column to mask) belongs in the feature's `beforeAll`, not in `env.ts` or `api-client.ts`.

### Page objects belong in feature directories

Do not place page objects in `framework/`. Each feature owns its page objects.

---

## Snapshot Pattern for Mode A

Mode A tests run against a shared, persistent Bytebase server. The snapshot system ensures tests clean up after themselves even if they crash.

### When to use

Use snapshots whenever your test suite modifies server state: policies, catalogs, project membership, etc. Mode B (disposable server) does not need snapshots.

### Pattern

```typescript
// 1. Declare scope — what state will this suite touch?
const scope: SnapshotScope = {
  policies: [
    "projects/my-project/policies/masking_exemption",
  ],
  catalogs: [
    "instances/my-instance/databases/my-db",
  ],
};

let snapshot: Snapshot | undefined;

test.beforeAll(async () => {
  env = loadTestEnv();

  // 2. Capture before modifying anything
  if (env.mode === "local") {
    snapshot = await createSnapshot(env.api, scope);
  }
});

test.afterAll(async () => {
  // 3. Restore after all tests in this suite finish
  if (env?.mode === "local" && snapshot) {
    await restoreSnapshot(env.api, snapshot);
    deletePersistedSnapshot();  // Remove the on-disk file after successful restore
  }
});
```

### Crash recovery

If a test run is interrupted (IDE killed, machine rebooted), `.e2e-snapshot.json` remains on disk with `status: "captured"`. The next run's `global-setup.ts` calls `checkCrashRecovery()`:

- In CI: automatically restores and deletes the snapshot.
- Locally: prompts the user before restoring.

---

## How to Extend the API Client

Add methods to `BytebaseApiClient` in `api-client.ts`. Only add methods that are actually needed by tests.

### Pattern

```typescript
// In BytebaseApiClient class:

async listGroups() {
  return this.request<{ groups: { name: string; title: string }[] }>(
    "GET",
    "/v1/groups?pageSize=100"
  );
}

async createGroup(id: string, title: string) {
  return this.request<{ name: string }>("POST", `/v1/groups`, { name: `groups/${id}`, title });
}
```

Follow the existing patterns:
- Use `this.request<T>()` for all calls — it handles auth and token refresh automatically.
- Use typed generics for response shapes.
- Let errors throw (do not swallow them in API methods). Only use `try/catch` in the caller when you genuinely want to ignore failures (e.g., `deletePolicy` wraps with `try/catch` because the policy may not exist).
- For list endpoints, include `?pageSize=100` to avoid truncated results in test environments.
