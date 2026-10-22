# The run ledger

Branch `claude/e2e-ledger` of this repository (orphan: no shared history with
`main`), written only by `scripts/ledger.sh record`. It exists so a run can
answer "what did the last run cover" without a second repository or a Linear
query, and so flake history survives between sessions.

```
README.md
last-run.json            # the most recent run (the file should-run.sh reads)
runs/<YYYY-MM-DD>-<sha7>.json
flakes.json              # {"runs": [sha7 ...last 20], "tests": {"<file>::<title>": [sha7 of runs where it was flaky]}}
```

`last-run.json` / `runs/*.json`:

```json
{
  "run_id": "20261022T140712Z",
  "sha": "<40-hex main sha the binary was built from>",
  "date": "2026-10-22",
  "result": "green | red | infra",
  "totals": {"tests": 221, "passed": 218, "failed": 2, "flaky": 1, "lock_flipped": 0, "lock_holding": 7, "skipped": 0, "duration_s": 2410},
  "failed": ["sql-editor/sql-editor-result.spec.ts::Request export shows for a developer"],
  "flaky":  ["plan-detail/plan-detail-rollout.spec.ts::..."],
  "infra": null,
  "note": "free text: decisions in one line, PR link",
  "run_url": "https://claude.ai/code/..."
}
```

Rules:

- `LAST_GREEN` for attribution is the newest `runs/*.json` with `"result": "green"`;
  when none exists, the last run's `sha`; when the ledger is empty, 60 days
  back on `main`. A spec can have been red for longer than any window, so
  `attribute.py` widens on its own to the spec file's last change when nothing
  inside the window scores.
- A run records itself exactly once, after `summarize.py`, whatever the result.
  An aborted run (no `report.json`) records `infra` with the reason.
- Flake threshold: a test flaky in 3 of the last 5 recorded runs is an issue
  candidate (`ledger.sh flakes`).
- Nothing in the ledger is secret; the branch is public like the repository.
  Screenshots and traces stay in the run directory and go to Linear as
  attachments only when an issue is created.
