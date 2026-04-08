# Bytebase E2E Test Framework

Playwright-based end-to-end test framework for Bytebase. Supports two modes: spinning up a disposable server (Mode B) or connecting to one you already have running (Mode A).

---

## Quick Start

### Mode B — Disposable server (no existing Bytebase required)

Build the binary, then run tests. The framework starts and stops the server automatically.

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
npx playwright test
```

### Mode A — Use an existing server (automated login)

Set `BYTEBASE_URL` to point at your running instance and provide credentials.

```bash
BYTEBASE_URL=http://localhost:3000 \
BYTEBASE_USER=admin@bytebase.com \
BYTEBASE_PASS=secret \
npx playwright test
```

### Mode A — Use an existing server (browser login)

Omit credentials to be prompted to log in via the browser during the setup step.

```bash
BYTEBASE_URL=http://localhost:3000 npx playwright test
```

---

## Environment Variable Reference

| Variable | Mode | Required | Default | Description |
|---|---|---|---|---|
| `BYTEBASE_URL` | A | Yes (triggers Mode A) | — | Base URL of the running Bytebase instance |
| `BYTEBASE_USER` | A | No | `admin@bytebase.com` | Admin email for API login |
| `BYTEBASE_PASS` | A | No | — | Admin password; omit for browser login |
| `BYTEBASE_BIN` | B | No | `./bytebase-build/bytebase` | Path to the Bytebase binary |
| `BYTEBASE_STARTUP_TIMEOUT` | B | No | `60000` | Milliseconds to wait for the server to become healthy |
| `CI` | Both | No | — | When set, enables automatic restore on crash recovery without prompting |

---

## How to Write a New Feature Test Suite

### 1. Create the spec file

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

test.describe("My Feature", () => {
  let env: ReturnType<typeof loadTestEnv>;
  let snapshot: Snapshot;

  // Define what state to capture and restore
  const scope: SnapshotScope = {
    policies: ["projects/my-project/policies/some_policy"],
    catalogs: ["instances/my-instance/databases/my-db"],
  };

  test.beforeAll(async () => {
    // Load shared environment (base URL, API client, discovered resources)
    env = loadTestEnv();

    // Feature-specific discovery: find or create what your tests need
    // e.g. locate the project, instance, database your tests will use

    // For Mode A: capture state before the test suite runs so it can be
    // restored afterward, leaving the server clean for the next run
    if (env.mode === "local") {
      snapshot = await createSnapshot(env.api, scope);
    }

    // Use env.api for API-based setup (faster and more reliable than UI)
    // e.g. await env.api.upsertPolicy(...)
  });

  test.afterAll(async () => {
    if (env?.mode === "local" && snapshot) {
      await restoreSnapshot(env.api, snapshot);
      deletePersistedSnapshot();
    }
  });

  test("verifies the feature in the browser", async ({ page }) => {
    // Use env.baseURL or the Playwright baseURL config for navigation
    await page.goto(`/my-feature-path`);

    // Assert using page objects or direct locators
    await expect(page.getByRole("heading", { name: "My Feature" })).toBeVisible();
  });
});
```

### 2. Separate API setup from browser verification

- **API** (`env.api`): create projects, configure policies, seed data — anything that doesn't need the browser.
- **Browser** (`page`): verify the UI renders correctly and interactions work.

This keeps tests fast and makes failures easier to diagnose.

---

## How Snapshot/Restore Works

The snapshot system protects a pre-existing Bytebase server from accumulating test state.

1. **`SnapshotScope`** — describes what to capture: policy paths, catalog paths, and optional SQL queries for raw instance data.
2. **`createSnapshot(api, scope)`** — reads the current state of each item in the scope and writes it to `.e2e-snapshot.json` on disk.
3. **`restoreSnapshot(api, snapshot)`** — writes the captured state back via the API, returning the server to the exact condition it was in before the test suite ran.
4. **`deletePersistedSnapshot()`** — removes the on-disk file once restore succeeds.

**Crash recovery**: if a test run is interrupted before `afterAll` runs, the snapshot file remains on disk. On the next run:
- In CI (`CI` env var set): the framework automatically restores and deletes the snapshot.
- Locally: the framework prompts you to confirm before restoring.

Mode B (disposable server) does not need snapshots — the server and its data are discarded after the run.

---

## Troubleshooting

### Port already in use

Mode B uses port `18234` by default. Kill whatever is holding it:

```bash
lsof -ti:18234 | xargs kill
```

### Binary not found

Build the binary before running Mode B tests:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Or point `BYTEBASE_BIN` at an existing binary:

```bash
BYTEBASE_BIN=/path/to/bytebase npx playwright test
```

### Stale PID file from a crashed run

If Mode B left a stale PID file, delete it manually:

```bash
rm /tmp/bytebase-e2e-pid
```

### Stale auth state

If the browser session is rejected (login loops, 401 errors), delete the saved auth state and re-authenticate:

```bash
rm -rf frontend/.auth/
npx playwright test
```
