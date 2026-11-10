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
(cd frontend && pnpm release)
bash frontend/tests/e2e/build-server.sh
```

`build-server.sh` builds with the `embed_frontend` tag, which serves the UI,
and without `release`, so the dev license validates; it sets dev-only product
switches to their release values, so SQL review runs V1 (see the E2E README).
Rebuild the affected frontend bundle/backend binary after code changes so the
test exercises the fix.

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
