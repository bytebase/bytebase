# E2E Tests

Playwright-based end-to-end tests for Bytebase. Starts a disposable Bytebase server with `--demo` data, runs tests, tears down.

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
| `CI` | No | — | Enables CI-specific reporter |

## Directory Layout

```
frontend/tests/e2e/
├── README.md              — this file (human docs)
├── AGENTS.md              — conventions for AI agents / contributors
├── framework/             — shared test infrastructure
│   ├── api-client.ts      — Bytebase v1 REST API wrapper
│   ├── env.ts             — TestEnv interface, serialization
│   ├── mode-start-new-bytebase.ts — server lifecycle (start/stop/cleanup)
│   ├── global-setup.ts    — starts the server before any tests
│   ├── global-teardown.ts — stops the server after all tests
│   └── setup-project.ts   — auth + discovery (runs as a Playwright setup project)
└── masking-exemption/     — feature test suite
    ├── masking-exemption.spec.ts
    └── masking-exemption.page.ts
```

## How It Works

1. **`globalSetup`**: Cleans orphaned processes, starts Bytebase with `--demo` + embedded Postgres on a random high port, reconciles sample instance data source ports.
2. **Setup project** (runs as a Playwright test before all others): Logs in as the demo admin (`demo@example.com` / `12345678`), discovers instances/databases/projects, saves auth state to `.auth/state.json`.
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
