# Triage: from a red test to a decision

A red test is a signal, not a finding. The routine turns each one into exactly
one class with evidence, and each class has one action and one owner. Work the
table top to bottom; the first row whose signal and verification both hold wins.

## Buckets: how `summarize.py` sorts a test

A test's evidence is its anomalous attempts: those whose status differs from
the test's `expectedStatus` (an ordinary test expects `passed`, a `test.fail()`
lock expects `failed`; a timeout or interruption is never the expected
status). The bucket follows from which attempts were anomalous:

| Expected | Attempts (3 at most) | Bucket |
|---|---|---|
| passed | all passed | passed |
| passed | every attempt anomalous | `failed` |
| passed | anomalous, then a pass | `flaky` |
| failed (lock) | all failed | `lock_holding` |
| failed (lock) | all passed | `lock_flipped` |
| failed (lock) | at least one pass and at least one failure | `lock_unstable` |
| failed (lock) | no pass, at least one timeout or interruption | `failed` if no attempt failed as locked, else `flaky`: the test now hangs, which is not the locked bug |

The error and the screenshots come from the latest anomalous attempt. An
unexpected pass has no error message, so its error reads "Expected to fail,
but passed".

## Classes

| Class | Signal (from summary.json) | Verification that must hold | Action | Owner |
|---|---|---|---|---|
| FOCUSED | a `failed` row "a focused test (.only) is committed": the runner passes `--forbid-only`, so Playwright ran nothing. Without the flag only the focused test would run, and pass | the spec line in the row holds `test.only` or `describe.only`; `git log -S '.only' -- <spec>` names the commit | Adaptation PR removes the `.only`, citing that commit; the next run covers the suite. No Linear issue | the commit's author reviews the adaptation |
| SERVER-CRASH | a `failed` row "Bytebase server crashed": the client lost the server (what INFRA calls `server-lost` or boot) and `server.log` holds a Go crash, a `panic:` or `fatal error:` line followed by a goroutine dump. The tests that died with it are in `lost_to_crash`, not triaged | the crash line and its first backend frame (`frame`) in `server.log`; `first_test` is the first test that lost the server, so the crash happened during it or the test before | PRODUCT issue, priority Urgent (every later test is blocked): `attribute.py --spec <frame file> --since "$(bash .claude/skills/e2e-routine/scripts/ledger.sh last-green)"` for the culprit; reproduce with `first_test` alone | culprit PR author via the frame's file; else the area owner of that file |
| INFRA | `infra` set: preflight or build failed (`--pre-run-failure`), no report, 0 tests ran with a boot/license/browser/psql signature, or the failed tests are environment casualties: at least 5 failures and 80% of them carry an environment signature (connection or boot is `server-lost`, unless `server.log` shows the server crashed: then SERVER-CRASH; browser, psql, toolchain or license keeps its own name), or the setup project failed with one and nothing else failed. The casualties move to `lost_to_infra` and are not triaged; failures without a signature stay in `failed` and are triaged row by row. A global-setup error with no such signature is a `failed` row named "global setup", and an error outside any test after tests ran (global teardown, a worker crash) is a `failed` row named "suite-level error"; both are triaged and attributed like any test. A nonzero Playwright exit that nothing else explains is infra (`unexplained-exit`): a run is green only when Playwright says so | `server.log` or `playwright.log` shows the boot, license, browser or psql error | No product issue. Record the ledger entry as `infra`; say what failed and the fix (environment, allowlist, setup script). Two consecutive infra runs: one BOT issue for the framework | framework owner |
| INCOMPLETE | `incomplete` set: the global timeout or a crash ended the run with tests never reached (nonzero exit, no failed test) | `playwright.log` ends with the timeout line or an abort; `summary.md` names how many tests never ran | Not green. Record `incomplete`; the next run retries the same SHA. Report the wall-clock split (build, suite) so the cap or a hung spec can be fixed; a hung spec that recurs becomes a BOT issue | framework owner |
| LOCK-UNSTABLE | `lock_unstable`: a `test.fail()` lock passed on some attempts and failed on others | rerun the one test alone, `-g "<title>" --retries=0 --repeat-each=10`, and count the passes; open the screenshot of a passing attempt and the trace of a failing one | The locked bug is intermittent, a race. Never remove the lock and never file a twin: comment on the locked issue with the pass rate (or propose the comment when filing is disabled), and reopen-worthy evidence goes there. If that issue is Done, say so: its fix is incomplete | the locked issue's assignee |
| LOCK-FLIPPED | `lock_flipped`: a `test.fail()` lock passed on every attempt | The describe title names an issue; that issue is Done, or its fix PR is in `since..main` | Adaptation PR removes `test.fail()`; comment on the issue when filing is enabled | none |
| FLAKE | `flaky`: failed, then passed on a retry | the isolated rerun (`--retries=0 --repeat-each=2`) is green. A `flaky` row that is red in isolation is not a flake: work it as a `failed` row (SPEC-ROT or PRODUCT). An error with `strict mode violation` is a locator that matches several elements, a spec defect and never timing: SPEC-ROT | Append to `flakes.json` (ledger.sh does it). Issue when the same test is flaky in 3 of the last 5 runs, with the five error lines | framework owner, unless the error names a product race (then PRODUCT) |
| TEST-ISOLATION | `failed`, but passes when its file runs alone (`--repeat-each=2`) | the isolated rerun is green | Adaptation PR when the leftover state is obvious (fixture not idempotent, order dependence); else a BOT issue for the framework | framework owner |
| SPEC-ROT | `failed`, red in isolation | a commit in `since..main` changed the asserted text, locator, route or flow **on purpose**: its PR title/body describes the new behavior, and the new screen is coherent (no error, no dead end, no permission or masking contradiction) | Adaptation PR: update the spec to the new behavior, citing the PR in the commit message. No Linear issue | the product PR's author reviews the adaptation |
| PRODUCT | `failed`, red in isolation | no intentional change explains it, **or** the new behavior breaks an invariant (see below) | Linear BOT issue, or a proposal when filing is disabled. In the adaptation PR, lock it with `test.describe('… (BUG BOT-n)')` + `test.fail()` only once the issue exists | culprit PR author via owners.json; else area default; else unassigned with candidates listed |

Invariants whose violation is PRODUCT even when a PR "intended" the change:
a user without the permission can act, or one with it cannot; masked data shows,
or unmasked data hides; an approval, review or plan-check gate is skipped; a
page shows an error, an empty state in place of data, or an action with no
effect; export or copy policy is not enforced; the workspace or project cannot
be reached after sign-in.

## Verification procedure for a `failed`, `flaky` or `lock_unstable` entry

1. **Isolate.** `run-suite.sh start <run-id>-iso-N -- <spec file> --retries=0 --repeat-each=2`, then `wait`. For a `failed` row, green here means TEST-ISOLATION; for a `flaky` row, red here means it is not a flake (see FLAKE). A `lock_unstable` row reruns the one test ten times instead (LOCK-UNSTABLE).
2. **Read the evidence, not the title.** The entry's `evidence` in `summary.json` names the files of the anomalous attempt it reports (see "Buckets"): `screenshots`, one per open page (open them and look: they are what a user would have seen), `error_context`, the aria snapshot of the page at failure (the real DOM), and `trace` when a retry ran. In `playwright.log`, the attempt's error with the locator and the expected vs received text. In `server.log`, 4xx/5xx or panics around the timestamp.
3. **Pin the assertion.** Open the spec at the failing line. Write down the exact UI text or locator and what the test expected.
4. **Find the component.** `attribute.py --text "<quoted text>"` resolves the text through `frontend/src/locales/en*.json` to its key and the files that use it, and lists the commits in `since..HEAD` that touched them or the text. It also lists commits to the spec file itself and to `frontend/tests/e2e/framework/`, marked as test changes: a test commit that broke its own test is SPEC-ROT, adapted and reviewed by its author. It exits 2 when `--since` or `--head` is not a commit here, rather than reporting an empty range. Read the top candidate's PR (`gh pr view <n> --json title,body`).
5. **Decide.** Apply the table. A decision cites: the attempt error line, the screenshot and what it shows, the spec line, and the commit (SPEC-ROT, PRODUCT) or the isolated rerun (TEST-ISOLATION).
6. **Weigh before filing.** Validity (does the evidence show it), frequency (every run, or one race), return (what a fix buys). A valid, rare, cosmetic failure is still recorded in the report with the decision "not worth an issue" and why; it is never silently dropped.

## Priority for PRODUCT issues

| Priority | When |
|---|---|
| Urgent (1) | smoke floor: sign-in, the permission or approval gate, masking or export enforcement, creating and rolling out a plan |
| High (2) | a primary flow of an area is blocked or shows wrong data |
| Medium (3) | a secondary flow, a wrong label with a workaround, a layout defect that hides an action |
| Low (4) | cosmetic, no user action affected |

## What an adaptation may and may not do

May: change a locator or expected text to the new, intended value; follow a
moved route; add a step the new flow requires; make a fixture idempotent;
remove a `test.fail()` whose bug is fixed; remove a committed `.only`; add a `test.fail()` lock that names
an issue.

May not: delete a test; mark a test skipped; widen an assertion so that both
the old and the new behavior pass; change framework files to paper over a
product error; change anything outside `frontend/tests/e2e/`. When the right
change is unclear, classify PRODUCT and leave the spec red.
