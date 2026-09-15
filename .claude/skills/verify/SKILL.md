---
name: verify
description: Build and drive Bytebase with the Playwright E2E harness when a change needs verification against the running app.
---

# Verify against the running app

Run commands from the repository root unless a command changes directory.
Read [the E2E README](../../../frontend/tests/e2e/README.md) for prerequisites,
license configuration, browser selection, and troubleshooting. Before writing
or changing tests, read [the E2E instructions](../../../frontend/tests/e2e/AGENTS.md).

## Build

```bash
pnpm --dir frontend release
go build -tags embed_frontend -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

The `embed_frontend` tag is required to serve the UI. Rebuild the affected
frontend bundle/backend binary after code changes so the test exercises the fix.

## Run the affected scenario

With `BYTEBASE_E2E_LICENSE` configured as described in the README:

```bash
cd frontend
pnpm exec playwright test sql-editor/sql-editor-lsp.spec.ts --reporter=list
```

Replace the example spec with the scenario relevant to the change. The harness
owns disposable server startup, authentication, sample-instance provisioning,
and teardown; use its current implementation rather than hand-rolling bootstrap.
When a browser download is unavailable, `BYTEBASE_BROWSER_CHANNEL=chrome` selects
a locally installed Chrome.

## Evidence

Verify the user-visible outcome, including the failure path relevant to the
change. For LSP work, consult the existing LSP spec's buffer helpers and websocket
assertions; connection status is a tooltip, not persistent page text.
Report the scenario, result, and any missing prerequisite or unverified behavior.
