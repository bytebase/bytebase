# E2E Test Framework for Bytebase

## Problem

Bytebase has no frontend e2e test infrastructure. Ad hoc browser tests written during feature reviews depend on local state — specific projects, databases, user accounts, and table data — making them non-portable and non-reproducible.

## Goals

- Generic framework usable by any Bytebase feature (masking, SQL review, access grants, etc.)
- Zero-configuration for common cases — runs with a single command
- Works when invoked directly via `npx playwright test`, not just through Claude Code
- No assumptions about the user's local environment

## Non-goals

- Visual regression testing (screenshot diffing)
- Performance benchmarking
- Mobile/touch testing (Bytebase is desktop-focused)

## Two Modes

The framework auto-detects which mode to use based on environment variables.

### Mode: Use Local Bytebase

Activated when `BYTEBASE_URL` is set. Connects to an existing Bytebase server the user is already running.

**Env vars:**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BYTEBASE_URL` | Yes (triggers this mode) | — | Base URL of running Bytebase |
| `BYTEBASE_USER` | No | — | Admin email for API login |
| `BYTEBASE_PASS` | No | — | Admin password for API login |

**Authentication flow:**
- `BYTEBASE_URL` + `BYTEBASE_USER` + `BYTEBASE_PASS` all set: fully automated API login
- `BYTEBASE_URL` + `BYTEBASE_USER` set (no password): open headed browser with email pre-filled, user types password manually, save auth state, switch to headless
- `BYTEBASE_URL` only: open headed browser, user fills both email and password manually

The headed browser authentication cases use a Playwright **setup project** (not `globalSetup`), because `globalSetup` functions do not have access to browser contexts. The setup project runs as a test that can launch a headed browser, save state, then subsequent test projects run headless using the saved state.

**Data strategy:** Tests discover existing data (instances, projects, databases) via API. This mode does NOT require `--demo` data — it works with any Bytebase server that has at least one Postgres instance with a database.

**Two data layers to track:** Tests can modify two distinct layers of data, and both must be snapshotted and restored:

1. **Metadata DB** — Bytebase's own state: policies (masking exemption, masking rules), database catalog config (semantic types, classification), project settings, access grants. Modified via Bytebase API.
2. **Connected instance data** — Actual data in the user's database instances: table rows inserted/updated/deleted by tests (e.g., via SQL Editor or schema changes through issues). Modified via SQL queries or Bytebase's change workflow.

The snapshot/restore system must handle both layers.

**Snapshot/restore:** Each feature test suite declares what it will modify via a `SnapshotScope`:

```typescript
interface SnapshotScope {
  // Metadata DB resources
  policies?: string[];     // e.g., "projects/hr/policies/masking_exemption"
  catalogs?: string[];     // e.g., "instances/prod/databases/hr_prod"

  // Connected instance data — SQL to capture and restore
  instanceData?: {
    instance: string;      // e.g., "instances/prod-sample-instance"
    database: string;      // e.g., "hr_prod"
    // Queries to capture state before tests (results stored as snapshot)
    captureQueries: string[];
    // Queries to restore state after tests (parameterized with captured data)
    restoreQueries: string[];
  }[];
}
```

Feature test suites register their scope in `beforeAll`. The framework captures the current state via API (for metadata) and SQL queries (for instance data), then restores in `afterAll`. Best-effort restoration — errors are logged as warnings, not failures, to avoid masking real test failures.

**Crash recovery:** Snapshots are persisted to disk at `.e2e-snapshot.json` immediately after capture. On the next test run, `global-setup` checks for this file:

1. **File exists with `status: "captured"`** — Previous run crashed before restore. The framework prompts (in CI, auto-restores): `"Found unrestored snapshot from previous test run (<timestamp>). Restore now? [Y/n]"`. If the user confirms (or CI auto-confirms), restore from the persisted snapshot, then delete the file. If the user declines, delete the file without restoring (user has manually fixed the data).
2. **File exists with `status: "restored"` or file doesn't exist** — Clean state, proceed normally.

The snapshot file records a timestamp so users can judge whether their manual changes came before or after the crash.

```typescript
// .e2e-snapshot.json
{
  "status": "captured",          // "captured" | "restored"
  "timestamp": "2026-04-08T...", // when snapshot was taken
  "metadata": { ... },           // serialized policy/catalog state
  "instanceData": { ... }        // captured query results
}
```

**Test isolation:** `beforeEach` in each test file resets relevant state via API.

### Mode: Start New Bytebase

Activated when `BYTEBASE_URL` is **not** set. Starts a disposable Bytebase instance.

**Env vars:**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BYTEBASE_BIN` | No | `./bytebase-build/bytebase` | Path to pre-built Bytebase binary |
| `BYTEBASE_STARTUP_TIMEOUT` | No | `60000` | Server startup timeout in ms |

If `BYTEBASE_USER`, `BYTEBASE_PASS`, or `PG_URL` are in the environment, they are **ignored** in this mode to prevent accidental interaction with user data.

**Server lifecycle:**
1. **Orphan cleanup:** Check for stale PID file at `/tmp/bytebase-e2e-pid`. If found, kill the orphaned process and delete its temp directory. This handles previous runs that crashed without cleanup.
2. Verify `BYTEBASE_BIN` exists. If missing, error with: `"Bytebase binary not found at <path>. Build it with: go build -ldflags '-w -s' -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go"`
3. Create temp data directory: `/tmp/bytebase-e2e-<random>`
4. Pick a high port. Scan for availability on **both** `PORT` and `PORT+2` (the backend allocates embedded Postgres on `PORT+2` — see `profile.go:16`). Start from 18234 and increment by 4 until a free pair is found.
5. Start server: `$BYTEBASE_BIN --port $PORT --data $TMPDIR --demo`
   - `PG_URL` is explicitly unset in the child process environment (`env: { ...process.env, PG_URL: "" }`) to force embedded Postgres and prevent accidental metadata DB overwrites
6. Write PID and temp directory path to `/tmp/bytebase-e2e-pid` for orphan cleanup.
7. **Two-phase readiness check:**
   - Phase 1: Poll `GET /healthz` until it returns 200 (basic TCP readiness)
   - Phase 2: Retry `POST /v1/auth/signup` until it succeeds (confirms API layer, migrations, and demo data are fully loaded). The `/healthz` endpoint is unreliable as a readiness indicator — the backend explicitly notes it can return 200 before the server is actually ready (see `root.go:196-198`).
   - Timeout: `BYTEBASE_STARTUP_TIMEOUT` (default 60s). Embedded Postgres startup + migrations + demo data loading can be slow on CI runners.
8. Save auth state.

**Data strategy:** `--demo` loads sample instances, projects, databases, and table data. Tests mutate freely — the entire temp directory is disposable. No `beforeEach` cleanup needed — test ordering within a file is deterministic with `workers: 1`.

**Cleanup:** `global-teardown` kills the server process (using process group to ensure embedded Postgres is also killed), deletes the temp data directory, and removes the PID file.

**Error messages for common failures:**
- Binary not found: build instructions (above)
- Port in use: `"Port <PORT> is in use. Check for orphaned Bytebase processes: lsof -ti:<PORT> | xargs kill"`
- Server not reachable: `"Bytebase server at <URL> is not responding. Is it running?"`
- Signup fails: `"Failed to create admin account. Server may not be fully initialized yet."`

## TestEnv Interface

Core `TestEnv` is generic — no feature-specific fields. Feature test suites extend it with their own discovery.

```typescript
// Core — provided by framework
interface TestEnv {
  baseURL: string;               // e.g., "http://localhost:18234"
  api: BytebaseApiClient;        // Authenticated API client
  adminEmail: string;            // e.g., "e2e-admin@bytebase.com"
  mode: "local" | "new";        // Which mode is active

  // Discovered demo data references
  project: string;               // e.g., "projects/hr"
  instance: string;              // e.g., "instances/prod-sample-instance"
  instanceId: string;            // e.g., "prod-sample-instance"
  database: string;              // e.g., "instances/prod-sample-instance/databases/hr_prod"
  databaseId: string;            // e.g., "hr_prod"
}
```

Feature-specific extensions are discovered in each test suite's `beforeAll`, not in global-setup:

```typescript
// In masking-exemption/masking-exemption.spec.ts
interface MaskingTestData {
  sampleTable: string;
  sampleSchema: string;
  sampleColumn: string;
  knownUnmaskedValue: string;
}

async function discoverMaskingData(env: TestEnv): Promise<MaskingTestData> {
  const catalog = await env.api.getCatalog(env.database);
  // find a text column, query for known value...
}
```

This keeps the framework layer truly generic.

`TestEnv` is serialized to `.e2e-env.json` by global-setup and read by test files via `loadTestEnv()`. This bridges separate Playwright worker processes. (With the current `workers: 1` setting this is technically unnecessary, but it future-proofs for parallel execution.)

### Discovery

During global-setup, after authentication:
1. List instances → find the first active Postgres instance
2. List databases in that instance → find a database with tables
3. Find the project that owns the database

Feature-specific discovery (columns, sample values) happens in each test suite's `beforeAll`.

## Lifecycle

The framework uses a **Playwright setup project** for authentication (supports headed browser login in Mode A) and `globalSetup`/`globalTeardown` functions for server lifecycle (Mode B).

```
globalSetup (runs before any project)
  |-- detect mode (BYTEBASE_URL set?)
  |-- [Mode B] orphan cleanup -> find binary -> start server -> readiness check -> signup admin
  |-- [Mode A] verify server reachable
  |-- write server info to .e2e-env.json (baseURL, mode)
  v
setup project (runs as a Playwright test, has browser access)
  |-- [Mode A] login via API or headed browser
  |-- [Mode B] login with generated credentials via API
  |-- discover demo data (project, instance, database)
  |-- update .e2e-env.json with full TestEnv
  |-- save auth state to .auth/state.json
  v
test projects run (headless, use saved auth state)
  |-- each test file calls loadTestEnv() to read .e2e-env.json
  |-- feature suites: beforeAll discovers feature-specific data + snapshots (Mode A)
  |-- feature suites: afterAll restores snapshots (Mode A)
  v
globalTeardown
  |-- [Mode A] no-op (suites handle their own restore)
  |-- [Mode B] kill server (process group), rm -rf temp dir, remove PID file
```

## API Client

Minimal surface — covers test setup needs. Grows as feature tests are added.

| Category | Methods | Used by |
|----------|---------|---------|
| Auth | `signup`, `login` | global-setup |
| Discovery | `listInstances`, `listDatabases`, `listProjects` | setup project |
| Policies | `getPolicy`, `upsertPolicy`, `deletePolicy` | masking tests, snapshot/restore |
| Catalog | `getCatalog`, `updateCatalog` | masking tests (configure masking) |
| Query | `query` | masking tests (verify masked/unmasked) |
| Health | `healthCheck` | Mode B (readiness check) |

The client stores a single auth token from login. If a test suite runs long enough for the token to expire, the client catches 401 responses and re-authenticates automatically.

Not included (add when needed): project/instance/database creation, user management, issue management, SQL review configuration.

## File Structure

```
frontend/
  playwright.config.ts
  .gitignore                            # Must include: .e2e-env.json, .auth/, test-results/, playwright-report/
  tests/e2e/
    framework/
      README.md                         # How to run, configure, write tests
      AGENTS.md                         # AI agent conventions and patterns
      env.ts                            # TestEnv interface, loadTestEnv(), mode detection
      snapshot.ts                       # SnapshotScope interface, capture/restore, crash recovery
      mode-use-local-bytebase.ts        # Mode A: connect, verify reachable
      mode-start-new-bytebase.ts        # Mode B: start, readiness check, cleanup, orphan handling
      api-client.ts                     # Bytebase v1 REST API wrapper (with token refresh)
      global-setup.ts                   # globalSetup: mode detection, server start (Mode B)
      global-teardown.ts                # globalTeardown: server cleanup (Mode B)
      setup-project.ts                  # Playwright setup project: auth, discovery, env write
    masking-exemption/                  # Feature: masking exemption tests
      masking-exemption.spec.ts
    sql-editor/                         # Feature: SQL editor tests (future)
    sql-review/                         # Feature: SQL review tests (future)
```

- `framework/` is generic — no feature-specific code
- Feature tests in own directories, import `TestEnv` from framework
- Page objects (if needed) belong in feature directories, not framework

## Test Patterns

### State setup via API, verification via browser

Tests use the API client for fast, reliable state setup and teardown. The browser is used only to verify UI behavior.

```typescript
import { loadTestEnv } from "../framework/env";

test.describe("Feature X", () => {
  const env = loadTestEnv();

  test("some behavior", async ({ page }) => {
    // Setup via API (fast)
    await env.api.upsertPolicy(env.project, "masking_exemption", { ... });

    // Verify via browser (what we're actually testing)
    await page.goto(`${env.baseURL}/projects/...`);
    await expect(page.getByText("expected")).toBeVisible();
  });
});
```

### Feature-specific discovery and snapshot

```typescript
import { loadTestEnv } from "../framework/env";
import { createSnapshot, restoreSnapshot, type Snapshot } from "../framework/snapshot";

let maskingData: MaskingTestData;
let snapshot: Snapshot;

test.beforeAll(async () => {
  const env = loadTestEnv();
  maskingData = await discoverMaskingData(env);

  // Mode A: snapshot what we'll modify
  if (env.mode === "local") {
    snapshot = await createSnapshot(env.api, {
      policies: [`${env.project}/policies/masking_exemption`],
      catalogs: [env.database],
    });
  }
});

test.afterAll(async () => {
  if (snapshot) {
    await restoreSnapshot(loadTestEnv().api, snapshot);
  }
});
```

### Masking verification

Compare query results against known unmasked values. Algorithm-agnostic — doesn't assume any specific masked output format.

```typescript
// Unmasked: known value appears in results
expect(await page.textContent("body")).toContain(maskingData.knownUnmaskedValue);

// Masked: known value does NOT appear
expect(await page.textContent("body")).not.toContain(maskingData.knownUnmaskedValue);
```

## Running Tests

```bash
# Mode B: fully self-contained (requires pre-built binary)
npx playwright test

# Mode A: against existing server (fully automated)
BYTEBASE_URL=http://localhost:3000 BYTEBASE_USER=admin@bytebase.com BYTEBASE_PASS=secret npx playwright test

# Mode A: against existing server (browser login)
BYTEBASE_URL=http://localhost:3000 npx playwright test
```

## Generated Artifacts

These files are created during test runs and must be in `.gitignore`:
- `.e2e-env.json` — serialized TestEnv
- `.e2e-snapshot.json` — Mode A crash recovery snapshot (persisted to survive crashes)
- `.auth/state.json` — Playwright auth cookies/storage
- `test-results/` — Playwright test artifacts (screenshots, traces)
- `playwright-report/` — HTML test report
- `/tmp/bytebase-e2e-*` — Mode B temp data directories
- `/tmp/bytebase-e2e-pid` — Mode B PID file for orphan cleanup
