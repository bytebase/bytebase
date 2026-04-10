# E2E Tests — Contributor & AI Agent Guide

Conventions for writing and maintaining Bytebase e2e tests. Follow these rules unless you have a strong, documented reason not to.

## Core Principles

### 1. Create your own test data, don't discover it

**Do:** Create a dedicated schema/table/rows for your test and drop them in `afterAll`.

```typescript
// In beforeAll — create what you need
execSql(env, dbName, `CREATE SCHEMA my_feature_test`);
execSql(env, dbName, `CREATE TABLE my_feature_test.t (id INT PRIMARY KEY, col TEXT)`);
execSql(env, dbName, `INSERT INTO my_feature_test.t VALUES (1, 'KnownValue')`);

// In afterAll — clean up
execSql(env, dbName, `DROP SCHEMA IF EXISTS my_feature_test CASCADE`);
```

**Don't:** Query `information_schema` or scan the demo data to find something usable. Discovery-based tests are fragile — they fail when demo data changes, when tables are empty, or when pre-existing masking hides the values you need.

**Why:** A test that owns its fixtures is deterministic. You know exactly what values exist, what's masked, what's not, and what state is left when the test completes. `masking-exemption.spec.ts:createMaskingTestData` is the canonical example.

### 2. Share a single browser across all tests in a file

**Do:** Create one `BrowserContext` and `Page` in the file-level `test.beforeAll`, and reuse them across every test in the file.

```typescript
let sharedContext: BrowserContext;
let page: Page;

test.beforeAll(async ({ browser }) => {
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test("my test", async () => {
  // use the shared `page` — no { page } destructuring
  await page.goto(...);
});
```

**Don't:** Use Playwright's default per-test `{ page }` fixture. It opens and closes a browser context for every test, which:
- Costs real time (especially on CI)
- Triggers NVirtualList first-render timing bugs
- Makes tests slower for no isolation benefit (we run `workers: 1` anyway)

**Side effects to clean up manually** (since we lose per-test isolation):
- Reset viewport in `afterAll` if you changed it mid-suite
- Clear URL/tab state explicitly if it matters (usually via `gotoWithDb` etc.)
- Reset policy state via API (`revokeAllExemptions`) in `beforeEach` when tests depend on clean state

### 3. API for setup, browser for verification

Bytebase's v1 REST API is your tool for fast, deterministic state setup. Use the browser only to verify what a user would see.

**Do:**
- Grant/revoke policies via API, then render them in the browser to assert the UI reflects the state.
- Create test data via `psql` (Unix socket), configure catalog via API, run the test UI flow in the browser.

**Don't:**
- Click through the UI to set up preconditions when an API call would do it.
- Screenshot-compare for logic assertions.

### 4. Use primary keys for masking verification queries

When verifying masked/unmasked data in the SQL editor, query by the row's primary key, not by `LIMIT n`. Ordering is not deterministic without `ORDER BY`, and `LIMIT` may not include your known value.

```typescript
const sql = `SELECT "${col}" FROM "${schema}"."${table}" WHERE "${pkColumn}" = '${pkValue}'`;
```

### 5. Avoid `waitForTimeout` for arbitrary delays

**Do:** Wait for specific conditions via locator auto-wait, `waitForResponse`, `waitForURL`, or `expect(...).toBeVisible()`.

**Don't:** Sprinkle `await page.waitForTimeout(500)` everywhere. Every arbitrary sleep is a flakiness source on slow CI.

*(Existing tests have technical debt here — new tests should not add to it.)*

### 6. Cross-platform keyboard shortcuts

Use `ControlOrMeta+a` (portable), not `Meta+a` (Mac-only) or `Control+a` (Linux/Windows-only).

### 7. Prefer `data-testid` over class-based selectors

Tailwind class substrings like `[class*='border border-gray']` break on any CSS refactor. If you need a new locator, add a `data-testid` attribute to the component. Existing class-based selectors are technical debt.

## Directory Layout

```
frontend/tests/e2e/
├── README.md              — human-facing docs (how to run tests)
├── AGENTS.md              — this file (conventions)
├── framework/             — shared infrastructure (don't add feature code here)
└── <feature-name>/        — one directory per feature test suite
    ├── *.spec.ts          — test files
    └── *.page.ts          — page object models (feature-specific)
```

## File Responsibilities

| File | Responsibility |
|------|----------------|
| `framework/api-client.ts` | Bytebase v1 REST API wrapper, token refresh on 401 |
| `framework/env.ts` | `TestEnv` interface, load/save via `.e2e-env.json` |
| `framework/mode-start-new-bytebase.ts` | Disposable server lifecycle + port reconciliation |
| `framework/global-setup.ts` | Starts server before any test runs |
| `framework/global-teardown.ts` | Stops server after all tests finish |
| `framework/setup-project.ts` | Auth + instance/database discovery, writes env + auth state |

## Adding a New Feature Test Suite

1. **Create directory**: `tests/e2e/<feature-name>/`
2. **Write page objects** in `<feature>.page.ts` (one class per UI surface). Accept `baseURL` in the constructor so pages can navigate via absolute URLs.
3. **Write spec file** `<feature>.spec.ts`:
   - Import `loadTestEnv` from `../framework/env`
   - Declare the shared `page` / `sharedContext` at module level
   - In file-level `test.beforeAll`: `loadTestEnv()`, login, create test data, open the shared browser
   - In `test.afterAll`: close browser, clean up test data via the same code path that created it
4. **Use the canonical example** — read `frontend/tests/e2e/masking-exemption/masking-exemption.spec.ts` as a reference.

## Extending the API Client

- Add methods to `BytebaseApiClient` in `framework/api-client.ts`
- Use `this.request<T>()` with typed responses
- Include `pageSize=100` on list endpoints
- Only add methods actually used by a test — no speculative API coverage

## Running DDL / DML (for test data setup)

The Bytebase query API is **read-only**. For DDL/DML, use the `execSql` helper (see `masking-exemption.spec.ts`) which shells out to `psql` via Unix socket to the sample Postgres instance.

**Port layout**: the disposable Bytebase server on `PORT` starts sample Postgres at:
- `PORT + 3` → `test-sample-instance` (hr_test)
- `PORT + 4` → `prod-sample-instance` (hr_prod)

Get the correct port from `getInstance(env.instance)` rather than hardcoding the offset.

## Known Constraints

- **Demo mode only**: tests run against `--demo` data. Demo data is pre-seeded with sample instances and has a built-in enterprise license.
- **Serial execution**: `fullyParallel: false` + `workers: 1`. Tests within and across files are sequential.
- **`psql` dependency**: must be on PATH for DDL/DML setup.
- **Unix-like OS only**: the sample Postgres uses Unix sockets in `/tmp`.
- **Demo admin credentials**: hardcoded `demo@example.com` / `12345678` (the well-known demo defaults, not a secret).
