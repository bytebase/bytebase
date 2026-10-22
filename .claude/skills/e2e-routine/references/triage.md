# Triage: from a red test to a decision

A red test is a signal, not a finding. The routine turns each one into exactly
one class with evidence, and each class has one action and one owner. Work the
table top to bottom; the first row whose signal and verification both hold wins.

## Classes

| Class | Signal (from summary.json) | Verification that must hold | Action | Owner |
|---|---|---|---|---|
| INFRA | `infra` set: no report, 0 tests ran, or most failures carry a boot/connection signature | `server.log` or `playwright.log` shows the boot, license, browser or psql error | No product issue. Record the ledger entry as `infra`; say what failed and the fix (environment, allowlist, setup script). Two consecutive infra runs: one BOT issue for the framework | framework owner |
| INCOMPLETE | `incomplete` set: the global timeout or a crash ended the run with tests never reached (nonzero exit, no failed test) | `playwright.log` ends with the timeout line or an abort; `summary.md` names how many tests never ran | Not green. Record `incomplete`; the next run retries the same SHA. Report the wall-clock split (build, suite) so the cap or a hung spec can be fixed; a hung spec that recurs becomes a BOT issue | framework owner |
| LOCK-FLIPPED | `lock_flipped`: a `test.fail()` lock passed | The describe title names an issue; that issue is Done, or its fix PR is in `since..main` | Adaptation PR removes `test.fail()`; comment on the issue when filing is enabled | none |
| FLAKE | `flaky`: failed, then passed on a retry | nothing more for this run | Append to `flakes.json` (ledger.sh does it). Issue when the same test is flaky in 3 of the last 5 runs, with the five error lines | framework owner, unless the error names a product race (then PRODUCT) |
| TEST-ISOLATION | `failed`, but passes when its file runs alone (`--repeat-each=2`) | the isolated rerun is green | Adaptation PR when the leftover state is obvious (fixture not idempotent, order dependence); else a BOT issue for the framework | framework owner |
| SPEC-ROT | `failed`, red in isolation | a commit in `since..main` changed the asserted text, locator, route or flow **on purpose**: its PR title/body describes the new behavior, and the new screen is coherent (no error, no dead end, no permission or masking contradiction) | Adaptation PR: update the spec to the new behavior, citing the PR in the commit message. No Linear issue | the product PR's author reviews the adaptation |
| PRODUCT | `failed`, red in isolation | no intentional change explains it, **or** the new behavior breaks an invariant (see below) | Linear BOT issue, or a proposal when filing is disabled. In the adaptation PR, lock it with `test.describe('… (BUG BOT-n)')` + `test.fail()` only once the issue exists | culprit PR author via owners.json; else area default; else unassigned with candidates listed |

Invariants whose violation is PRODUCT even when a PR "intended" the change:
a user without the permission can act, or one with it cannot; masked data shows,
or unmasked data hides; an approval, review or plan-check gate is skipped; a
page shows an error, an empty state in place of data, or an action with no
effect; export or copy policy is not enforced; the workspace or project cannot
be reached after sign-in.

## Verification procedure for a `failed` entry

1. **Isolate.** `run-suite.sh start <run-id>-iso-N -- <spec file> --retries=0 --repeat-each=2`, then `wait`. Green here means TEST-ISOLATION.
2. **Read the evidence, not the title.** In `test-results/<test dir>/`: `error-context.md` is the aria snapshot of the page at failure (the real DOM); `test-failed-1.png`; `trace.zip` when a retry ran. In `playwright.log`, the attempt's error with the locator and the expected vs received text. In `server.log`, 4xx/5xx or panics around the timestamp.
3. **Pin the assertion.** Open the spec at the failing line. Write down the exact UI text or locator and what the test expected.
4. **Find the component.** `attribute.py --text "<quoted text>"` resolves the text through `frontend/src/locales/en*.json` to its key and the files that use it, and lists the commits in `since..origin/main` that touched them or the text. Read the top candidate's PR (`gh pr view <n> --json title,body`).
5. **Decide.** Apply the table. A decision cites: the attempt error line, the evidence file, the spec line, and the commit (SPEC-ROT, PRODUCT) or the isolated rerun (TEST-ISOLATION).
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
remove a `test.fail()` whose bug is fixed; add a `test.fail()` lock that names
an issue.

May not: delete a test; mark a test skipped; widen an assertion so that both
the old and the new behavior pass; change framework files to paper over a
product error; change anything outside `frontend/tests/e2e/`. When the right
change is unclear, classify PRODUCT and leave the spec red.
