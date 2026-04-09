# E2E Test Framework

Playwright-based e2e tests for Bytebase. Starts a disposable Bytebase server with `--demo` data, runs tests, tears down.

## Prerequisites

Build the binary with embedded frontend:

```bash
pnpm --dir frontend release
go build -tags embed_frontend -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

## Running Tests

```bash
# Headless (default)
cd frontend && pnpm exec playwright test

# Headed (watch tests run in browser)
cd frontend && BYTEBASE_HEADED=1 pnpm exec playwright test

# With list reporter
cd frontend && pnpm exec playwright test --reporter=list
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BYTEBASE_BIN` | No | `./bytebase-build/bytebase` | Path to pre-built binary |
| `BYTEBASE_STARTUP_TIMEOUT` | No | `300000` (5 min) | Server startup timeout in ms |
| `BYTEBASE_HEADED` | No | — | Set to `1` for headed browser |
| `CI` | No | — | Enables CI-specific reporter |

## How It Works

1. **globalSetup**: Cleans orphaned processes, starts Bytebase with `--demo` + embedded Postgres on a random port
2. **Setup project**: Logs in as demo admin, discovers instances/databases/projects, saves auth state
3. **Tests run**: Each test reads `TestEnv` via `loadTestEnv()`, uses API for setup and browser for verification
4. **globalTeardown**: Kills server, removes temp dir and PID file

## Writing a New Test Suite

```typescript
import { test, expect } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

let env: ReturnType<typeof loadTestEnv>;

test.beforeAll(async () => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  // Feature-specific setup via API
});

test("example", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/...`);
  await expect(page.getByText("expected")).toBeVisible();
});
```

## Troubleshooting

- **Port in use**: `lsof -ti:18234 | xargs kill`
- **Binary not found**: Run the build commands above. Or set `BYTEBASE_BIN`.
- **Stale PID**: `rm /tmp/bytebase-e2e-pid`
- **Stale auth**: `rm -rf frontend/.auth/`
