# Task-run log — a statement is a payload, not a line

Status: proposal · 2026-09-11

The task-run log cuts every executed statement to 80 characters and appends `...`. For the
statements customers actually run — SDL, or a `CREATE TABLE` with a few hundred columns — the row
says almost nothing, and the log offers no way to read or copy the rest.
[#21276](https://github.com/bytebase/bytebase/pull/21276) removes the cut, which is the right
diagnosis and a partial fix: the statement is still collapsed to a single run-on line with every
newline replaced by a space, it now floods a 200px scroll box, and there is still no way to copy
it. The rule this doc lands on: **the model returns two forms of every entry — the line the row
shows and the text that line stands for — and destroys neither.** Truncation becomes a CSS
property of the line. Copy and unfold act on the text. A statement row folds to one line by
default and unfolds in place with its original formatting; an error row stays open, because an
error is why the log was opened. The change is frontend-only and touches four surfaces that embed
the viewer.

## Problem

`getCommandExecuteDetail` (`frontend/src/components/task-run-log/model.ts:305`) ends:

```ts
const normalized = statement.trim().replace(/\s+/g, " ");
return normalized.length > 80
  ? `${normalized.substring(0, 80)}...`
  : normalized;
```

Two lossy steps, both in the model. `\s+ → " "` matches `\n` and `\r\n`, so a formatted statement
becomes one line. The 80-character cut then throws away everything past the opening clause. The
model hands `SectionContent` a single `detail: string`, so by the time a row renders, the
statement is no longer in the component tree at all — nothing downstream can offer to show or copy
it, and nothing can even tell that something was removed.

The number 80 is not a product decision that was made and can be revisited; it is a constant that
arrived with a refactor. The Vue cell it replaced
(`.../TaskRunLogTable/StatementCell.vue`, deleted in
[#18383](https://github.com/bytebase/bytebase/pull/18383)) clamped the row to one line too, but
passed the untouched statement to a `TextOverflowPopover` and a `CopyButton` beside it. The
rewrite kept the clamp, dropped both escape hatches, and hardcoded the width the clamp used to
derive from the container.

Two facts make the current row worse than it looks:

- **The uniform-row assumption is already false.** `SectionContent` sizes its scroll box as
  `ITEM_HEIGHT * MAX_VISIBLE_ITEMS` = 10 rows at 20px = 200px, but an error row returns
  `command.response.error` through the same `detail` field and renders under `break-words`, so it
  wraps to whatever height it needs. Rows of one fixed height are an assumption of the geometry,
  not a property of the data.
- **The UX contract already requires the missing half.** `docs/agents/frontend-ux.md`: *"Long
  single-line identifiers use truncation with a tooltip or another way to inspect the complete
  value."* The log has the truncation and not the way.

What #21276 changes and does not: dropping the `substring` returns the whole statement, but the
normalization above it stays, so a 200-line `CREATE TABLE` renders as one unbroken paragraph of
prose; it reflows to dozens of visual lines inside a 200px window, pushing the timing column it is
supposed to annotate off-screen; and copy is still absent, so the reader who needs the statement
must leave for the sheet. The finding is real and the contributor should get the credit for it;
this doc is the shape the fix takes.

## Principle

> **A row shows one line. The statement behind it is never destroyed to produce that line.**
> The model returns both forms — `detail`, the line, and `fullDetail`, the original text — and the
> view decides which to render. Collapsing whitespace and clamping width are properties of the
> line alone. Copy and unfold always act on the original.

What follows from it:

- No length constant lives in the model. The line is clamped by the width it is given, which is
  the only thing that knows how much fits.
- Any row that carries an original can offer to copy it, whether or not it can be unfolded.
- Whether a row starts folded is a statement about what the row is *for*, not about its length.
- The log stays a scannable run history. Unfolding is a deliberate act on one row, not a mode.

## Decisions

**D1 · Two forms on the item.** `DisplayItem` (`types.ts`) gains `fullDetail?: string` beside the
existing `detail: string`. `detail` keeps the `trim()` + `\s+ → " "` normalization — for a
one-line row that is exactly right, and it keeps stray newlines from breaking the row geometry.
`fullDetail` is the statement as the sheet stored it, untrimmed and unmodified. `getEntryDetail`
returns both; the rest of the model is unchanged. Absent for rows that have no underlying text
(`BEGIN`, `Completed`, retry counts), which is what makes it the test for both affordances below.

**D2 · Truncation moves to CSS.** The row's line renders under `truncate`; the 80-character
`substring` is deleted. A wide screen shows more of the statement and a narrow sheet shows less,
which is what the reader expects and what the constant could never do.

**D3 · Copy acts on the original, on any row that has one.** The shared `CopyButton`
(`components/ui/copy-button.tsx`) already owns the clipboard write, the toast, the 2-second check
state and the `common.copy` tooltip; it takes `content` as a thunk, so the string is resolved on
click rather than held per row. It copies `fullDetail` — real newlines, untrimmed — regardless of
whether the row is folded. This is the affordance that makes a statement useful outside the log,
and it is the one an error row needs most.

**D4 · The default fold state follows what the row is for.** A statement identifies *which*
command ran; the reader scanning for the failure does not want it open. An error is *why* they
opened the log. So a statement row starts folded and an error row renders exactly as it does
today — wrapped, whole, no fold control — and simply gains the copy button. Rows with no
`fullDetail` get neither control.

**D5 · A row is foldable when there is more to see.** Two independent causes: the original differs
from the line (it contained a newline or repeated whitespace), or the line is clamped at its
current width. The first is a string comparison. The second is a measurement — `scrollWidth >
clientWidth` on the line element, recomputed from one `ResizeObserver` on the scroll container and
when the item list changes — and it is what catches a single-line statement too wide for the
viewport, which the comparison alone would miss. Rows that are not foldable show no chevron and
do not respond to a click.

**D6 · The chevron is the control.** The row cannot be a `<button>`: it already contains one for
copy, and nested buttons are invalid. So the fold control is its own `<button>` carrying
`aria-expanded` and a name of its own ("Show full statement" / "Hide full statement"), the copy
button is a second, separate tab stop, and the row itself stays a `<div>` with no `role`. The
chevron's slot is reserved on every command row so the text column stays aligned whether or not a
given row is foldable.

**D7 · Clicking the row toggles it, except when a selection ends there.** Click-anywhere is what
makes the affordance usable at 12px, and it is the convention of every CI log. But an unfolded
statement is text people select and copy by hand, and a naive handler collapses the row on
mouse-up at the end of a drag. The handler ignores a click when `window.getSelection()` is not
collapsed, and ignores clicks that originate inside a button. Mouse convenience only — assistive
technology sees D6's controls.

**D8 · The section's cap rises while a row is open.** A 630px statement inside a 240px box is a
keyhole. While any row in a section is unfolded, that section's scroll box is capped at
`max(240px, 60vh)` instead of `ITEM_HEIGHT * MAX_VISIBLE_ITEMS`, and returns to the collapsed cap
when the last row folds. Both halves matter: `60vh` is the honest unit for "how much of the screen
may this take", and the `max()` floor keeps a short window from shrinking the box below its
collapsed height. The box still scrolls — a statement with a few hundred columns is taller than
any cap worth setting — but it scrolls over most of a screen instead of over ten lines.

**D9 · One scroll context.** The unfolded block never gets its own `overflow` — it grows to its
natural height and the section scrolls. A scrollbar inside a scrollbar inside a page is not a
thing the reader can operate at this size.

**D10 · The unfolded block replaces the line.** Same cell, same left edge; `whitespace-pre-wrap
break-words` on the original text, in the row's mono face, on `bg-background` inside a
`border-control-border` block. Replacing rather than appending keeps one statement on screen at a
time and keeps the index, timestamps and status glyph pinned to the row's first line, which
`items-start` already does.

**D11 · Nothing in the row moves on hover, and the row gets 4px taller.** The copy button occupies
its slot in the existing right-hand cluster whether or not it is visible, so revealing it shifts
nothing. It is visible on hover, on focus, while the row is open, and always where hover does not
exist (`pointer-coarse`). Both slots are reserved on *every* row, including `BEGIN` and
`Completed`, so one row height holds for the whole list. That height moves from 20px to 24px: the
smallest `Button` size is `h-6`, and a control that tall would push the row to 28px, so the copy
button is constrained to `h-5` with a `size-3` icon — 20px of content plus the existing `py-0.5`.
`ITEM_HEIGHT` moves from 20 to 24 with it, so `MAX_VISIBLE_ITEMS` keeps meaning ten rows and the
collapsed cap becomes 240px. A 20px target is the floor worth accepting here; anything smaller is
not clickable, and anything larger costs another 40px of vertical space per ten rows.

**D12 · Strings.** `common.copy`, `common.copied` and `common.copy-failed` already exist and come
free with `CopyButton`. The fold control needs two new keys under `task-run.log-detail`; they are
accessible names, not visible labels, and land in `en-US` with the other locales falling back
until translated.

## States

Mockups A–E are in the PR description. Product typography, spacing and semantic colors are taken
from the live component.

| | State | What it settles |
|---|---|---|
| A | Today | The 80-character cut, for comparison |
| B | Folded, hover | D2, D3, D5, D6, D11 — the two affordances and the reserved slots |
| C | Unfolded | D8, D9, D10 — original formatting, raised cap, one scrollbar |
| D | An error and a statement in one section | D4 — the two default fold states side by side |
| E | Narrow container | D2 — the same row in the deploy sheet, clamped by its width |

## Scope

Frontend only. The viewer is embedded by `DatabaseChangelogDetailPage`, `RevisionDetailPanel`,
`DeployTaskRunHistorySheet` and `DeployLatestTaskRunInfo`; all four inherit the change.

| File | Change |
|---|---|
| `task-run-log/types.ts` | `fullDetail?: string` on `DisplayItem` |
| `task-run-log/model.ts` | Delete the `substring`; return both forms from `getEntryDetail` |
| `task-run-log/SectionContent.tsx` | Fold control, copy button, CSS clamp, overflow measurement, section cap, `ITEM_HEIGHT` 20 → 24 |
| `locales/en-US.json` | Two accessible names |

Tests, none of which exist today — which is how #21276 passed a clean suite while changing the
behavior of this function:

- `model.test.ts`: a multi-line statement yields a collapsed `detail` and an untouched
  `fullDetail`; a statement past 80 characters is not truncated (regression); an error populates
  both; an entry with no statement yields `"-"` and no `fullDetail`.
- `SectionContent`: a foldable row toggles and reports `aria-expanded`; copy receives the original
  text, not the line; the open set resets when `datasetKey` changes, as `showAllItems` already
  does; an error row renders wrapped with a copy button and no chevron.

## Not in this PR

The implementation, which follows separately. Also deliberately out:

- **Error rows becoming foldable.** They are open by default (D4); a fold control on them is a
  second decision, worth making only if long errors turn out to crowd the sections they appear in.
- **Syntax highlighting in the unfolded block.** A Monaco instance per log row is far past what
  this surface can afford. The statement is mono, preformatted and copyable; the editor is one
  click away on the pages that embed the viewer.
- **The other entry types.** `SCHEMA_DUMP`, `DATABASE_SYNC` and friends carry status words, not
  payloads. They gain `fullDetail` only if they ever carry one.
