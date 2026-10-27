# E2E Tests

Playwright-based end-to-end tests for Bytebase. Starts a disposable Bytebase server, signs up an admin, provisions sample instances, runs tests, tears down.

## Prerequisites

1. **Bytebase binary with embedded frontend**, from the repository root:
   ```bash
   (cd frontend && pnpm release)
   bash frontend/tests/e2e/build-server.sh
   ```
   `build-server.sh` builds a dev binary, so the dev license validates, with
   dev-only product switches set to their release values: the suite tests what
   release builds ship. Today that switch is SQL review V2, which a plain dev
   build turns on; the suite covers SQL review V1.

2. **`psql` client** on PATH — required for DDL setup (the Bytebase query API is read-only; tests connect directly to the sample Postgres via Unix socket).

3. **Playwright Chromium** — installed automatically. `globalSetup` runs
   `playwright install chromium` before booting the server, so a fresh checkout
   or a `@playwright/test` version bump (which invalidates the cached browser)
   self-heals on the next run. Playwright browsers live in a global cache keyed
   to the package version, outside `node_modules`, so `pnpm i` never fetches
   them. The auto-install is **skipped when `BYTEBASE_BROWSER_CHANNEL` is set**
   (see below) — that mode drives a locally installed browser and needs no
   download. To pre-install (or fix a failed auto-install) manually:
   ```bash
   pnpm --dir frontend test:e2e:install   # or: pnpm exec playwright install chromium
   ```

## Running Tests

```bash
cd frontend

# Headless (default)
pnpm exec playwright test

# Headed — watch tests run in a visible browser
BYTEBASE_HEADED=1 pnpm exec playwright test

# Single test file
pnpm exec playwright test masking-exemption
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BYTEBASE_BIN` | No | `./bytebase-build/bytebase` | Path to pre-built binary |
| `BYTEBASE_STARTUP_TIMEOUT` | No | `300000` (5 min) | Server startup timeout in ms |
| `BYTEBASE_HEADED` | No | — | Set to `1` for headed browser |
| `BYTEBASE_BROWSER_CHANNEL` | No | — | Drive a locally installed browser channel (e.g. `chrome`) instead of the downloaded Chromium; skips the Chromium auto-install. Useful when the download is unavailable (offline). |
| `BYTEBASE_E2E_LICENSE` | No | dev license from `frontend/.env.dev` | Enterprise license JWT override (see below) |
| `CI` | No | — | Enables CI-specific reporter |

### Enterprise license

The suite exercises enterprise-only features (masking, JIT drawer, approval
rules, query-data-policy gates like `disableExport` / `disableCopyData`,
database groups) and does not run on the free plan. Global setup installs a
license before any spec runs, resolved in this order:

1. `BYTEBASE_E2E_LICENSE`, when set.
2. The `BB_DEV_ENTERPRISE_LICENSE` line in `frontend/.env.dev`: the dev-signed
   enterprise license the dev build offers in its profile menu.

So a plain `pnpm exec playwright test` works with no extra setup. The dev
license is signed with the dev key, which only a binary built **without** the
`release` tag accepts; the build command above is that binary. A license
signed with the production key needs a `-tags embed_frontend,release` build
and `BYTEBASE_E2E_LICENSE`.

Never print a license into logs, chat, issue trackers or memory files, and
never commit a production-signed one.

## Nightly routine

The suite is in no CI workflow. A cloud scheduled task runs it nightly against
`main` with [`.claude/skills/e2e-routine`](../../../.claude/skills/e2e-routine/SKILL.md):
build, run, classify each red test, attribute an owner, file or propose Linear
issues, and record the run on the `claude/e2e-ledger` branch. The same skill
runs locally from a worktree (`/e2e-routine`).

## Directory Layout

```
frontend/tests/e2e/
├── README.md              — this file (human docs)
├── AGENTS.md              — test-authoring conventions for AI agents / contributors
├── framework/             — shared test infrastructure
│   ├── api-client.ts      — Bytebase v1 REST API wrapper
│   ├── env.ts             — TestEnv interface, serialization
│   ├── mode-start-new-bytebase.ts — server lifecycle (start/stop/cleanup)
│   ├── global-setup.ts    — starts the server before any tests
│   ├── global-teardown.ts — stops the server after all tests
│   ├── setup-project.ts   — auth + discovery (runs as a Playwright setup project)
│   ├── seed-test-data.ts  — workspace-level baseline seeded once per boot
│   ├── sign-in.ts         — POST-login helper that captures per-user storageState
│   ├── psql.ts            — psql-over-Unix-socket helpers for DDL/DML setup
│   └── bare-server.ts     — a second, first-run server for the spec that needs one
├── sql-editor/            — SQL Editor suite (connection, result, tabs, saved query,
│                            schema, admin-mode, history, jit, permissions,
│                            workspace-gates, batch, misc) + sql-editor.page.ts
├── plan-detail/           — plan detail suite (lifecycle, checks, rollout, sections, tasks)
│                            + plan-detail.page.ts, plan-helpers.ts
├── workspace/             — workspace-level suite (external-URL banner)
├── workspace-setup/       — first-run setup wizard, on a server of its own
└── masking-exemption/     — masking exemption suite
    ├── masking-exemption.spec.ts
    └── masking-exemption.page.ts
```

## How It Works

1. **`globalSetup`**: Cleans orphaned processes, starts Bytebase + embedded Postgres on a random high port, signs up the admin (`demo@example.com` / `12345678`), creates `project-sample`, and calls `PrepareSampleProjectInstance` to provision its instance on `PORT+3`.
2. **Setup project** (runs as a Playwright test before all others): Logs in as the admin, discovers the sample database, adds `hr_prod` on the same instance for multi-database coverage, and saves auth state to `.auth/state.json`.
3. **Tests run**: New independent tests use isolated browser contexts; existing suites may share a context under the exceptions in [AGENTS.md](./AGENTS.md). Tests call `loadTestEnv()` to get the API client and env data, create their own test data, and run UI-driven verification.
4. **`globalTeardown`**: Kills the server process group, removes temp data dir and PID file.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Bytebase binary not found` | Run the build commands above or set `BYTEBASE_BIN` |
| `spawn psql ENOENT` | Install Postgres client: `brew install postgresql` or `apt install postgresql-client` |
| Server won't start / port in use | `lsof -ti:18234 \| xargs kill` and check for orphan processes matching `bytebase-e2e` |
| Stale PID file | `rm /tmp/bytebase-e2e-pid` |
| Stale auth state | `rm -rf frontend/.auth/ frontend/tests/.auth/` |
| `postmaster became multithreaded during startup` (macOS) | The shell has no valid UTF-8 `LANG`; the harness defaults one, so export `LANG=en_US.UTF-8` only when running the binary by hand |
| `token signature is invalid` on license install | The license is signed with the production key: build with `-tags embed_frontend,release` |
| "This Bytebase build does not bundle frontend" error | Binary was built without `embed_frontend` tag — rebuild with the prerequisite commands above |
