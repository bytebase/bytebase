# Linear issue for a PRODUCT regression found by the e2e routine

Team `BOT`, parent = the umbrella issue from `E2E_ROUTINE_UMBRELLA`, label
`Bug`, priority per `triage.md`, assignee from `attribute.py` (or none).
Search first: `list_issues team=BOT query=<spec title words>` and the open
children of the umbrella; a matching open issue gets a comment, not a twin.

Title: the symptom in the user's terms, under 100 characters, not the test
name. Example: `SQL Editor: "Request export" button missing from the result pane for a developer with query access`.

Body (Markdown, literal newlines):

```markdown
## What the user sees

Two to four sentences: the screen, the action, what appears (quote the UI text), what the
user cannot do as a result.

## Expected

What the product promised, and the basis: the spec's assertion, the docs, the prior behavior
on `<last green sha>`, or an invariant (permission, masking, gate).

## Evidence

- Spec: `frontend/tests/e2e/<area>/<file>.spec.ts:<line>` "<test title>", red on every attempt of run <run-id>, red in isolation (`--repeat-each=2`)
- Error: `<first line of the Playwright error>`
- Screenshot and aria snapshot attached (`test-failed-1.png`, `error-context.md`)
- Server log: <the 4xx/5xx or panic line, or "clean">

## Since

Candidate commits between `<last green sha>` and `<head sha>` (score from attribute.py):

- `<sha>` <date> <author> #<pr> <subject> — <why: touches the file / adds the text>

## Reproduce

    bash .claude/skills/e2e-routine/scripts/build.sh
    BYTEBASE_E2E_LICENSE=... pnpm --dir frontend exec playwright test <area>/<file>.spec.ts -g "<test title>" --reporter=list

## Routine

Run <run-id> on `<head sha>`; report: <link to the umbrella comment or the session>.
```

Comment on an existing issue (same mechanism already filed):

```markdown
Seen again by the e2e routine: run <run-id> on `<head sha>` (<date>), `<spec>:<line>` "<title>", error `<first line>`. Candidates since `<last green>`: <sha> #<pr>.
```
