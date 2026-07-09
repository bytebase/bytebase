---
name: verify
description: Build, launch, and drive Bytebase end-to-end to verify a change against the running app (backend + production frontend bundle + Playwright).
---

# Verifying Bytebase changes against the running app

## Build the app (production bundle, embedded)

```bash
pnpm --dir frontend release   # vite build → backend/server/dist
go build -tags embed_frontend -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

The `embed_frontend` tag is required — without it the binary serves no UI.

## Drive it with the e2e harness

The harness (`frontend/tests/e2e/`) boots a fresh instance per run: embedded
Postgres, first-signup admin (`demo@example.com` / `12345678`), and sample
instances via `POST /v1/actuator:setupSample` (hr_prod / hr_test with the
employee schema).

```bash
cd frontend
export BYTEBASE_E2E_LICENSE=$(grep BB_DEV_ENTERPRISE_LICENSE .env.dev-local | cut -d= -f2)
pnpm exec playwright test sql-editor/<spec>.spec.ts --reporter=list
```

- License is mandatory (suite is enterprise-only); the dev enterprise JWT in
  `frontend/.env.dev-local` works.
- If the Playwright browser download hangs (it has here — 40min stall), use
  the locally installed Chrome instead: `export BYTEBASE_BROWSER_CHANNEL=chrome`
  (wired through `playwright.config.ts` `use.channel`).
- Server boot takes 3–6 min (embedded PG + migrations + sample instances);
  it is spent in globalSetup, not test time.

## Iterating without a 6-min reboot per attempt

For debugging, boot a standalone server once and drive it with a plain
Playwright script; a reusable pair of scripts exists in session scratchpads
under `lsp-debug/` (boot-server.mjs / drive.mjs pattern): spawn
`bytebase-build/bytebase --port <p> --data <tmp>` with `PG_URL=""`, poll
`/healthz`, signup/login via `/v1/auth/*`, `POST /v1/actuator:setupSample`,
poll `/v1/instances` until the sample databases appear.

## Gotchas observed

- `ControlOrMeta+a` select-all in Monaco is unreliable on macOS headless
  Chrome — clear buffers by backspacing the measured content length
  (see `setBuffer` in `sql-editor/sql-editor-lsp.spec.ts`).
- Clicking a suggest-widget row to accept is flaky; assert on the row's
  visibility and dismiss with Escape.
- The LSP connection indicator text ("connected") lives in a hover tooltip —
  don't locate it; assert on websocket frames instead (see the
  `page.on("websocket")` frame capture in the LSP spec).
- Backend LSP capabilities: completion (trigger chars `.` and space) and
  executeCommand only — no hover.
