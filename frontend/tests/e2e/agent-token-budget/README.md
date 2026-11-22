# Agent Token Budget

These deterministic browser journeys use the real composer, tools, run writer,
and recovery controls. Every Chat request is intercepted, including unexpected
requests; no request reaches a configured provider or spends provider credits.
Responses use the generated Connect JSON or binary protobuf schema. The only
executed tools are harmless current-page reads and terminal answers.

## Standard Harness

From `frontend/`, with the prerequisites in the [E2E guide](../README.md):

```bash
pnpm exec playwright test agent-token-budget
```

The standard harness starts its disposable server and supplies authentication.
Because it has no provider configuration, this suite intercepts only AI readiness
reads: `GetActuatorInfo.aiEnabled` and `GetSetting("settings/AI").value.ai.enabled`.
It preserves unrelated response fields and does not write settings. All remaining
Bytebase API requests are real. Binary fixture handling is supported; the recorded
local QA run used the development server's JSON transport.

## Existing Server

Do not use the repository's normal Playwright config against an existing server:
its global setup starts a different server and provisions data. Use a standalone
config with this test directory, fresh browser contexts, no global setup/teardown,
no setup-project dependency, and no saved authentication state.

The local QA artifact includes `existing-server.config.ts`; run that config from
`frontend/` after supplying these variables through the process environment:

```bash
BYTEBASE_TOKEN_BUDGET_BASE_URL=http://localhost:4317 \
BYTEBASE_TOKEN_BUDGET_MOCK_READINESS=1 \
pnpm exec playwright test --config /path/to/existing-server.config.ts
```

`BYTEBASE_E2E_EMAIL` and `BYTEBASE_E2E_PASSWORD` must already be present in that
process environment. Never put their values in source, reports, traces, or saved
authentication state. The fixture uses cookie-based web login, and closes each
disposable context after its test. Without `BYTEBASE_TOKEN_BUDGET_MOCK_READINESS=1`,
existing-server mode uses the server's real readiness reads.

Set `BYTEBASE_TOKEN_BUDGET_SCREENSHOTS` to an absolute output directory to keep the
redacted screenshots outside Playwright's temporary test output. Screenshots are
labeled `deterministic-provider-mock`; full-viewport captures include portaled
panels without relying on viewport-relative clip coordinates. Navigation, banner,
Agent header, context labels, chat group headings, and timestamps are masked. The OS-dark
preference case is labeled `osdark-agent-light`: the Agent is not claimed to have
a separate dark theme.

## Coverage

The journeys exercise the 100K default, pending/790/800 thresholds,
preferences and minimize remount during a held request, pause after a complete
two-tool group, numeric validation, Cancel, increase-and-resume confirmation,
active-run Apply, End/new run,
completion exactly at and above the limit, partial/zero/failed usage, Stop/Retry,
reload recovery, two-tab contention, unsupported Web Locks, mobile panels, and
Chinese mobile labels. Recovery asserts exact unchanged read-tool call/result IDs,
distinct attempt IDs, the same logical run, no new user row, preserved draft,
and unchanged preference default. Desktop and mobile action checks verify that
exhaustion shows only Increase limit and End run, and that one confirmation resumes
the run. Geometry checks preserve the 12px composer gap without leaving empty space
when the budget band disappears.

Both preference and run-limit editors accept K tokens: `2` means 2,000 tokens.
The run editor is titled "Increase limit for this run" and preserves single-token
precision (`0.001K`), including a 7.202K usage minimum that requires at least
7.203K to continue. These overrides do not change the preference default.

These journeys do not establish all 37 rows of the browser matrix. Remaining
gaps include API permission/expiry, clarification variants, finalization recovery,
storage-failure injection, attempt/request-size caps, legacy migration, concurrent
different chats and late settlements, scope/archive boundaries, every locale,
exhaustive keyboard/focus checks, normal-overlay/critical-layer interactions, and
an actual dark-themed Agent. Unit and live-provider evidence are separate.
