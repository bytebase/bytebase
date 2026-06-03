# E2E Tests

Playwright-based end-to-end tests for Bytebase. Starts a disposable Bytebase server, signs up an admin, provisions sample instances, runs tests, tears down.

## Prerequisites

1. **Bytebase binary with embedded frontend:**
   ```bash
   pnpm --dir frontend release
   go build -tags embed_frontend -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
   ```

2. **`psql` client** on PATH — required for DDL setup (the Bytebase query API is read-only; tests connect directly to the sample Postgres via Unix socket).

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
| `BYTEBASE_E2E_LICENSE` | **Yes** | — | Enterprise license JWT (see below) |
| `CI` | No | — | Enables CI-specific reporter |

### Enterprise license (`BYTEBASE_E2E_LICENSE`)

The suite exercises enterprise-only features (masking, JIT drawer, approval
rules, query-data-policy gates like `disableExport` / `disableCopyData`,
database groups). A license is **required** — without `BYTEBASE_E2E_LICENSE`
the run fails fast in global setup. There is no free-plan fallback.

The license JWT is signed by Bytebase's license RSA key — ask
Bytebase ops for a dev/test license. **Never commit it to the repo
or paste it into chat / issue trackers / memory files.**

Recommended local options (pick one):

```bash
# (a) Out-of-repo file + on-demand env (recommended)
mkdir -p ~/.config/bytebase
chmod 700 ~/.config/bytebase
echo '<your-jwt>' > ~/.config/bytebase/e2e-license
chmod 600 ~/.config/bytebase/e2e-license

# Then when running the suite:
BYTEBASE_E2E_LICENSE=$(cat ~/.config/bytebase/e2e-license) \
  pnpm exec playwright test

# (b) direnv (auto-loaded on cd into the repo)
echo 'export BYTEBASE_E2E_LICENSE=$(cat ~/.config/bytebase/e2e-license)' \
  >> .envrc
direnv allow

# (c) Shell rc / login profile (always available)
# Add to ~/.zshrc.local or similar gitignored file:
export BYTEBASE_E2E_LICENSE='<your-jwt>'
```

Do **not** put the JWT in `frontend/.env*` or any file inside the
repo, even if gitignored — too easy to accidentally commit, and
`.env*` files inside repos are commonly opened by IDEs / pasted
into screenshots.

## Directory Layout

```
frontend/tests/e2e/
├── README.md              — this file (human docs)
├── AGENTS.md              — conventions + QA doctrine for AI agents / contributors
├── framework/             — shared test infrastructure
│   ├── api-client.ts      — Bytebase v1 REST API wrapper
│   ├── env.ts             — TestEnv interface, serialization
│   ├── mode-start-new-bytebase.ts — server lifecycle (start/stop/cleanup)
│   ├── global-setup.ts    — starts the server before any tests
│   ├── global-teardown.ts — stops the server after all tests
│   ├── setup-project.ts   — auth + discovery (runs as a Playwright setup project)
│   ├── seed-test-data.ts  — workspace-level baseline seeded once per boot
│   ├── sign-in.ts         — POST-login helper that captures per-user storageState
│   └── psql.ts            — psql-over-Unix-socket helpers for DDL/DML setup
├── sql-editor/            — SQL Editor suite (connection, result, tabs, worksheet,
│                            schema, admin-mode, history, jit, permissions,
│                            workspace-gates, batch, misc) + sql-editor.page.ts
├── plan-detail/           — plan detail suite (checks, rollout, sections, tasks)
│                            + plan-detail.page.ts, plan-helpers.ts
├── workspace/             — workspace-level suite (external-URL banner)
└── masking-exemption/     — masking exemption suite
    ├── masking-exemption.spec.ts
    └── masking-exemption.page.ts
```

## How It Works

1. **`globalSetup`**: Cleans orphaned processes, starts Bytebase + embedded Postgres on a random high port, signs up the admin (`demo@example.com` / `12345678`), and calls `SetupSample` to provision the sample project and instances on `PORT+3` / `PORT+4`.
2. **Setup project** (runs as a Playwright test before all others): Logs in as the admin, discovers instances/databases/projects, saves auth state to `.auth/state.json`.
3. **Tests run**: Each spec file shares one browser context/page (see [AGENTS.md](./AGENTS.md)). Tests call `loadTestEnv()` to get the API client and env data, create their own test data, and run UI-driven verification.
4. **`globalTeardown`**: Kills the server process group, removes temp data dir and PID file.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Bytebase binary not found` | Run the build commands above or set `BYTEBASE_BIN` |
| `spawn psql ENOENT` | Install Postgres client: `brew install postgresql` or `apt install postgresql-client` |
| Server won't start / port in use | `lsof -ti:18234 \| xargs kill` and check for orphan processes matching `bytebase-e2e` |
| Stale PID file | `rm /tmp/bytebase-e2e-pid` |
| Stale auth state | `rm -rf frontend/.auth/ frontend/tests/.auth/` |
| "This Bytebase build does not bundle frontend" error | Binary was built without `embed_frontend` tag — rebuild with the prerequisite commands above |
