# Cloud scheduled task: prompt and configuration

Everything the routine needs is in this repository; the prompt only points at
the skill and sets the policy knobs. Keep the prompt short and the skill long,
so changes land through PRs rather than by editing the task.

## Prompt (paste as the task's message)

```
Run the Bytebase e2e regression routine on origin/main.

Read .claude/skills/e2e-routine/SKILL.md first and follow its phases and invariants exactly; use its scripts under .claude/skills/e2e-routine/scripts/ for the mechanical steps and paste each phase's gate evidence in your transcript.

Policy for this task:
- E2E_ROUTINE_FILE_ISSUES, E2E_ROUTINE_UMBRELLA and the Bash timeouts are set on the cloud environment. Do not file into Linear unless E2E_ROUTINE_FILE_ISSUES=1; otherwise write the proposals in the report.
- Phase 1 exit 3 or 4 means nothing to do: post the one-line note and stop.
- Never push to main. Push only claude/e2e-ledger (ledger.sh) and claude/e2e-adapt-<date>.
- Never print the license or the contents of frontend/.env.dev.
- Never kill a process you did not start.
- Linear: team BOT only, under the umbrella issue; never edit or close an existing issue.
- Finish with Phase 6 even when an earlier phase failed, and end your final message with the report.
```

## Bootstrap prompt (until the skill is on `main`)

The cloud clone starts on `main`. While the skill lives on a branch, prepend
this to the prompt and drop it once the PR merges:

```
First: git fetch origin worktree-e2e-routine && git checkout -q worktree-e2e-routine. Then read .claude/skills/e2e-routine/SKILL.md from that checkout. For this bootstrap run use FORCE=1 on Phase 1, run Phases 0 to 4 and 6, skip the adaptation PR of Phase 5 (list what it would change instead), and report build and suite timings.
```

## Environment `bytebase-e2e` (create once at claude.ai/code, Environments)

| Setting | Value |
|---|---|
| Network access | Custom: check "Also include default list", add `cdn.playwright.dev` and `playwright.download.prss.microsoft.com` |
| Setup script | contents of `scripts/setup-env.sh` |
| Environment variables | `BASH_DEFAULT_TIMEOUT_MS=600000`, `BASH_MAX_TIMEOUT_MS=3600000`, `E2E_ROUTINE_FILE_ISSUES=0`, `E2E_ROUTINE_UMBRELLA=BOT-<n>` |
| Network secrets | none needed (the dev license is in the repo; GitHub goes through the proxy; Linear is a connector) |

Switch `E2E_ROUTINE_FILE_ISSUES` to `1` after a week of reviewing proposals.

## Task settings

| Setting | Value |
|---|---|
| Repository | `bytebase/bytebase` (needs the Claude GitHub App on the repo) |
| Environment | `bytebase-e2e` |
| Schedule | daily; `7 14 * * *` UTC = 10:07 New York = 22:07 Beijing, after the day's merges |
| Model | `claude-opus-5-5` recommended (the value is in the triage); `claude-sonnet-5-5` is cheaper and fine for green nights |
| Connectors | Linear only (remove the rest); Slack too only when `E2E_ROUTINE_SLACK_CHANNEL` is used |
| Notifications | push on completion |

## Create body for `RemoteTrigger` / the API

```json
{
  "name": "Bytebase e2e regression (nightly)",
  "cron_expression": "7 14 * * *",
  "enabled": true,
  "job_config": {
    "ccr": {
      "environment_id": "<env id of bytebase-e2e>",
      "session_context": {
        "model": "claude-opus-5-5",
        "sources": [{"git_repository": {"url": "https://github.com/bytebase/bytebase"}}],
        "allowed_tools": ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch"]
      },
      "events": [{"data": {"uuid": "<lowercase v4 uuid>", "session_id": "", "type": "user", "parent_tool_use_id": null,
        "message": {"role": "user", "content": "<the prompt above>"}}}]
    }
  },
  "mcp_connections": [{"connector_uuid": "<Linear connector uuid>", "name": "Linear", "url": "https://mcp.linear.app/mcp"}]
}
```

## First run checklist

1. Create the umbrella issue in `BOT` by hand ("E2E routine: nightly regression runs") and put its id in `E2E_ROUTINE_UMBRELLA`.
2. Create the environment; start one interactive cloud session in it and run `check-tools`, then `bash .claude/skills/e2e-routine/scripts/preflight.sh`. Every line must be `OK` or a port `WARN`.
3. In that session run Phases 1 to 3 by hand once (`FORCE=1`) and note the build and suite times; adjust `BASH_MAX_TIMEOUT_MS` if `wait` needs more than an hour.
4. Create the task with `E2E_ROUTINE_FILE_ISSUES=0`; use **Run now**; read the transcript, not just the green status.
5. After a week of sane proposals, set `E2E_ROUTINE_FILE_ISSUES=1`.
