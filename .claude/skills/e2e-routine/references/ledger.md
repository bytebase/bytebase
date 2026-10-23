# The run ledger

Branch `claude/e2e-ledger` of this repository (orphan: no shared history with
`main`), written only by `scripts/ledger.sh record`. It exists so a run can
answer "what did the last run cover" without a second repository or a Linear
query, and so flake history survives between sessions.

```
README.md
last-run.json            # the most recent run (the file should-run.sh reads)
runs/<YYYY-MM-DD>-<sha7>-<run_id>.json
flakes.json              # {"runs": [sha7 ...last 20], "tests": {"<file>::<title>": [sha7 of runs where it was flaky]}}
```

`last-run.json` / `runs/*.json`:

```json
{
  "run_id": "20261022T140712Z",
  "sha": "<40-hex main sha the binary was built from>",
  "date": "2026-10-22",
  "result": "green | red | infra | incomplete",
  "totals": {"tests": 221, "passed": 218, "failed": 2, "flaky": 1, "lock_flipped": 0, "lock_holding": 7, "skipped": 0, "duration_s": 2410},
  "failed": ["sql-editor/sql-editor-result.spec.ts::Request export shows for a developer"],
  "flaky":  ["plan-detail/plan-detail-rollout.spec.ts::..."],
  "infra": null,
  "note": "free text: decisions in one line, PR link",
  "run_url": "https://claude.ai/code/..."
}
```

Rules:

- `sha` is always a commit on `main`: the one whose product files (`frontend`,
  `backend`, `proto`, `go.mod`, `go.sum`, outside `frontend/tests/e2e`) the
  binary was built from. Local commits that touch only the e2e tree or
  `.claude/` (an adaptation, an overlay) do not change it, so `run-suite.sh`
  records the merge base with `origin/main` in that case. A tree whose
  product files match no `main` commit is a branch, not main: `ledger.sh
  record` refuses it (exit 1) instead of writing a SHA no later run can use.
- The attribution baseline (`ledger.sh last-green`) is the newest `runs/*.json` with `"result": "green"`
  whose `sha` is on `main`; when none exists, the last such run's `sha`; when
  there is none, 60 days back on `main`. Entries whose `sha` is not on `main`
  (written before this rule) are skipped. A spec can have been red for longer than any window, so
  `attribute.py` widens on its own to the spec file's last change when nothing
  inside the window scores.
- A run records itself exactly once, after `summarize.py`, whatever the result.
  An aborted run (no `report.json`) records `infra` with the reason; a run the
  global timeout or a crash cut short records `incomplete`.
- Only `green` and `red` mean the SHA was tested. `should-run.sh` runs the same
  SHA again when the last entry is `infra` or `incomplete`, so a transient
  failure never leaves a commit uncovered.
- A `red` run whose serial groups stopped at a failure has `totals.not_run`
  tests it never reached. It still counts as tested on purpose: the same SHA
  would stop at the same failure and skip the same tests, so a retry buys
  nothing. Those tests run again once the blocking failure is fixed, and the
  report lists each under the test that blocked it.
- Flake threshold: a test flaky in 3 of the last 5 recorded runs is an issue
  candidate (`ledger.sh flakes`).
- A dry run (`E2E_LEDGER_PUSH=0`) never pushes. It commits onto the local ref
  `refs/e2e-routine/ledger-dry-run` (starting from the remote ledger the
  first time), and every `ledger.sh` read prefers that ref while
  `E2E_LEDGER_PUSH=0`, so the dry run sees its own entries. Delete the ref
  with `git update-ref -d refs/e2e-routine/ledger-dry-run` to start over.
- Nothing in the ledger is secret; the branch is public like the repository.
  Screenshots and traces stay in the run directory; a screenshot leaves it
  only as an upload to the Linear issue it is evidence for, shown inline in
  the description or comment.
