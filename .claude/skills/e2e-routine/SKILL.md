---
name: e2e-routine
description: >-
  Nightly regression routine for the Bytebase Playwright e2e suite: decide
  whether main moved, build the frontend-embedded binary, run the licensed
  suite on a disposable server, classify every red test (product regression,
  spec rot, flake, test isolation, infra, flipped lock), attribute an owner,
  file or propose Linear BOT issues, open a test-only adaptation PR, and record
  the run ledger. Use it as the prompt target of the cloud scheduled task, or
  locally with /e2e-routine from a worktree. Not for exploratory QA or writing
  new specs (that is bytebase-e2e-testing) and not for verifying one change
  (that is verify).
---

# E2E regression routine

The suite under `frontend/tests/e2e/` (about 220 tests in 40 files, serial,
enterprise features, a disposable server per run) runs in no CI workflow. This
routine is where it runs. Its product is not a pass count: it is one decision
per red test, with evidence, an owner and an action, plus a ledger entry that
lets the next run know what was covered.

Scripts live in `scripts/`; the decision table is `references/triage.md`; the
issue body is `references/issue-template.md`; the ledger format is
`references/ledger.md`; owner routing is `owners.json`.

## Invariants

- **Pushes**: never to `main`. Only `claude/e2e-ledger` (through `ledger.sh`) and
  one `claude/e2e-adapt-<YYYY-MM-DD>` branch per run.
- **License**: never print it, never `cat` or grep-print `frontend/.env.dev`.
  `lib.sh` passes it into the detached run only.
- **Processes**: never kill a process you did not start. The harness owns its
  server (PID file, port 18234 with fallback). A port conflict is reported, not
  resolved by killing.
- **Linear**: team `BOT` only, parent = the issue in `E2E_ROUTINE_UMBRELLA`.
  Create issues and comments only when `E2E_ROUTINE_FILE_ISSUES=1`; otherwise
  write proposals into the report. Never edit, retitle, close, reassign or
  relabel an existing issue. Search before creating; one issue per mechanism.
- **Build**: `-tags embed_frontend` without `release`, so the dev-signed license
  validates.
- **Evidence before claims**: every class cites the attempt error line, the
  evidence file and, for SPEC-ROT or PRODUCT, a commit. A gate without pasted
  evidence is not passed.
- **Adaptations** change only `frontend/tests/e2e/`, never delete or skip a test,
  and never widen an assertion so that old and new behavior both pass.

## Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `E2E_ROUTINE_FILE_ISSUES` | `0` | `1` files into Linear; `0` writes proposals only |
| `E2E_ROUTINE_UMBRELLA` | none | the BOT umbrella issue (parent of filed issues, target of the run comment) |
| `E2E_ROUTINE_SLACK_CHANNEL` | none | when set and the Slack connector is attached, post the report summary there |
| `E2E_RESULTS_ROOT` | `/tmp/e2e-routine` | run directories |
| `E2E_LEDGER_BRANCH` | `claude/e2e-ledger` | ledger branch |
| `E2E_LEDGER_PUSH` | `1` | `0` commits the ledger entry locally without pushing (local dry runs) |
| `E2E_GLOBAL_TIMEOUT_MS` | `7200000` | wall-clock cap for one suite run; tests not reached count as skipped and the run is reported as such |
| `BYTEBASE_E2E_LICENSE` | from `frontend/.env.dev` | override only when a different license is wanted |
| `FORCE` | `0` | `FORCE=1 should-run.sh` runs even when main did not move |
| `BASH_DEFAULT_TIMEOUT_MS`, `BASH_MAX_TIMEOUT_MS` | tool defaults | set on the cloud environment (10 min / 60 min) so `wait` can block |

## Phase 0: Preflight

```bash
bash .claude/skills/e2e-routine/scripts/preflight.sh
```

Paste the table. Any `FAIL` line ends the run as INFRA (Phase 6 still records
it). A `WARN` on a port means another server is listening; the harness picks the
next free range, so continue, and say so in the report.

## Phase 1: Decide

```bash
bash .claude/skills/e2e-routine/scripts/should-run.sh; echo "exit $?"
```

Exit 3 (`NO_CHANGE`) or 4 (`NOT_RELEVANT`): do not build, do not record. Post
one line ("e2e routine: nothing to do, main at `<sha7>`, <n> docs-only
commits") as a comment on the umbrella when filing is enabled, otherwise as the
final message, and stop. Exit 0: keep `head_sha` and `last_sha` from the JSON.

Set `LAST_GREEN`: the newest ledger run with `"result": "green"`
(`ledger.sh show` for the last run; `git show origin/claude/e2e-ledger:runs/<file>`
for older ones). Fall back per `references/ledger.md`.

## Phase 2: Build

```bash
RUN_ID=$(date -u +%Y%m%dT%H%M%SZ)
bash .claude/skills/e2e-routine/scripts/build.sh /tmp/e2e-routine/$RUN_ID
cat /tmp/e2e-routine/$RUN_ID/build-timing.txt
```

Build from `origin/main` checked out at `head_sha` (`git checkout -q
<head_sha>` in the clone is fine; the ledger records the SHA). A build failure
is INFRA unless the failing step is the Go or frontend compile of a product
file, which is PRODUCT with the compile error as evidence (main is protected
by CI, so this is rare and worth a Linear issue regardless of filing mode).

## Phase 3: Run

```bash
DIR=$(bash .claude/skills/e2e-routine/scripts/run-suite.sh start $RUN_ID)
bash .claude/skills/e2e-routine/scripts/run-suite.sh wait "$DIR" 120
python3 .claude/skills/e2e-routine/scripts/summarize.py "$DIR"
```

`start` detaches the suite (its own session, so a tool timeout cannot kill it)
and prints the run directory. Call `wait` with the Bash tool's maximum timeout;
if the tool returns before the suite ends, call `wait` again: it resumes. Do not
run anything else that boots a Bytebase while it runs. The suite takes roughly
40 to 60 minutes.

`summarize.py` writes `summary.json` and `summary.md` (buckets: `failed`,
`flaky`, `lock_flipped`, `lock_holding`, `skipped`, `infra`). Paste
`summary.md`. Exit 2 means no `report.json`: INFRA, go to Phase 6.

## Phase 4: Classify and verify

Work `references/triage.md` for every entry of `failed`, `flaky` and
`lock_flipped`. For each `failed` entry:

1. Isolate: `run-suite.sh start $RUN_ID-iso-<n> -- <spec file> --retries=0 --repeat-each=2`, then `wait`.
2. Read `test-results/<test>/error-context.md`, the screenshot, the error in
   `playwright.log`, and `server.log` around the timestamp.
3. Pin the spec line and the quoted UI text or locator.
4. `python3 .claude/skills/e2e-routine/scripts/attribute.py --spec <file> --since $LAST_GREEN --text "<quoted text>"`,
   then `gh pr view <n> --json title,body` for the top candidate.
5. Decide the class, the owner and the action; write the row.

Decision table (one row per non-green test):

| Test | Class | Evidence | Culprit / owner | Action |
|---|---|---|---|---|

Flaky: compare with `ledger.sh flakes`; three of the last five runs makes it an
issue candidate. Lock flipped: read the issue id from the describe title and
check it in Linear (`get_issue`).

### Phase 4 gate

The decision table, with every `failed` row showing the isolated rerun result
and a commit or the words "no commit explains it", and every PRODUCT row naming
the violated expectation.

## Phase 5: Act

**PRODUCT**: search `BOT` (`list_issues team=BOT query=...`) and the umbrella's
open children. Existing open issue: comment (template at the end of
`references/issue-template.md`). Otherwise create per the template: parent
umbrella, label `Bug`, priority per `triage.md`, assignee = attributed owner
(none when unassigned, with candidates in the body), attach
`test-failed-1.png` and `error-context.md`
(`prepare_attachment_upload` + `create_attachment_from_upload`). With filing
disabled, put the full issue text under "Proposed issues" in the report
instead.

**SPEC-ROT, LOCK-FLIPPED, TEST-ISOLATION, PRODUCT locks**: one branch
`claude/e2e-adapt-<YYYY-MM-DD>` from `head_sha`, changes only under
`frontend/tests/e2e/`, one commit per adapted spec whose message names the
product PR (`test(e2e): follow #21489 export format select`). Rerun each changed
file (`run-suite.sh start $RUN_ID-adapt -- <files>`) and paste the green list
output. Push and open the PR with `gh pr create --title "test(e2e): adapt to main <date>"`;
the body lists each change with its product PR and asks that PR's author for
review. Never merge it. A PRODUCT lock (`test.describe('… (BUG BOT-n)')` +
`test.fail()`) goes in only once the issue exists; with filing disabled, leave
the spec red and say so.

**INFRA, FLAKE below threshold**: report only.

### Phase 5 gate

Issue ids or proposal headings per PRODUCT row; the adaptation PR URL with its
rerun output; or "no action" per class.

## Phase 6: Record and report

```bash
bash .claude/skills/e2e-routine/scripts/ledger.sh record "$DIR" "<one line: classes, PR, issues>"
bash .claude/skills/e2e-routine/scripts/ledger.sh show
```

Then write the report, in this order, and post it as a comment on the umbrella
issue when filing is enabled (otherwise it is the session's final message; with
`E2E_ROUTINE_SLACK_CHANNEL` set, post the first block to Slack as well):

1. `e2e routine <date>: <green|red|infra> on main <sha7>` and the counts line from `summary.md`, the build time, the suite time.
2. The decision table from Phase 4 with links (issue, PR, run directory names).
3. Proposed issues (filing disabled) in full.
4. Not covered: specs skipped or blocked, checks the run could not make, flakes under threshold.
5. Infra notes and what to change in the environment, if anything.

A run that stops early still does Phase 6 with what it has; the ledger never
gets a `green` entry without a `report.json` behind it.

## Local use

From a worktree, with the same phases: `FORCE=1` on Phase 1 when main did not
move, `E2E_ROUTINE_FILE_ISSUES` left at `0`, proposals listed for the user to
approve item by item. Build and run from the worktree only; do not touch the
main checkout.
