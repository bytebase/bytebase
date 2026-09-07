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

**Don't:** Query `information_schema` or scan the sample data to find something usable. Discovery-based tests are fragile — they fail when sample data changes, when tables are empty, or when pre-existing masking hides the values you need.

**Why:** A test that owns its fixtures is deterministic. You know exactly what values exist, what's masked, what's not, and what state is left when the test completes. `masking-exemption.spec.ts:createMaskingTestData` is the canonical example.

### 2. Isolate browser state by default

Use Playwright's per-test `{ page }` fixture for new independent tests. Serial
execution does not isolate cookies, local storage, navigation, or in-memory UI
state. Tests still own their database and policy setup and cleanup.

A shared context/page is appropriate for a deliberate multi-step scenario or a
measured setup bottleneck. Document the reason and reset state between independent
tests. Existing shared-context suites can remain; preserve their cleanup when
editing them. Keep a dependent user journey in one test rather than relying on
another test's leftovers.

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

**Exception — Monaco editors:** Monaco derives its `CtrlCmd` modifier from the user agent, and Playwright's headless Chromium reports a Windows UA on every host, while Playwright resolves `ControlOrMeta` from the host OS. For Monaco's own keybindings (select-all, editor actions) use `Control+…`; `ControlOrMeta+…` is silently ignored on macOS hosts (Linux CI hosts happen to agree, so CI does not catch it). Browser-native editing commands (copy/paste) follow the host OS instead, so don't use a key for them at all — call `document.execCommand("copy")` from `page.evaluate` right after a keypress (see `SchemaEditorPage.planStatementText`).

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
   - Use the per-test `{ page }` fixture and configured authentication state
   - Load the test environment and create owned data in setup hooks
   - Clean up owned data and policy state in teardown hooks
4. **Use the canonical example** — read `frontend/tests/e2e/masking-exemption/masking-exemption.spec.ts` for API setup and owned fixtures; its shared browser is not the default for new tests.

## Extending the API Client

- Add methods to `BytebaseApiClient` in `framework/api-client.ts`
- Use `this.request<T>()` with typed responses
- Include `pageSize=100` on list endpoints
- Only add methods actually used by a test — no speculative API coverage

## Running DDL / DML (for test data setup)

The Bytebase query API is **read-only**. For DDL/DML, use the `execSql` helper (see `masking-exemption.spec.ts`) which shells out to `psql` via Unix socket to the sample Postgres instance.

**Port layout**: the disposable Bytebase server on `PORT` starts the project-scoped sample Postgres instance (`hr_test`) at `PORT + 3`.

Get the correct port from `getInstance(env.instance)` rather than hardcoding the offset.

## Known Constraints

- **Sample-data bootstrap**: `globalSetup` creates `project-sample` and calls `PrepareSampleProjectInstance`. The single project-scoped sample Postgres instance comes up on `PORT+3` with `hr_test`; the setup project adds `hr_prod` on that instance as an E2E-only multi-database fixture.
- **License required**: the suite exercises enterprise-only features (masking, JIT, approval workflow, query-data-policy, database groups) and does NOT run on the free plan. Set `BYTEBASE_E2E_LICENSE` to a license JWT signed by Bytebase's license key (ask Bytebase ops for a dev/test license; not stored in this repo); `globalSetup` installs it via `PATCH /v1/subscription/license`. If the env var is absent the bootstrap throws and the whole run stops — there is no per-spec `test.skip` fallback.
- **Serial execution**: `fullyParallel: false` + `workers: 1`. Tests within and across files are sequential.
- **`psql` dependency**: must be on PATH for DDL/DML setup.
- **Unix-like OS only**: the sample Postgres uses Unix sockets in `/tmp`.
- **Admin credentials**: hardcoded `demo@example.com` / `12345678`. The first user created via `/v1/auth/signup` becomes workspace admin, so e2e signs up this fixture.
- **DBA fixture**: `dba1@example.com` / `12345678` is created during `globalSetup` and granted `roles/workspaceDBA`. Used as the second approver by plan-detail approval specs. If a spec needs additional users (developer, QA, etc.), provision them the same way in `mode-start-new-bytebase.ts` — don't assume they exist.

## Assertions and regression coverage

- Assert user-visible outcomes. For a reproduced regression, demonstrate that the test fails on the buggy behavior and passes with the fix; a temporary mutation can establish this when useful.
- Use `test.fail()` for a known-unfixed regression so an unexpected pass is visible. Explain any skipped coverage.
- Test permission-sensitive flows as both an authorized and a restricted user.
- Parameterize relevant data shapes with a loop that declares Playwright `test(...)` cases.
- For visual bugs, use screenshot baselines or assertions on the relevant visual relationship. Capture request counts when the regression concerns unexpected refetches.
- Keep sequence-dependent interactions in one user journey; assert its terminal outcome.

## Exploratory QA

For exploratory or release QA, read [the field manual](../../../docs/agents/exploratory-qa.md).
Its exhaustive interaction and observation passes apply to that scoped activity,
not to every E2E test edit.
