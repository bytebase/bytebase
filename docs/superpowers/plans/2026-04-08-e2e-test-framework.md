# E2E Test Framework Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a two-mode Playwright e2e test framework for Bytebase that auto-detects whether to connect to an existing server (Mode A) or start a disposable one (Mode B), with snapshot/restore for safe testing against user data.

**Architecture:** `globalSetup` handles server lifecycle (Mode B start/stop). A Playwright setup project handles auth (supports headed browser login in Mode A). `TestEnv` is serialized to `.e2e-env.json` for cross-worker access. Feature test suites extend the generic `TestEnv` with their own discovery and snapshot scopes.

**Tech Stack:** Playwright Test, TypeScript, Bytebase v1 REST API, Node `child_process` for server management.

**Spec:** `docs/superpowers/specs/2026-04-08-e2e-test-framework-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `frontend/playwright.config.ts` | Rewrite | Config with globalSetup/Teardown + setup project + chromium project |
| `frontend/.gitignore` | Modify | Add e2e artifacts |
| `frontend/tests/e2e/framework/api-client.ts` | Create | Bytebase v1 API wrapper with token refresh |
| `frontend/tests/e2e/framework/env.ts` | Create | TestEnv interface, `loadTestEnv()`, `saveTestEnv()`, mode detection |
| `frontend/tests/e2e/framework/mode-start-new-bytebase.ts` | Create | Mode B: start server, readiness check, orphan cleanup, stop |
| `frontend/tests/e2e/framework/mode-use-local-bytebase.ts` | Create | Mode A: verify reachable, crash recovery check |
| `frontend/tests/e2e/framework/snapshot.ts` | Create | SnapshotScope, capture/restore, persist to disk |
| `frontend/tests/e2e/framework/global-setup.ts` | Create | `globalSetup` function: mode detection, server start |
| `frontend/tests/e2e/framework/global-teardown.ts` | Create | `globalTeardown` function: server stop, cleanup |
| `frontend/tests/e2e/framework/setup-project.ts` | Create | Playwright setup test: auth, discovery, write env |
| `frontend/tests/e2e/framework/README.md` | Create | Human-facing docs |
| `frontend/tests/e2e/framework/AGENTS.md` | Create | AI agent conventions |
| `frontend/tests/e2e/masking-exemption/masking-exemption.spec.ts` | Create (replace existing) | Masking exemption tests using framework |
| `frontend/tests/e2e/helpers/` | Delete | Replaced by `framework/` |
| `frontend/tests/e2e/global-setup.ts` | Delete | Replaced by `framework/global-setup.ts` |
| `frontend/tests/e2e/masking-exemption/masking-exemption.page.ts` | Keep + modify | Page objects updated to use TestEnv |
| `frontend/tests/e2e/masking-exemption/masking-exemption-*.spec.ts` | Delete | Consolidated into single spec |
| `frontend/tests/e2e/masking-exemption/masking-exemption-stale-expiry-timer.spec.ts` | Keep | Bug-locking test stays separate |

---

## Chunk 1: Foundation (api-client, env, .gitignore)

### Task 1: API Client

**Files:**
- Create: `frontend/tests/e2e/framework/api-client.ts`

- [ ] **Step 1: Create API client with auth methods**

```typescript
// frontend/tests/e2e/framework/api-client.ts
export interface ApiClientOptions {
  baseURL: string;
  credentials?: { email: string; password: string };
}

export class BytebaseApiClient {
  private baseURL: string;
  private token = "";
  private credentials?: { email: string; password: string };

  constructor(options: ApiClientOptions) {
    this.baseURL = options.baseURL.replace(/\/$/, "");
    this.credentials = options.credentials;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (this.token) headers["Authorization"] = `Bearer ${this.token}`;

    const resp = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    // Token refresh on 401
    if (resp.status === 401 && this.credentials) {
      await this.login(this.credentials.email, this.credentials.password);
      headers["Authorization"] = `Bearer ${this.token}`;
      const retry = await fetch(`${this.baseURL}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!retry.ok) throw new Error(`API ${method} ${path} failed (${retry.status}): ${await retry.text()}`);
      return retry.json() as Promise<T>;
    }

    if (!resp.ok) throw new Error(`API ${method} ${path} failed (${resp.status}): ${await resp.text()}`);
    return resp.json() as Promise<T>;
  }

  // Auth
  async login(email: string, password: string): Promise<string> {
    const { token } = await this.request<{ token: string }>("POST", "/v1/auth/login", { email, password });
    this.token = token;
    this.credentials = { email, password };
    return token;
  }

  async signup(email: string, password: string, title: string): Promise<string> {
    const resp = await this.request<{ token?: string }>("POST", "/v1/auth/signup", { email, password, title });
    if (resp.token) this.token = resp.token;
    return this.token;
  }

  // Health
  async healthCheck(): Promise<boolean> {
    try {
      const resp = await fetch(`${this.baseURL}/healthz`);
      return resp.ok;
    } catch {
      return false;
    }
  }

  // Discovery
  async listInstances() {
    return this.request<{ instances: { name: string; engine: string; title: string }[] }>("GET", "/v1/instances?pageSize=100&showDeleted=false");
  }

  async listDatabases(parent: string) {
    return this.request<{ databases: { name: string; project: string }[] }>("GET", `/v1/${parent}/databases?pageSize=100`);
  }

  async listProjects() {
    return this.request<{ projects: { name: string; title: string }[] }>("GET", "/v1/projects?pageSize=100&showDeleted=false");
  }

  // Policies
  async getPolicy(policyName: string) {
    try {
      return await this.request<Record<string, unknown>>("GET", `/v1/${policyName}`);
    } catch {
      return null;
    }
  }

  async upsertPolicy(parent: string, policyType: string, policy: unknown) {
    return this.request<unknown>("PATCH", `/v1/${parent}/policies/${policyType}?allowMissing=true`, policy);
  }

  async deletePolicy(parent: string, policyType: string) {
    try { await this.request<unknown>("DELETE", `/v1/${parent}/policies/${policyType}`); } catch { /* ignore */ }
  }

  // Catalog
  async getCatalog(dbName: string) {
    return this.request<Record<string, unknown>>("GET", `/v1/${dbName}/catalog`);
  }

  async updateCatalog(dbName: string, catalog: unknown) {
    return this.request<unknown>("PATCH", `/v1/${dbName}/catalog`, catalog);
  }

  // Query
  async query(instanceName: string, databaseName: string, statement: string) {
    return this.request<{ results: unknown[] }>("POST", `/v1/${instanceName}:query`, {
      name: instanceName,
      connectionDatabase: databaseName,
      statement,
      limit: 10,
    });
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/api-client.ts
git commit -m "feat(e2e): add Bytebase API client with token refresh"
```

### Task 2: TestEnv and mode detection

**Files:**
- Create: `frontend/tests/e2e/framework/env.ts`

- [ ] **Step 1: Create env module**

```typescript
// frontend/tests/e2e/framework/env.ts
import * as fs from "fs";
import * as path from "path";
import { BytebaseApiClient } from "./api-client";

const ENV_FILE = path.join(__dirname, "../../.e2e-env.json");

export interface TestEnv {
  baseURL: string;
  adminEmail: string;
  adminPassword?: string; // Only present if credentials were provided/generated
  mode: "local" | "new";
  project: string;
  instance: string;
  instanceId: string;
  database: string;
  databaseId: string;
}

// Serializable subset (no api client)
type SerializedTestEnv = Omit<TestEnv, "api">;

export function detectMode(): "local" | "new" {
  return process.env.BYTEBASE_URL ? "local" : "new";
}

export function getBaseURL(): string {
  return process.env.BYTEBASE_URL ?? `http://localhost:${getPort()}`;
}

export function getPort(): number {
  // Mode B default port. Actual port is written to env file after scanning.
  return 18234;
}

export function saveTestEnv(env: TestEnv): void {
  const serialized: SerializedTestEnv = { ...env };
  delete (serialized as Record<string, unknown>)["api"];
  fs.writeFileSync(ENV_FILE, JSON.stringify(serialized, null, 2));
}

export function loadTestEnv(): TestEnv & { api: BytebaseApiClient } {
  if (!fs.existsSync(ENV_FILE)) {
    throw new Error(
      ".e2e-env.json not found. Run the setup project first (npx playwright test)."
    );
  }
  const raw: SerializedTestEnv = JSON.parse(fs.readFileSync(ENV_FILE, "utf-8"));
  const api = new BytebaseApiClient({
    baseURL: raw.baseURL,
    credentials: raw.adminPassword
      ? { email: raw.adminEmail, password: raw.adminPassword }
      : undefined,
  });
  // Login immediately so the token is ready
  if (raw.adminPassword) {
    // Fire and forget — tests will await api calls which will trigger login if needed
  }
  return { ...raw, api };
}

export function cleanupEnvFile(): void {
  if (fs.existsSync(ENV_FILE)) fs.unlinkSync(ENV_FILE);
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/env.ts
git commit -m "feat(e2e): add TestEnv interface and mode detection"
```

### Task 3: Update .gitignore

**Files:**
- Modify: `frontend/.gitignore`

- [ ] **Step 1: Append e2e artifacts to .gitignore**

Add to the end of `frontend/.gitignore`:
```
# E2E test artifacts
.e2e-env.json
.e2e-snapshot.json
.auth/
test-results/
playwright-report/
.playwright-cli/
.playwright-mcp/
```

- [ ] **Step 2: Commit**

```bash
git add frontend/.gitignore
git commit -m "chore: add e2e test artifacts to .gitignore"
```

---

## Chunk 2: Mode B — Start New Bytebase

### Task 4: Mode B server management

**Files:**
- Create: `frontend/tests/e2e/framework/mode-start-new-bytebase.ts`

- [ ] **Step 1: Implement server start, readiness check, orphan cleanup, stop**

Key behaviors:
- `cleanupOrphans()`: Read `/tmp/bytebase-e2e-pid`, kill stale process if exists, delete its temp dir
- `findAvailablePort()`: Start from 18234, check both `PORT` and `PORT+2` via `net.createServer().listen()`, increment by 4
- `startServer()`: Spawn binary with `--port`, `--data`, `--demo`, `PG_URL=""` in env. Write PID file. Two-phase readiness: poll `/healthz` then retry signup until success. Timeout configurable via `BYTEBASE_STARTUP_TIMEOUT`.
- `stopServer()`: Kill process group (`-pid` signal), delete temp dir and PID file

See spec section "Mode: Start New Bytebase" for exact logic. Use `child_process.spawn` with `detached: true` and `process.kill(-pid, 'SIGTERM')` for process group cleanup.

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/mode-start-new-bytebase.ts
git commit -m "feat(e2e): add Mode B — start disposable Bytebase with --demo"
```

### Task 5: Mode A — connect to local server

**Files:**
- Create: `frontend/tests/e2e/framework/mode-use-local-bytebase.ts`

- [ ] **Step 1: Implement verify reachable + crash recovery check**

Key behaviors:
- `verifyReachable(baseURL)`: Fetch `/healthz`, throw with helpful message if unreachable
- `checkCrashRecovery(api)`: Read `.e2e-snapshot.json`. If `status === "captured"`, prompt user (or auto-restore in CI via `process.env.CI`). Delegate actual restore to `snapshot.ts`.

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/mode-use-local-bytebase.ts
git commit -m "feat(e2e): add Mode A — connect to local Bytebase with crash recovery"
```

---

## Chunk 3: Snapshot, global-setup/teardown, setup project

### Task 6: Snapshot/restore system

**Files:**
- Create: `frontend/tests/e2e/framework/snapshot.ts`

- [ ] **Step 1: Implement SnapshotScope, capture, restore, persist to disk**

```typescript
export interface SnapshotScope {
  policies?: string[];
  catalogs?: string[];
  instanceData?: {
    instance: string;
    database: string;
    captureQueries: string[];
    restoreQueries: string[];
  }[];
}

export interface Snapshot {
  status: "captured" | "restored";
  timestamp: string;
  metadata: {
    policies: Record<string, unknown>;   // policyPath -> policy JSON
    catalogs: Record<string, unknown>;   // catalogPath -> catalog JSON
  };
  instanceData: Record<string, unknown[]>; // key -> query results
}
```

- `createSnapshot(api, scope)`: Fetch each policy/catalog via API, run captureQueries via `api.query()`, persist to `.e2e-snapshot.json` with `status: "captured"`.
- `restoreSnapshot(api, snapshot)`: Upsert each policy/catalog, run restoreQueries, update file to `status: "restored"`.
- `loadPersistedSnapshot()` / `deletePersistedSnapshot()`: Read/delete `.e2e-snapshot.json`.

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/snapshot.ts
git commit -m "feat(e2e): add snapshot/restore with crash recovery persistence"
```

### Task 7: globalSetup and globalTeardown

**Files:**
- Create: `frontend/tests/e2e/framework/global-setup.ts`
- Create: `frontend/tests/e2e/framework/global-teardown.ts`

- [ ] **Step 1: Implement globalSetup**

```typescript
// global-setup.ts
import { detectMode, saveTestEnv, getBaseURL } from "./env";
import { cleanupOrphans, startServer } from "./mode-start-new-bytebase";
import { verifyReachable, checkCrashRecovery } from "./mode-use-local-bytebase";

async function globalSetup() {
  const mode = detectMode();

  if (mode === "new") {
    cleanupOrphans();
    const { baseURL, adminEmail, adminPassword } = await startServer();
    saveTestEnv({
      baseURL, adminEmail, adminPassword, mode,
      // Placeholders — setup project fills in discovered data
      project: "", instance: "", instanceId: "", database: "", databaseId: "",
    });
  } else {
    const baseURL = getBaseURL();
    await verifyReachable(baseURL);
    saveTestEnv({
      baseURL, mode, adminEmail: process.env.BYTEBASE_USER ?? "",
      adminPassword: process.env.BYTEBASE_PASS,
      project: "", instance: "", instanceId: "", database: "", databaseId: "",
    });
  }
}

export default globalSetup;
```

- [ ] **Step 2: Implement globalTeardown**

```typescript
// global-teardown.ts
import { detectMode, cleanupEnvFile } from "./env";
import { stopServer } from "./mode-start-new-bytebase";

async function globalTeardown() {
  const mode = detectMode();
  if (mode === "new") {
    stopServer();
  }
  cleanupEnvFile();
}

export default globalTeardown;
```

- [ ] **Step 3: Commit**

```bash
git add frontend/tests/e2e/framework/global-setup.ts frontend/tests/e2e/framework/global-teardown.ts
git commit -m "feat(e2e): add globalSetup/globalTeardown for server lifecycle"
```

### Task 8: Setup project (auth + discovery)

**Files:**
- Create: `frontend/tests/e2e/framework/setup-project.ts`

- [ ] **Step 1: Implement setup test**

This is a Playwright test file that runs as a "setup" project. It has browser access for headed login in Mode A.

Key logic:
1. Read partial `TestEnv` from `.e2e-env.json`
2. **Auth**: 
   - Mode B: `api.login(adminEmail, adminPassword)` (credentials from globalSetup)
   - Mode A + user + pass: `api.login(user, pass)` 
   - Mode A + user only: headed browser, pre-fill email, user types password
   - Mode A only: headed browser, user fills everything
3. **Suppress "New version" modal**: Set `bb.release` localStorage entry
4. **Discovery**: `listInstances()` → find first Postgres → `listDatabases()` → find first non-system DB → `listProjects()` → find owning project
5. **Mode A crash recovery**: call `checkCrashRecovery(api)`
6. Update `.e2e-env.json` with discovered data
7. Save browser auth state to `.auth/state.json`

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/setup-project.ts
git commit -m "feat(e2e): add setup project — auth, discovery, env persistence"
```

### Task 9: Playwright config

**Files:**
- Rewrite: `frontend/playwright.config.ts`

- [ ] **Step 1: Rewrite config**

```typescript
import { defineConfig, devices } from "@playwright/test";
import * as path from "path";

export default defineConfig({
  testDir: "./tests/e2e",
  testIgnore: ["**/framework/**"],
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "html",
  globalSetup: path.resolve(__dirname, "tests/e2e/framework/global-setup.ts"),
  globalTeardown: path.resolve(__dirname, "tests/e2e/framework/global-teardown.ts"),
  use: {
    storageState: ".auth/state.json",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "setup",
      testMatch: /framework\/setup-project\.ts/,
      use: { storageState: undefined },
    },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      dependencies: ["setup"],
    },
  ],
});
```

Note: `baseURL` is not set here because it's dynamic (Mode B picks a port at runtime). Tests read it from `loadTestEnv().baseURL`.

- [ ] **Step 2: Commit**

```bash
git add frontend/playwright.config.ts
git commit -m "feat(e2e): rewrite playwright config with globalSetup/Teardown + setup project"
```

---

## Chunk 4: Masking exemption tests (first feature suite)

### Task 10: Update page objects

**Files:**
- Modify: `frontend/tests/e2e/masking-exemption/masking-exemption.page.ts`

- [ ] **Step 1: Update page objects to accept baseURL from TestEnv**

Remove hardcoded URLs. All `goto` methods accept `baseURL` as parameter or read from TestEnv. The `SqlEditorPage.gotoWithDb()` method constructs the URL dynamically. No other structural changes — the page objects from the earlier session are well-designed.

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/masking-exemption/masking-exemption.page.ts
git commit -m "refactor(e2e): update page objects to use dynamic baseURL"
```

### Task 11: Rewrite masking exemption tests

**Files:**
- Create: `frontend/tests/e2e/masking-exemption/masking-exemption.spec.ts` (replaces 3 old spec files)
- Delete: `frontend/tests/e2e/masking-exemption/masking-exemption-e2e-masking.spec.ts`
- Delete: `frontend/tests/e2e/masking-exemption/masking-exemption-grant-revoke.spec.ts`
- Delete: `frontend/tests/e2e/masking-exemption/masking-exemption-list.spec.ts`
- Keep: `frontend/tests/e2e/masking-exemption/masking-exemption-stale-expiry-timer.spec.ts`

- [ ] **Step 1: Write consolidated masking-exemption.spec.ts**

Structure:
```typescript
import { test, expect } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { createSnapshot, restoreSnapshot, type Snapshot } from "../framework/snapshot";

interface MaskingTestData {
  sampleTable: string;
  sampleSchema: string;
  sampleColumn: string;
  knownUnmaskedValue: string;
}

let env: TestEnv & { api: BytebaseApiClient };
let maskingData: MaskingTestData;
let snapshot: Snapshot | undefined;

test.beforeAll(async () => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword!);
  maskingData = await discoverMaskingData(env);
  // Configure masking on the discovered column
  await configureMasking(env, maskingData);
  // Snapshot if Mode A
  if (env.mode === "local") {
    snapshot = await createSnapshot(env.api, {
      policies: [`${env.project}/policies/masking_exemption`],
      catalogs: [env.database],
    });
  }
});

test.afterAll(async () => {
  if (snapshot) await restoreSnapshot(env.api, snapshot);
});

test.describe("Exemption List Page", () => {
  // Tests from masking-exemption-list.spec.ts — adapted to use env
});

test.describe("Grant and Revoke", () => {
  // Tests from masking-exemption-grant-revoke.spec.ts — adapted
});

test.describe("E2E Masking Verification", () => {
  // Tests from masking-exemption-e2e-masking.spec.ts — adapted
  // Uses maskingData.knownUnmaskedValue for algorithm-agnostic checks
});
```

Helper functions `discoverMaskingData()` and `configureMasking()` go in the same file (feature-specific, not framework).

- [ ] **Step 2: Delete old spec files**

```bash
rm frontend/tests/e2e/masking-exemption/masking-exemption-e2e-masking.spec.ts
rm frontend/tests/e2e/masking-exemption/masking-exemption-grant-revoke.spec.ts
rm frontend/tests/e2e/masking-exemption/masking-exemption-list.spec.ts
```

- [ ] **Step 3: Commit**

```bash
git add frontend/tests/e2e/masking-exemption/
git commit -m "feat(e2e): rewrite masking exemption tests using framework"
```

### Task 12: Delete old helpers

**Files:**
- Delete: `frontend/tests/e2e/helpers/api-client.ts`
- Delete: `frontend/tests/e2e/helpers/test-fixtures.ts`
- Delete: `frontend/tests/e2e/global-setup.ts`

- [ ] **Step 1: Remove old files**

```bash
rm -rf frontend/tests/e2e/helpers
rm frontend/tests/e2e/global-setup.ts
```

- [ ] **Step 2: Commit**

```bash
git add -A frontend/tests/e2e/helpers frontend/tests/e2e/global-setup.ts
git commit -m "chore(e2e): remove old test helpers replaced by framework"
```

---

## Chunk 5: Documentation + verification

### Task 13: README.md

**Files:**
- Create: `frontend/tests/e2e/framework/README.md`

- [ ] **Step 1: Write README**

Cover:
- Quick start (3 commands: build, run Mode B, run Mode A)
- Env var reference table
- How to write a new feature test suite (with template)
- How snapshot/restore works
- Troubleshooting (port in use, binary not found, stale PID)

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/README.md
git commit -m "docs(e2e): add framework README"
```

### Task 14: AGENTS.md

**Files:**
- Create: `frontend/tests/e2e/framework/AGENTS.md`

- [ ] **Step 1: Write AGENTS.md**

Cover:
- File structure and responsibilities
- How to add a new feature test suite (step by step)
- Conventions: API for setup, browser for verification
- Snapshot pattern for Mode A
- Do NOT hardcode project/instance/database names
- How to extend the API client

- [ ] **Step 2: Commit**

```bash
git add frontend/tests/e2e/framework/AGENTS.md
git commit -m "docs(e2e): add framework AGENTS.md for AI agents"
```

### Task 15: Run tests in both modes

- [ ] **Step 1: Run Mode A (against local server)**

```bash
BYTEBASE_URL=http://localhost:3000 BYTEBASE_USER=admin@bytebase.com BYTEBASE_PASS=bytebase npx playwright test --reporter=list
```

Expected: All tests pass. Snapshot restores cleanly.

- [ ] **Step 2: Run Mode B (disposable server)**

```bash
# Build binary first if not already built
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

npx playwright test --reporter=list
```

Expected: Server starts, tests pass, server stops, temp dir deleted.

- [ ] **Step 3: Verify crash recovery (Mode A)**

1. Run tests with `BYTEBASE_URL` set
2. Kill the test runner mid-run (`Ctrl+C` or `kill`)
3. Check `.e2e-snapshot.json` exists with `status: "captured"`
4. Re-run tests — should prompt to restore, then proceed normally

- [ ] **Step 4: Verify orphan cleanup (Mode B)**

1. Run tests without `BYTEBASE_URL`
2. Kill test runner mid-run
3. Check `/tmp/bytebase-e2e-pid` exists
4. Re-run tests — should clean up orphaned process before starting fresh

- [ ] **Step 5: Commit final state**

```bash
git add -A
git commit -m "feat(e2e): complete e2e test framework with both modes verified"
```

---

## Chunk 6: Bug fixes discovered during PR #19743 review

These bugs were found during exploratory QA of the masking exemption redesign (PR #19743). Fix them and add regression tests.

### Task 16: Fix stale expiry timer in ExemptionGrantSection

**Bug:** `isExpired` and `expiryLabel` in `ExemptionGrantSection` use `Date.now()` inside `useMemo` but don't include it in the dependency array. A grant showing "expires in 2 days" will display that label forever until page reload, even after it actually expires. The `isExpired` flag also won't flip from `false` to `true` when the expiration time is reached.

**Location:** `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx` — the `ExemptionGrantSection` component, approximately lines 1098-1119.

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx`

- [ ] **Step 1: Investigate the current code**

Read `ExemptionGrantSection` in `ProjectMaskingExemptionPage.tsx`. Find the two `useMemo` calls that use `Date.now()`:
```typescript
const isExpired = useMemo(
  () => !!grant.expirationTimestamp && grant.expirationTimestamp <= Date.now(),
  [grant.expirationTimestamp]  // <-- Date.now() not in deps
);
const expiryLabel = useMemo(() => {
  // ...
  const msRemaining = grant.expirationTimestamp - Date.now();  // <-- stale
  // ...
}, [grant.expirationTimestamp, t]);  // <-- Date.now() not in deps
```

- [ ] **Step 2: Fix — remove useMemo wrappers**

These are cheap computations (subtraction + comparison). Remove the `useMemo` wrappers so they recompute on every render. This is the simplest fix — no interval timers needed since the values will update whenever any state change triggers a re-render.

```typescript
const isExpired = !!grant.expirationTimestamp && grant.expirationTimestamp <= Date.now();

const expiryLabel = (() => {
  if (!grant.expirationTimestamp) return "";
  const msRemaining = grant.expirationTimestamp - Date.now();
  // ... rest of computation
})();
```

- [ ] **Step 3: Run type check and lint**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend check
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx
git commit -m "fix(masking-exemption): remove stale useMemo on expiry timer

isExpired and expiryLabel used Date.now() inside useMemo without
including it in the dependency array, causing the expiry display
to freeze until page reload. These are cheap computations —
remove useMemo so they recompute on every render."
```

### Task 17: Fix "-1" display for instances/databases in exemption resource table

**Bug:** In the masking exemption detail panel, some grants show instance and database names as "-1" in the resource table. This appears when a grant has `databaseResources` with `databaseFullName` set to a path that doesn't match the expected `instances/{instance}/databases/{database}` pattern.

**Files:**
- Investigate: `frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx` (ExemptionResourceTable)
- Investigate: `frontend/src/components/SensitiveData/exemptionDataUtils.ts` (extractDatabaseName)
- Investigate: Backend policy storage to understand how "-1" gets into `databaseFullName`

- [ ] **Step 1: Trace the "-1" value**

1. In `ProjectMaskingExemptionPage.tsx`, find `ExemptionResourceTable` and look at how it calls `extractDatabaseResourceName(resource.databaseFullName)`.
2. In the API, query the masking exemption policy for the project and find the exemption that produces "-1": `curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:3000/v1/projects/project-sample/policies/masking_exemption"` — search for "-1" in the response.
3. Check if "-1" is a sentinel value in the database (meaning "all instances" or "all databases") or if it's a data corruption issue from a previous version.
4. Check `extractDatabaseResourceName` from `@/utils/v1/database` — what does it return when the input doesn't match the `instances/{instance}/databases/{database}` pattern?

- [ ] **Step 2: Fix the display**

Based on findings:
- **If "-1" is a sentinel for "all"**: Show "All" instead of "-1" in the resource table (similar to how empty string is already handled with `isSentinel`).
- **If "-1" is invalid data**: Consider filtering it out or showing a more descriptive placeholder.
- **If it can be prevented at write time**: Fix the code that creates the exemption to avoid writing "-1".

- [ ] **Step 3: Run type check and lint**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend check
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/ProjectMaskingExemptionPage.tsx
git commit -m "fix(masking-exemption): fix -1 display in exemption resource table"
```
