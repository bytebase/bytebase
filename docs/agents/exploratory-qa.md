# Exploratory and Release QA

Doctrine for pre-release regression QA. Distilled from gaps between what methodical
walks find and what humans file. Use this guide for an explicitly scoped exploratory or release QA pass. Apply
coverage requirements to the agreed area; routine test edits follow
[the E2E authoring guide](../../frontend/tests/e2e/AGENTS.md).


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
and placebo confirm dialogs ("Delete this saved query?" with no body
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
