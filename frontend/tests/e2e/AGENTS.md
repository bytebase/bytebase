# E2E Tests — Contributor & AI Agent Guide

Conventions for writing and maintaining Bytebase e2e tests. Follow these rules unless you have a strong, documented reason not to.

## Core Principles

### 1. Create your own test data, don't discover it

**Do:** Create a dedicated schema/table/rows for your test and drop them in `afterAll`.

```typescript
// In beforeAll — create what you need
execSql(env, dbName, `CREATE SCHEMA my_feature_test`);
execSql(env, dbName, `CREATE TABLE my_feature_test.t (id INT PRIMARY KEY, col TEXT)`);
execSql(env, dbName, `INSERT INTO my_feature_test.t VALUES (1, 'KnownValue')`);

// In afterAll — clean up
execSql(env, dbName, `DROP SCHEMA IF EXISTS my_feature_test CASCADE`);
```

**Don't:** Query `information_schema` or scan the sample data to find something usable. Discovery-based tests are fragile — they fail when sample data changes, when tables are empty, or when pre-existing masking hides the values you need.

**Why:** A test that owns its fixtures is deterministic. You know exactly what values exist, what's masked, what's not, and what state is left when the test completes. `masking-exemption.spec.ts:createMaskingTestData` is the canonical example.

### 2. Share a single browser across all tests in a file

**Do:** Create one `BrowserContext` and `Page` in the file-level `test.beforeAll`, and reuse them across every test in the file.

```typescript
let sharedContext: BrowserContext;
let page: Page;

test.beforeAll(async ({ browser }) => {
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test("my test", async () => {
  // use the shared `page` — no { page } destructuring
  await page.goto(...);
});
```

**Don't:** Use Playwright's default per-test `{ page }` fixture. It opens and closes a browser context for every test, which:
- Costs real time (especially on CI)
- Triggers NVirtualList first-render timing bugs
- Makes tests slower for no isolation benefit (we run `workers: 1` anyway)

**Side effects to clean up manually** (since we lose per-test isolation):
- Reset viewport in `afterAll` if you changed it mid-suite
- Clear URL/tab state explicitly if it matters (usually via `gotoWithDb` etc.)
- Reset policy state via API (`revokeAllExemptions`) in `beforeEach` when tests depend on clean state

### 3. API for setup, browser for verification

Bytebase's v1 REST API is your tool for fast, deterministic state setup. Use the browser only to verify what a user would see.

**Do:**
- Grant/revoke policies via API, then render them in the browser to assert the UI reflects the state.
- Create test data via `psql` (Unix socket), configure catalog via API, run the test UI flow in the browser.

**Don't:**
- Click through the UI to set up preconditions when an API call would do it.
- Screenshot-compare for logic assertions.

### 4. Use primary keys for masking verification queries

When verifying masked/unmasked data in the SQL editor, query by the row's primary key, not by `LIMIT n`. Ordering is not deterministic without `ORDER BY`, and `LIMIT` may not include your known value.

```typescript
const sql = `SELECT "${col}" FROM "${schema}"."${table}" WHERE "${pkColumn}" = '${pkValue}'`;
```

### 5. Avoid `waitForTimeout` for arbitrary delays

**Do:** Wait for specific conditions via locator auto-wait, `waitForResponse`, `waitForURL`, or `expect(...).toBeVisible()`.

**Don't:** Sprinkle `await page.waitForTimeout(500)` everywhere. Every arbitrary sleep is a flakiness source on slow CI.

*(Existing tests have technical debt here — new tests should not add to it.)*

### 6. Cross-platform keyboard shortcuts

Use `ControlOrMeta+a` (portable), not `Meta+a` (Mac-only) or `Control+a` (Linux/Windows-only).

### 7. Prefer `data-testid` over class-based selectors

Tailwind class substrings like `[class*='border border-gray']` break on any CSS refactor. If you need a new locator, add a `data-testid` attribute to the component. Existing class-based selectors are technical debt.

## Directory Layout

```
frontend/tests/e2e/
├── README.md              — human-facing docs (how to run tests)
├── AGENTS.md              — this file (conventions)
├── framework/             — shared infrastructure (don't add feature code here)
└── <feature-name>/        — one directory per feature test suite
    ├── *.spec.ts          — test files
    └── *.page.ts          — page object models (feature-specific)
```

## File Responsibilities

| File | Responsibility |
|------|----------------|
| `framework/api-client.ts` | Bytebase v1 REST API wrapper, token refresh on 401 |
| `framework/env.ts` | `TestEnv` interface, load/save via `.e2e-env.json` |
| `framework/mode-start-new-bytebase.ts` | Disposable server lifecycle + port reconciliation |
| `framework/global-setup.ts` | Starts server before any test runs |
| `framework/global-teardown.ts` | Stops server after all tests finish |
| `framework/setup-project.ts` | Auth + instance/database discovery, writes env + auth state |

## Adding a New Feature Test Suite

1. **Create directory**: `tests/e2e/<feature-name>/`
2. **Write page objects** in `<feature>.page.ts` (one class per UI surface). Accept `baseURL` in the constructor so pages can navigate via absolute URLs.
3. **Write spec file** `<feature>.spec.ts`:
   - Import `loadTestEnv` from `../framework/env`
   - Declare the shared `page` / `sharedContext` at module level
   - In file-level `test.beforeAll`: `loadTestEnv()`, login, create test data, open the shared browser
   - In `test.afterAll`: close browser, clean up test data via the same code path that created it
4. **Use the canonical example** — read `frontend/tests/e2e/masking-exemption/masking-exemption.spec.ts` as a reference.

## Extending the API Client

- Add methods to `BytebaseApiClient` in `framework/api-client.ts`
- Use `this.request<T>()` with typed responses
- Include `pageSize=100` on list endpoints
- Only add methods actually used by a test — no speculative API coverage

## Running DDL / DML (for test data setup)

The Bytebase query API is **read-only**. For DDL/DML, use the `execSql` helper (see `masking-exemption.spec.ts`) which shells out to `psql` via Unix socket to the sample Postgres instance.

**Port layout**: the disposable Bytebase server on `PORT` starts sample Postgres at:
- `PORT + 3` → `test-sample-instance` (hr_test)
- `PORT + 4` → `prod-sample-instance` (hr_prod)

Get the correct port from `getInstance(env.instance)` rather than hardcoding the offset.

## Known Constraints

- **Sample-data bootstrap**: tests run against data provisioned by `SetupSample` (called from `globalSetup`). The two sample Postgres instances (`test-sample-instance`, `prod-sample-instance`) come up on `PORT+3` / `PORT+4`.
- **License required**: the suite exercises enterprise-only features (masking, JIT, approval workflow, query-data-policy, database groups) and does NOT run on the free plan. Set `BYTEBASE_E2E_LICENSE` to a license JWT signed by Bytebase's license key (ask Bytebase ops for a dev/test license; not stored in this repo); `globalSetup` installs it via `PATCH /v1/subscription/license`. If the env var is absent the bootstrap throws and the whole run stops — there is no per-spec `test.skip` fallback.
- **Serial execution**: `fullyParallel: false` + `workers: 1`. Tests within and across files are sequential.
- **`psql` dependency**: must be on PATH for DDL/DML setup.
- **Unix-like OS only**: the sample Postgres uses Unix sockets in `/tmp`.
- **Admin credentials**: hardcoded `demo@example.com` / `12345678`. The first user created via `/v1/auth/signup` becomes workspace admin, so e2e signs up this fixture.
- **DBA fixture**: `dba1@example.com` / `12345678` is created during `globalSetup` and granted `roles/workspaceDBA`. Used as the second approver by plan-detail approval specs. If a spec needs additional users (developer, QA, etc.), provision them the same way in `mode-start-new-bytebase.ts` — don't assume they exist.

---

<!-- Merged from the former PRINCIPLES.md (QA finding/writing doctrine). -->

## QA Principles — Field Manual

Doctrine for pre-release regression QA. Distilled from gaps between what methodical
walks find and what humans file. Two parts:

- **Part I** — principles for *finding* bugs (during exploration)
- **Part II** — principles for *writing* tests (once a bug is reproduced)


---

### Who we are

> We are QA engineers in pre-release regression. Our defining trait is **curiosity
> tempered by logic**. We touch every button to see what happens; we also know what
> *should* happen and why — by deducing from sibling screens, from commit history,
> from the user's mental model. We trust what users see over what the codebase
> asserts. We are skeptical: a button that renders is not a button that works.
> We are persistent: a bug noticed once but not yet reproduced is not yet found.
> Everything below follows from this.

---

## Part I — Principles for finding bugs

Ranked from most to least important. Grouped by tier for skimming, but the order
within and across tiers reflects impact.

### Tier 1 — Mindset (foundational)

These are not techniques. They are who you have to *be* before any technique helps.

#### 1. Be curious — touch every clickable thing

Open every dropdown, click every checkbox, type into every input, right-click
everywhere. Explore paths the docs don't list. Curiosity is the source of finding;
without it, no method produces anything. A page is "checked" only after you have
probed every interactive element on it.

**In practice:** enumerate every `button`, `input`, `checkbox`, `dropdown`,
`menuitem`, and right-click target on the screen before declaring it covered.

#### 2. Be logical — reason about expected behavior

Curiosity without logic is noise. Before flagging anything, ask: *what should this
screen do, and why?* Deduce expected behavior from related views, from how comparable
products work, from the underlying data model, from commit history. Cross-reference:
if X holds on screen A, screen B's behavior must follow. *New* is not *broken*;
*weird* is not always *bug* — but verify with reasoning, not assumption.

**In practice:** when a behavior looks off, write down "I expected X because Y"
before declaring it a defect. If your *why* is "it just feels wrong," you haven't
reasoned yet.

#### 3. Be the user — narrate the experience

Stop describing what the codebase does. Describe what a person sees and feels.
"I clicked Run and waited five seconds for nothing to happen" surfaces bugs that
"Run button has class X" never will. Users notice in 30 seconds what exhaustive
automation misses in an hour.

**In practice:** after each interaction, ask "what would a customer say happened
just now?" If the answer is interesting, file it.

### Tier 2 — Coverage (what to test)

Once you have the right mindset, these principles direct *where* to look.

#### 4. Stress what looks static

Static screens pass. Bugs hide under load. For every input field: try empty,
near-empty, full-to-the-limit, beyond-the-limit, unicode, emoji, very long strings,
special characters. For every button: click rapidly, click many times in succession,
click during loading, click while another action is in flight. For every list or
table: empty state, one item, hundreds, thousands. Bugs that are invisible at rest
become loud under pressure.

**In practice:** every CUJ has a "stressed" variant. Add it deliberately, don't
wait to discover it.

**Enumerate, don't just stress.** Before declaring an area covered, list every
clickable leaf (button, icon, toggle, menuitem) in that area and click each one
once. Coverage holes — like a Schema-diagram tab nobody opened, or a Copy button
nobody clicked in admin mode — come from following CUJs without enumeration.

#### 5. Test sequences, not just isolated actions

Bugs cluster at the boundaries between actions, not inside them. Build interaction
graphs: *switch context → return → re-engage*. *Open dialog → cancel → reopen*.
*Trigger A → trigger B in flight → trigger A again*. The third action in a chain
is where state corruption lives. A test of "X works" and "Y works" does not prove
that "X then Y" works.

**In practice:** for every pair of major actions, walk A→B and B→A. For every
modal, sidebar, or tab, test the re-engagement path after closing.

**Side-effects across unrelated views.** Open view A → trigger an unrelated action
in B → return to A and assert A is unchanged. Each half is "correct" in isolation —
the bug only exists when both views are alive at the same time. Example: opening
the History sidebar then running a query in the editor silently nuked the history
list. Running and history-search worked fine alone.

#### 6. Vary the data, not just the queries

Renderers break on shapes other than the happy case. The same surface against
different data — NULLs, wide rows, deeply-nested objects, binary, errors, large
strings, special characters, multi-statement, empty result — exercises a different
code path each time. A single happy-path test is a smoke test, not a stress test.

**In practice:** keep a "data variety" pass per result-rendering surface. Each
variant is its own test.

#### 7. Test from real user states, not just admin

Admin bypasses most gates. The interesting bugs live at permission boundaries —
visible only to scoped users. Maintain a fixture user roster (no-membership, viewer,
developer, admin) and run the same tour as each. The bugs that ship are the ones
admins can't see.

**In practice:** every CUJ touching permissions runs at least twice — admin and
non-admin. Helpers for grant/revoke make this cheap.

#### 8. Let the diff drive the work

The bug surface is what changed. `git log` since the previous release ranks risk.
Weight exploration time toward high-delta surfaces; do not spend cycles on code
that has been stable for a year.

**In practice:** read the diff before designing CUJs. Score each CUJ by lines-changed
in its surface area. Walk highest-risk first while you're still fresh.

### Tier 3 — Observation (how to see)

Curiosity and coverage put you in front of the bug. These principles ensure you
*see* it.

#### 9. Look at the screen, not just the DOM

JS evals tell you what's *there*. Screenshots tell you what a user *sees*. They
disagree more often than you'd expect: contrast, overlap, truncation, two-line
expansion of single-line widgets, disabled-looking-but-clickable controls. Most
visual bugs are invisible to DOM-only inspection.

**In practice:** after every meaningful state change, capture a screenshot and
read it like an image, not a log. Describe what you see in plain language before
moving on.

**And read every word.** Form labels, dropdown options, helper text, version
banners, confirm-dialog titles — read them one by one. Duplicate wording (a field
labeled "Expiration" with a "4 hours" dropdown that *also* says "hours" inline),
backwards version banners ("New version 3.17.1 available" while you're on 3.18),
and placebo confirm dialogs ("Confirm to delete this worksheet?" with no body
text) all hide in plain text we breeze past.

#### 10. Record video for transient bugs

Some bugs only exist in motion: a button that blinks on hover, a modal that flashes
and disappears, an overlay that flickers during a race condition, an animation that
stutters. Screenshots miss these entirely. Run a screen recorder for every
exploratory session; review at half speed when something feels off.

**In practice:** keep a video capture running continuously. Mark timestamps when
something feels off; review and trim before filing.

#### 11. Profile network per action

Every user click has a request budget. Anything fanning out to `List*` endpoints on
a simple switch, or repeating identical requests, or firing more than a handful of
calls on a navigation — is suspect. Performance regressions hide from functional
tests.

**In practice:** capture `performance.getEntriesByType('resource')` before and
after each action; diff the count and the URLs.

#### 12. Read commit messages — they name risk surfaces

Each "refactor: migrate X" is a hypothesis to test. Each "fix: cannot Y" is a
region of recent fragility. Commit history is the engineering team's confession
about where the bugs were; it tells you where the next ones probably are. Use it.

**In practice:** before exploring an area, `git log --oneline <prev-release>..HEAD -- <area>`.
Treat each commit as a target.

### Tier 4 — Discipline (how to verify honestly)

Once you think you have a finding, these principles prevent self-deception.

#### 13. Distinguish "wired up" from "works"

Three layers before ✓: render (DOM exists), interactive (click registers a state
change), end-to-end (action produces a user-visible outcome). Do not conflate
them. Many automation tools fire events that the framework's handlers do not
register — the click looks made but nothing happens downstream.

**In practice:** for every control, confirm a state change after the click —
not just that the click was issued.

#### 14. When DOM and screen disagree, trust the screen

The user sees the screen. If your selector reports one thing and the rendered
output shows another, your selector is wrong — not the product. Stop debugging
the product; debug your observation.

**In practice:** when JS state surprises you, screenshot first, then debug the
selector. The image is ground truth.

#### 15. Compare across surfaces — same widget, every page

The same button, the same icon, the same toggle should render the same way on
every page that uses it. The Issues-page Save button should look like the
SQL-Editor Save button should look like the Settings-page Save button — same
size, same color, same hover state, same affordance. Single-surface walks miss
this entire bug class because they only ever see one instance of each widget.

Visual consistency is *the* tell of a well-maintained design system; lapses are
*the* tell of a recent migration that didn't quite finish.

**In practice:** keep a list of widgets that should be identical (Save, Cancel,
the avatar dropdown, status badges, etc.) and screenshot each one on at least
three different surfaces. Diff the images visually; mismatches are bugs.

---

## Part II — Principles for writing tests

Once a bug is reproduced or a CUJ is validated, the next job is locking it in
durable `.spec.ts` code. These conventions cover both bug-lock tests and CUJ
tests.

#### A. Assert outcomes, not implementation

Check the user-observable thing. Tests survive refactors only when they verify
what the user sees, not what the codebase happens to render.

- Bad: `expect(button).toHaveClass('bg-accent')`
- Good: `await expect(page.getByText('Exit admin mode')).toBeVisible()`

#### B. Bug-lock tests must fail on the buggy build

A passing bug-lock test is a broken test. Each must fail with a message that
pinpoints the bug. Use `test.fail()` when the bug is known-unfixed. **Never use
`test.skip()`** — skipping hides regressions instead of catching them.

#### C. Tests own their fixtures, not discover them

Create your own schema, tables, rows in `beforeAll`; drop in `afterAll`. Don't
discover demo data — discovery-based tests fail when seed data drifts.

#### D. Integration sequences > isolated actions

One spec doing `switch tab → click sidebar → switch project → return → click sidebar`
catches state-corruption bugs that three isolated specs would miss. Tests should
read like user stories, not unit tests.

#### E. Visual assertions via screenshot baselines

For visual bugs (icon contrast, overlap, truncation), add
`await expect(page).toHaveScreenshot('export-dialog.png')` baselines. No JS
assertion catches "icons too light." Add explicit baselines per visual surface,
not just on failure.

#### F. Test by role, not just by admin

Build `loginAs('sqlEditorReadUser')` and friends. Same spec, multiple users.
Permission bugs only surface this way. Use API-based role grant/revoke in
`beforeAll`/`afterAll` so tests don't lean on seeded IAM.

#### G. Parameterize data-variety in result tests

One spec, many cases: NULL, JSON, binary, wide-row, error, large-string. Each
a different test case via `test.describe.each`. Closes the gap that single
happy-path tests leave open.

#### H. Network-budget assertions

For dialogs and switches that shouldn't refetch:

```typescript
expect(networkLog.count).toBeLessThanOrEqual(N);
```

Catches performance regressions. Use Playwright's `page.on('request')` per spec.

#### I. One concept per test; name = user behavior

Names become the spec doc. Write them as sentences a PM could read.

- Bad: `'handleClick fires updateCurrentTab'`
- Good: `'clicking 10000-rows preset updates the displayed export limit'`

#### J. Test boundaries, not just happy paths

Every input gets: empty, max, min, special-chars. Most missed bugs are boundary
issues. Empty-state, one-item, hundreds, thousands all deserve their own test
case.

#### K. Test the test before trusting it

Mutate the production code (delete the handler, change the value) and re-run.
If the test still passes, it isn't actually checking the thing.

#### L. Self-contained state

No test depends on another's leftovers. Every spec sets up and tears down its
own DB schema, worksheets, IAM bindings. Independence is the only way tests
survive at scale.

#### M. Assert visual relationships, not absolute thresholds

When a bug is "X looks wrong against Y", phrase the test as a comparison
between X and Y — not as X must equal some specific value. The fix could pick
any reasonable shade, size, or position; your test should pass for every
reasonable choice, not just the one you happened to expect.

- Bad: `expect(maxChannel(block)).toBeLessThan(80)` — pins "dark" to a
  specific RGB threshold; fails if the design system later picks a slightly
  lighter dark, even though the user-visible problem is solved.
- Good: `expect(maxChannelDelta(block, shell)).toBeLessThan(100)` — asserts
  the block harmonizes with its container. Any reasonable theme passes.

This applies anywhere the bug is contextual: a misaligned button (test
relative position to its row, not absolute pixels), a row highlight that
matches the wrong column (test the row container's bg vs the surrounding
rows, not a literal rgb), a tooltip overlapping its trigger (test
bounding-rect intersection, not absolute coordinates), an inconsistent
icon size across surfaces (compare across instances, not against `16px`).

Why this matters: thresholds tied to specific values rot fast — the design
system revises them and the test fails for a reason unrelated to the bug.
Relational assertions capture the user's actual judgment ("does this look
out of place?") and survive any tweak that preserves the relationship. The
failure message is also more useful: "block (rgb 243/244/246) differs from
shell (rgb 30/30/30) by 216 channels" tells a future maintainer what was
wrong; "max channel was 246, expected <80" tells them only that you guessed
80.

---

### Quick reference — mapping principles to bug categories

| Bug class | Principles that catch it |
|---|---|
| Visual / layout / contrast | #09 screen · #10 video · E visual baselines · M visual relationships |
| Permission / role-gated | #07 real user states · F test by role |
| State corruption / sequence | #05 sequences · D integration sequences |
| Performance / network | #11 network profiling · H budgets |
| Render edge cases (data) | #06 data variety · G parameterized data |
| Stress / load / rapid clicks | #04 stress · J boundary tests |
| "Wired but broken" | #13 wired vs works · #14 trust screen · K test the test |
| New regression in migrated code | #08 diff-driven · #12 read commits |
| Out-of-theme / inconsistent styling | #15 cross-surface compare · M visual relationships |
| Anything subtle a user notices | #03 be the user · #01 curiosity · #02 logic |

---

### Closing

A QA engineer who follows none of these but is genuinely curious will still find
more bugs than one who follows all of them mechanically. The mindset principles
in Tier 1 are necessary; the rest are amplifiers. Curiosity tempered by logic,
applied to the real user's experience — that is the entire craft.

The methods are just shape we give to a habit.
