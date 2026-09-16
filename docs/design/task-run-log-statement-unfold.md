# Task-run log — a statement is a payload, not a line

Status: proposal · 2026-09-11 (revised 2026-09-16)

The task-run log cuts every executed statement to 80 characters and appends `...`. For the
statements customers actually run — SDL, or a `CREATE TABLE` with a few hundred columns — the row
says almost nothing, and the log offers no way to read or copy the rest. A failed command is worse:
the row shows the driver's error and drops the statement entirely, so the one row a reader opened
the log for cannot tell them which SQL produced it.
[#21276](https://github.com/bytebase/bytebase/pull/21276) removes the cut, which is the right
diagnosis and a partial fix: the statement is still collapsed to a single run-on line with every
newline replaced by a space, it now floods a 200px scroll box, there is still no way to copy it,
and the failed row is untouched. The rule this doc lands on: **a row shows one line, and nothing
the row is about is destroyed to produce that line.** Truncation becomes a CSS property of the
line. Unfolding always reveals the statement, and so does copy. The change is frontend-only and
touches four surfaces that embed the viewer.

## Problem

`getCommandExecuteDetail` (`frontend/src/components/task-run-log/model.ts:305`) is lossy twice, and
silently drops a payload a third time:

```ts
if (command.response?.error) return command.response.error;   // the statement is never read

const statement = command.statement || (command.range ? extractStatementFromRange(...) : undefined);
if (!statement) return "-";

const normalized = statement.trim().replace(/\s+/g, " ");
return normalized.length > 80
  ? `${normalized.substring(0, 80)}...`
  : normalized;
```

`\s+ → " "` matches `\n` and `\r\n`, so a formatted statement becomes one line. The 80-character
cut then throws away everything past the opening clause. And the early return above it means a
failed command's row is the error and only the error. The model hands `SectionContent` a single
`detail: string`, so by the time a row renders, the statement is not in the component tree at all —
nothing downstream can offer to show or copy it, and nothing can even tell that something was
removed.

The failed row is the sharpest case, because the data is already there and already joined. The API
converter builds the entry from the store's `COMMAND_EXECUTE` record with `Range` and `Statement`,
then, on the following `COMMAND_RESPONSE` record, pops that same entry and hangs the response on it
(`backend/api/v1/rollout_service_converter.go:396-418`). One entry, both payloads. Nor does the
error text stand in for the statement: every driver logs a bare `err.Error()`
(`backend/plugin/db/pg/pg.go:573` and its siblings), so the reader gets
`ERROR: column "risk_score" of relation "employee" already exists (SQLSTATE 42701)` and no way to
know which of forty statements said it, short of counting row indexes against the sheet by hand.

The number 80 is not a product decision that was made and can be revisited; it is a constant that
arrived with a refactor. The Vue cell it replaced
(`.../TaskRunLogTable/StatementCell.vue`, deleted in
[#18383](https://github.com/bytebase/bytebase/pull/18383)) clamped the row to one line too, but
passed the untouched statement to a `TextOverflowPopover` and a `CopyButton` beside it. The
rewrite kept the clamp, dropped both escape hatches, and hardcoded the width the clamp used to
derive from the container.

Two more facts shape the fix:

- **The uniform-row assumption is already false.** `SectionContent` sizes its scroll box as
  `ITEM_HEIGHT * MAX_VISIBLE_ITEMS` = 10 rows at 20px = 200px, but an error row renders under
  `break-words` and wraps to whatever height it needs. Rows of one fixed height are an assumption
  of the geometry, not a property of the data.
- **The UX contract already requires the missing half.** `docs/agents/frontend-ux.md`: *"Long
  single-line identifiers use truncation with a tooltip or another way to inspect the complete
  value."* The log has the truncation and not the way.

## Principle

> **A row shows one line. Nothing the row is about is destroyed to produce that line.**
> A command row is about the statement it ran and, when it failed, the error that came back — two
> payloads the entry already carries together. The model returns the line plus both payloads, and
> the view decides which one the line summarizes, which one unfolding reveals, and which one copy
> takes.

What follows from it:

- No length constant lives in the model. The line is clamped by the width it is given, which is
  the only thing that knows how much fits.
- The line is the news: the error when the command failed, the statement when it did not.
- Unfolding always reveals the statement, and so does copy. One meaning per control, on every row.
- Whether a row starts unfolded says what the row is *for*, not how long it is.

## Decisions

**D1 · Both payloads on the item.** `DisplayItem` (`types.ts`) grows two optional fields beside the
existing `detail: string`:

```ts
detail: string;      // the text the row shows
statement?: string;  // the statement this row ran, as the sheet stored it
error?: string;      // the error the command returned
```

`detail` keeps the `trim()` + `\s+ → " "` normalization when it summarizes a statement — for a
one-line row that is exactly right, and it keeps stray newlines from breaking the row geometry.
`statement` and `error` are verbatim, untrimmed. `getEntryDetail` returns all three; the entry
types that carry status words rather than payloads (`BEGIN`, `Completed`, retry counts) return
`detail` alone, which is what makes the two new fields the test for both controls below.

**D2 · Truncation moves to CSS.** The row's line renders under `truncate`; the 80-character
`substring` is deleted. A wide screen shows more of the statement and a narrow sheet shows less,
which is what the reader expects and what the constant could never do. An error line is not
clamped — it wraps whole, as it does today.

**D3 · Copy takes the statement.** The shared `CopyButton` (`components/ui/copy-button.tsx`)
already owns the clipboard write, the toast, the 2-second check state and the `common.copy`
tooltip; it takes `content` as a thunk, so the string is resolved on click rather than held per
row. It copies `statement` — real newlines, untrimmed — on every row that ran one, folded or not,
failed or not, so "copy gives you the SQL" needs no exceptions. It falls back to `error` on a
failed row whose statement could not be recovered (no `statement`, no usable `range`, or a sheet
that came back partial). Rows with neither get no copy button.

**D4 · What unfolds by default says what the row is for.** A successful statement identifies
*which* command ran; the reader scanning for the failure does not want it open, so it starts
folded. A failed command is the thing the log was opened for, and execution stops at the first one,
so at most one row per run is affected: it starts **unfolded**, with the error as its line and the
statement that failed in the block beneath. Either can be toggled, and the control means the same
thing in both directions.

**D5 · A row is foldable when it ran a statement.** Nothing more. An earlier draft measured
`scrollWidth > clientWidth` from a shared `ResizeObserver` so the chevron could be hidden on rows
where unfolding changed nothing; that bought a cosmetic gain with a layout-measurement subsystem
across up to 50 rendered rows. Once unfolding means "show me the SQL" rather than "show me the rest
of this line", unfolding a short statement is redundant rather than wrong, and D11 reserves the
slot on every row regardless — so the uniform chevron costs ink, not layout, and the measurement
is deleted.

**D6 · Both controls are shared `Button`s.** The fold control is an icon-only `Button` carrying
`aria-expanded` and a name of its own ("Show full statement" / "Hide full statement"); `CopyButton`
is a second, separate tab stop; the row itself stays a `<div>` with no `role`. The row cannot be
the control — it contains two buttons, and nested buttons are invalid — and neither control may be
native markup: `no-native-control` in `frontend/scripts/check-ui-guideline.mjs` fails
`pnpm --dir frontend check` on a raw `<button>` in feature code. The chevron's slot is reserved on
every command row so the text column stays aligned.

**D7 · Clicking the row toggles it, except when a selection ends there.** Click-anywhere is what
makes the affordance usable at 12px, and it is the convention of every CI log. But an unfolded
statement is text people select and copy by hand, and a naive handler collapses the row on
mouse-up at the end of a drag. The handler ignores a click when `window.getSelection()` is not
collapsed, and ignores clicks that originate inside a button. Mouse convenience only — assistive
technology sees D6's controls.

**D8 · The section's cap rises while a row is open.** A 630px statement inside a 280px box is a
keyhole. While any row in a section is unfolded, that section's scroll box is capped at
`max(280px, 60vh)` instead of `ITEM_HEIGHT * MAX_VISIBLE_ITEMS`, and returns to the collapsed cap
when the last row folds. Both halves matter: `60vh` is the honest unit for "how much of the screen
may this take", and the `max()` floor keeps a short window from shrinking the box below its
collapsed height. A run that failed therefore opens taller than one that did not, by way of D4,
which is the right way round. The box still scrolls — a statement with a few hundred columns is
taller than any cap worth setting — but it scrolls over most of a screen instead of over ten lines.

**D9 · One scroll context.** The unfolded block never gets its own `overflow` — it grows to its
natural height and the section scrolls. A scrollbar inside a scrollbar inside a page is not a
thing the reader can operate at this size.

**D10 · The unfolded block takes the line's place, or sits under it.** Same cell, same left edge;
`whitespace-pre-wrap break-words` on the verbatim statement, in the row's mono face, on
`bg-background` inside a `border-control-border` block. On a successful row it replaces the line,
so one statement is on screen at a time. On a failed row the error keeps the line and the block
sits beneath it, because both payloads are the point. Either way the index, timestamps and status
glyph stay pinned to the row's first line, which `items-start` already does.

**D11 · Nothing moves on hover, and the row gets 8px taller.** Both controls occupy their slots
whether or not they are visible, so revealing copy shifts nothing; it is visible on hover, on
focus, while the row is open, and always where hover does not exist (`pointer-coarse`). Both slots
are reserved on every row, including `BEGIN` and `Completed`, so one row height holds for the whole
list. That height is set by the shared size contract and not by this surface:
`docs/agents/frontend-ux.md:75-83` forbids a consumer from replacing a shared control's managed
height or resizing its managed icon, `no-button-dimension-override` enforces it in
`check-ui-guideline.mjs`, and the smallest shared size is `xs` at 24px with a 14px icon. So both
controls are `xs`, the row's content is 24px, the existing `py-0.5` makes the row 28px, and
`ITEM_HEIGHT` moves from 20 to 28 so `MAX_VISIBLE_ITEMS` keeps meaning ten rows — a collapsed cap
of 280px, up from 200px. Two ways to buy that density back were considered and not taken: dropping
`MAX_VISIBLE_ITEMS` to 7 holds the old page footprint but shows fewer entries before scrolling,
and a 20px size tier in the shared `Button` CVA and the contract table is a design-system change
that needs its own owner.

**D12 · Strings.** `common.copy`, `common.copied` and `common.copy-failed` already exist and come
free with `CopyButton`. The fold control needs two new keys under `task-run.log-detail`; they are
accessible names, not visible labels, and land in `en-US` with the other locales falling back
until translated.

## States

Mockups A–E are in the PR description. Product typography, spacing and semantic colors are taken
from the live component.

| | State | What it settles |
|---|---|---|
| A | Today | The 80-character cut, and a failed row that is error-only |
| B | Folded, hover | D2, D3, D5, D6, D11 — one meaning per control, reserved slots |
| C | Unfolded | D8, D9, D10 — verbatim formatting, raised cap, one scrollbar |
| D | A failed command | D1, D4, D10 — the error keeps the line, the failed statement sits beneath it |
| E | Narrow container | D2 — the same rows in the deploy sheet, clamped by its width |

## Scope

Frontend only. The viewer is embedded by `DatabaseChangelogDetailPage`, `RevisionDetailPanel`,
`DeployTaskRunHistorySheet` and `DeployLatestTaskRunInfo`; all four inherit the change.

| File | Change |
|---|---|
| `task-run-log/types.ts` | `statement?: string` and `error?: string` on `DisplayItem` |
| `task-run-log/model.ts` | Delete the `substring`; read the statement for failed commands too; return all three fields |
| `task-run-log/SectionContent.tsx` | Fold control, copy button, CSS clamp, default-open failed rows, section cap, `ITEM_HEIGHT` 20 → 28 |
| `locales/en-US.json` | Two accessible names |

Tests, none of which exist today — which is how #21276 passed a clean suite while changing the
behavior of this function:

- `model.test.ts`: a multi-line statement yields a collapsed `detail` and a verbatim `statement`; a
  statement past 80 characters is not truncated (regression); a failed command yields the error as
  `detail` *and* the failed statement in `statement`, including when it has to come from `range`;
  an entry with no statement yields `"-"` and no `statement`.
- `SectionContent`: a foldable row toggles and reports `aria-expanded`; a failed row starts
  unfolded and can be folded; copy receives the verbatim statement, not the line, and falls back to
  the error only when no statement was recovered; the open set resets when `datasetKey` changes, as
  `showAllItems` already does.

## Not in this PR

The implementation, which follows separately. Also deliberately out:

- **A smaller shared control size.** The 8px the row gains is the cost of the shared size contract
  (D11). A 20px tier is a design-system change, not a log-viewer one.
- **Syntax highlighting in the unfolded block.** A Monaco instance per log row is far past what
  this surface can afford. The statement is mono, preformatted and copyable; the editor is one
  click away on the pages that embed the viewer.
- **The other entry types.** `SCHEMA_DUMP`, `DATABASE_SYNC` and friends carry status words, not
  payloads. They gain the new fields only if they ever carry one.
