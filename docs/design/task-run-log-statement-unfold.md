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
key: string;          // the entry's identity, not its position — see D14
detail: string;       // the text the row shows
statement?: string;   // the statement this row ran, as the sheet stored it
error?: string;       // the error the command returned
defaultOpen?: boolean // this is the row D4 picked; the mark D13 and D14 speak of
```

`key` is listed because its contents change even though the field does not. It is the only
identifier `SectionContent` ever sees, so if it stays positional the view has nothing stable to
report when a reader toggles a row, and D14's persistence cannot be built at all.

`detail` keeps the `trim()` + `\s+ → " "` normalization when it summarizes a statement — for a
one-line row that is exactly right, and it keeps stray newlines from breaking the row geometry.
`statement` and `error` are verbatim, untrimmed. `getEntryDetail` returns the first three; the entry
types that carry status words rather than payloads (`BEGIN`, `Completed`, retry counts) return
`detail` alone, which is what makes `statement` and `error` the test for both controls below.

A failure whose statement could not be recovered — no `statement`, no usable `range`, or a sheet
that came back partial — still gets marked, because the mark's first job is to say *this is the row
that explains the outcome*, and D13 has to render and scroll to it or the error itself stays hidden
behind *Load more*. What it does not get is a taller section: D8's cap answers to a block actually
being on screen, not to the mark, and there is no block here. Opening such a row is a no-op, which
is why D5 gives it no control to open with.

`defaultOpen` is the fourth, and it exists because the view cannot work it out. D4's pick is made
across a whole execution context, and `SectionContent` sees one section at a time; nor can it infer
the pick from `error`, since every transient failed attempt carries one and opening all of them is
the thing D4 exists to prevent. So `buildSectionsFromEntries` decides and marks the row it chose,
and everything downstream — D13's render window, D14's derived open set — reads that flag rather
than re-deriving a judgement it lacks the inputs for.

**D2 · Truncation moves to CSS.** The row's line renders under `truncate`; the 80-character
`substring` is deleted. A wide screen shows more of the statement and a narrow sheet shows less,
which is what the reader expects and what the constant could never do. An error line is not
clamped — it wraps whole, as it does today.

**D3 · Copy takes the statement, and sits beside it.** The shared `CopyButton`
(`components/ui/copy-button.tsx`) already owns the clipboard write, the toast, the 2-second check
state and the `common.copy` tooltip; it takes `content` as a thunk, so the string is resolved on
click rather than held per row. It copies `statement`, verbatim — real newlines, untrimmed — and
never anything else. Where it sits follows what it copies, because position is what a reader
actually reads: in the row's right-hand cluster while the statement *is* the line, and in the
top-right corner of the unfolded block once the statement is a block, which is the placement every
code block on GitHub has already trained people to expect. A failed row's line is the error, so no
copy button appears on it; that row's copy is in its block, which D4 opens by default. The error
gets none: wanting an error in the clipboard is rare next to wanting the SQL, and it does not
justify a second control on the densest surface in the product — it stays selectable text, as it
is today. A row whose statement could not be recovered at all (no `statement`, no usable `range`,
or a sheet that came back partial) gets no copy button rather than a fallback to something else,
so the control never means two things. One meaning, one place: copy is the SQL, beside the SQL.

**D4 · What unfolds by default says what the row is for.** A successful statement identifies
*which* command ran; the reader scanning for the failure does not want it open, so it starts
folded. A failed command is what the log was opened for, so it starts **unfolded**, with the error
as its line and the statement that failed in the block beneath — but only the one that explains the
outcome, which is not the same as every failed row. A run can hold several: on a lock timeout the
Postgres driver retries the whole command list up to `MaximumRetries`
(`backend/plugin/db/pg/pg.go:463-485`), re-logging every command on each attempt, so the same
statement can fail three times and the run still succeed. Opening all of them would expand the same
DDL three times over.

So the rule is: **open the last failed command row in an execution context, and only when no
command after it succeeded.** The context is the entry sequence `buildSectionsFromEntries` is
handed — one replica, one release file — and the test runs over that sequence *before* it is
grouped. It cannot be section-local: sections are typed groups, and `groupEntriesByType` starts a
new one whenever the entry type changes, while a retry emits a `RETRY_INFO` entry between attempts
and, in transaction mode, a rollback and a fresh begin as well. Every transient failure would
therefore be the last failure in a section of its own, and all of them would open. Scanned across
the context instead, a run that recovered has successful commands after its failure and opens
nothing, while a run that really failed ends at the failure and opens exactly that row. Per
context, not per run, so a failure on one replica is never silenced by another replica's success.
A live log is a prefix of the finished one, so the test also treats a failure followed by a
`RETRY_INFO` entry as not terminal. The driver writes that marker before it re-runs
(`backend/plugin/db/driver.go:357-367`, called at `pg.go:470`), so the marker is the standing
signal that another attempt is coming; without it, the five-second poll
(`useTaskRunLogData.ts:101`) landing between a failed attempt and its retry would open a failure
that is about to be superseded. What remains is the ~200ms between the failure and the marker,
and that window is left open deliberately. A poll landing inside it — roughly one chance in
twenty-five, and only for someone watching a retrying run live — opens a failure that the next
poll folds again when the marker arrives and the mark moves off it, so the row can sit expanded for
up to five seconds. Suppressing the mark while the run is still `RUNNING` would remove the flicker
and take D14's entire point with it, since the case the mark exists for *is* a deploy watched from
the plan page that fails while you are looking at it. The transient state is not false either: that
command really did fail at that moment; it is only about to be tried again.

The test reads only the entries, which matters because `taskRunStatus` is an optional prop that the
changelog and revision pages do not pass. One guard uses it where it exists: when a caller passes a
status of `DONE`, nothing is marked at all. That is cheap, it costs D14 nothing — D14 is about a run
still `RUNNING` — and it covers the one shape the entries get wrong today. CockroachDB's autocommit
path logs a single `COMMAND_EXECUTE` outside its retry and a response *per attempt* from inside it
(`backend/plugin/db/cockroachdb/cockroachdb.go:485-503`), while the converter attaches the first
response and drops the rest (`rollout_service_converter.go:408-417`), so a statement that succeeded
on retry is recorded as a failure. No rule reading entries can recover a response the converter
discarded, and the row is already wrong before this design reaches it: the log shows a red ✗ for a
command that worked. The guard is opportunistic mitigation, not part of the rule: it
costs nothing where the prop already exists and it is not extended to where it does not. The
changelog and revision pages would need a success signal invented for them — a changelog's status
is not a task run's — to half-cover one engine's converter bug, and on those two pages that row
**already** reads as a failure today, before this design touches it. Auto-opening makes a wrong row
larger; it does not make it wrong. The converter fix below closes it on all four surfaces at once,
which is why that is where it belongs. Everything else starts folded, and every row toggles either
way.

**D5 · A row is foldable when unfolding would show something new.** There are two independent
reasons it would, and a row needs only one of them:

- the row carries both an `error` and a `statement`, because then the line is the error and the
  statement is never on it — unfolding is the only way to reach the SQL, however short it is;
- the verbatim statement differs from the line it was collapsed into, because it had newlines or
  runs of whitespace — a string comparison, free;
- the line is clamped by the width it was given — `scrollWidth > clientWidth`, which has to be
  measured.

The first is the one an implementation would most easily miss, and missing it sets a trap. A
terminal failure on something as short as `SELECT 1` satisfies neither of the other two — its
statement has no newlines and its line, being the error, is never clamped — so the row would open
by default under D4 and then, once the reader folded it, offer no way back: the statement and the
copy button that lives with it are both inside the block. Fold once, lose the SQL. A row holding
two payloads is always foldable.

Neither covers the other. A three-line statement can collapse to a line that fits, and a one-line
statement can be far too wide; mockup E is the proof, where `SET statement_timeout TO '3600s';`
fits whole at 900px and is cut at the deploy sheet's 560px. A rule without the measurement would
leave that row truncated on screen with no way to open it.

An earlier draft dropped both tests and gave a chevron to every row that ran a statement, arguing
that unfolding a short statement is redundant rather than wrong. It is wrong. A control that
reveals nothing teaches the reader that the control is decoration, and then they stop reaching for
it on the rows where it matters — mockup D's row 6 is exactly that, a `COMMENT ON TABLE` that fits
its line and would unfold to itself.

A row that is open **and has a statement** is foldable — no measurement, no comparison. The
statement clause is not decoration: a failure whose statement could not be recovered is still
marked (D1) and so is nominally open, but it has no block, and an unqualified rule would hand it
the chevron that D1 and D5 both promise it will not get. The exception exists to let a reader close
a block that is on screen, so it reaches exactly as far as a block does. Beyond that it sounds like
a truism and is in fact the subtlest case here. A successful single-line statement qualifies only
by the clamp, and D10 has the block *replace* the line when it opens, so the thing the clamp was
measured on no longer exists. Worse, opening raises D8's cap, which resizes the scroll box, which
fires the very observer that would reclassify it — so the chevron would vanish from a row while it
sat expanded, and with it the way back, since the row's click target is gated on the same verdict.
You must always be able to close what you opened; the verdict is recomputed only for closed rows.

The measurement has two triggers, and the width one is not enough on its own. A `ResizeObserver` on
the section's scroll box catches the container changing width, but that box is capped: pressing
*Load more*, or a poll appending entries, grows its scroll height and leaves its border box exactly
where it was, so the observer never fires and the new rows would render clamped with no control. So
the measurement also re-runs whenever the rendered set changes — `showAllItems` flipping, `items`
changing identity, the marked row moving — and each run covers **every row currently rendered**,
not the first `MAX_RENDERED_ITEMS`. One observer plus one layout effect, reading only.

Two consequences worth stating rather than discovering:

- **A narrow container turns most rows foldable.** At the deploy sheet's width a line holds roughly
  55 characters, so most statements clamp and most rows earn a chevron. That is the honest result,
  not a defect: at that width, unfolding each of them really does reveal more. Which rows are *cut*
  is already signalled by the ellipsis the clamp draws — the chevron is the control, not the
  indicator, and the two carry different news. A chevron with no ellipsis means there is formatting
  to see; an ellipsis means there is text to see.
- **Only a row that ran a statement ever gets one.** `BEGIN`, `COMMIT`, `Completed`, retry counts,
  and a failed row whose statement could not be recovered all carry no `statement` (D1), so they
  get neither control — while keeping the reserved slot, so the statement column stays straight
  (D15).

**D6 · Both controls are shared `Button`s.** The fold control is an icon-only `Button` carrying
`aria-expanded` and a name of its own ("Show full statement" / "Hide full statement"); `CopyButton`
is a second, separate tab stop; the row itself stays a `<div>` with no `role`. The row cannot be
the control — it contains two buttons, and nested buttons are invalid — and neither control may be
native markup: `no-native-control` in `frontend/scripts/check-ui-guideline.mjs` fails
`pnpm --dir frontend check` on a raw `<button>` in feature code. The chevron's slot is reserved on
every command row so the text column stays aligned.

**D7 · Clicking the row toggles it — anywhere but the statement itself.** Click-anywhere is what
makes the affordance usable at 12px, and it is the convention of every CI log, so the index, the
timestamps, the glyph and the empty space all toggle. The statement does not, whether it is the
clamped line or the unfolded block, because text a reader can select must not double as a button.

A selection guard alone cannot carry that. It catches the drag — mouse-up at the end of a sweep,
where the selection is already non-collapsed — but not the double-click, whose *first* click
arrives with the selection still collapsed and would toggle the row, replacing the very text the
second click was aiming at. Nothing observable at that moment distinguishes it from a single click,
and waiting to find out would put a double-click delay on every toggle. Excluding the text removes
the question instead of timing it.

So: clicks inside a button are ignored (D6 owns those), clicks on the statement are ignored, a
click that ends a selection is still ignored as a belt-and-braces guard for a drag that began on
the metadata and finished over the text, and everything else toggles. Mouse convenience only —
assistive technology sees D6's controls.

**D8 · The section's cap rises while a block is on screen.** A 630px statement inside a 280px box is a
keyhole. While any row in a section is unfolded, that section's scroll box is capped at
`max(280px, 60vh)` instead of `ITEM_HEIGHT * MAX_VISIBLE_ITEMS`, and returns to the collapsed cap
when the last row folds. Both halves matter: `60vh` is the honest unit for "how much of the screen
may this take", and the `max()` floor keeps a short window from shrinking the box below its
collapsed height. A run that failed therefore opens taller than one that did not, by way of D4,
which is the right way round. The box still scrolls — a statement with a few hundred columns is
taller than any cap worth setting — but it scrolls over most of a screen instead of over ten lines.

**D9 · The viewer owns one scroll context.** The unfolded block never gets its own `overflow` — it
grows to its natural height and the section scrolls, so nothing the viewer renders stacks a
scrollbar inside a scrollbar. It cannot claim the same for the page it sits on:
`DeployTaskRunHistorySheet` renders the viewer inside `SheetBody`, which is `overflow-y-auto`
(`components/ui/sheet.tsx:161-167`), so on that one surface the section's box has always sat inside
an outer scroller, and D8's raised cap makes the inner region bigger. The nesting predates this
change and the sheet's `overscroll-contain` keeps it from chaining. Handing the scroll to the host
— no cap when the viewer is inside a sheet, `SheetBody` scrolling the expanded statement — would
remove it, at the price of a second layout mode for the viewer to carry, test, and keep consistent
with `MAX_RENDERED_ITEMS`. Recorded as the option and not taken: the nesting is pre-existing and
mild, the mode would be permanent.

**D10 · The unfolded block takes the line's place, or sits under it.** Same cell, same left edge;
`whitespace-pre-wrap break-words` on the verbatim statement, in the row's mono face, on
`bg-background` inside a `border-block-border` block — the block is a framed content region, and
`docs/agents/frontend-ux.md:143-144` keeps `border-control-border` for controls — with D3's copy
button in its top-right corner over padding wide enough to hold it, never over the SQL. That
clearance is not decoration: the contract says text "must not overlap adjacent controls"
(`frontend-ux.md:113-114`), and the rows D5 newly makes foldable are precisely the ones whose first
wrapped line runs the full width of the block, so an overlay with no reserved padding would sit on
top of the statement in the common case rather than a rare one. The block's right padding reserves
the control's width plus the gap beside it; the text wraps before it
corner — positioned inside the block's padding so it overlays rather than reflows the SQL, and
visible for as long as the block is. On a successful row the block replaces the line, so one
statement is on screen at a time and the cluster's copy button gives way to the block's. On a
failed row the error keeps the line and the block sits beneath it, because both payloads are the
point. Either way the index, timestamps and status glyph stay pinned to the row's first line,
which `items-start` already does.

**D11 · Nothing moves on hover, and the row gets 8px taller.** The cluster's copy button occupies
its slot whether or not it is visible, so revealing it on hover shifts the duration and
affected-rows beside it by nothing; it shows on hover, on focus, and always where hover does not
exist (`pointer-coarse`). Row height comes from the fold control's slot instead, which D6 reserves
on every row including `BEGIN` and `Completed`, so one height holds for the whole list whatever
else a row carries. That height is set by the shared size contract and not by this surface:
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

**D13 · A row that opens by default has to be rendered, numbered and in view.** `SectionContent`
renders only the first
`MAX_RENDERED_ITEMS` (50) entries of a section until *Load more* is pressed
(`SectionContent.tsx:29-32`). A migration whose 300th statement fails would therefore have D4 mark
a row that is not in the DOM, and the promise would quietly do nothing in the case that needs it
most — the longest sections are exactly the ones where hunting for the failure by hand is worst.
The rendered window is therefore the first `MAX_RENDERED_ITEMS` items **plus the marked row when it
falls outside them**, with the existing *Load more* button between the two reporting how many
entries it hides; pressing it renders the rest and the marked row keeps its place in sequence.
Rendering the whole section would keep the promise too, but a section has no bound — a release file
can carry thousands of statements — and not having to render all of them is what the cap is for.

Rendering it is not the same as showing it. Fifty rows at 28px is roughly 1,400px of content above
the marked row, inside a box capped at `max(280px, 60vh)` whose `scrollTop` starts at zero, so the
row would be mounted, expanded and off-screen — the promise kept in the DOM and broken on the
screen. When a row becomes the marked one, the section sets its own `scrollTop` to bring it into
view. Its own, not `scrollIntoView`, which would scroll the page under a reader who was looking at
something else. It fires once, when the mark lands on a row — and on mount, if a
marked row is already there, which is what a reader expanding a section they had collapsed sees.
Later polls do not re-scroll, so a reader who has scrolled elsewhere stays where they are.

The window has to carry each item's own index with it. `SectionContent` numbers rows by their
position in the rendered array (`index + 1` over `visibleItems`, `SectionContent.tsx:40-50`), so a
sparse window would label statement 300 as row 51 and then renumber it to 300 once *Load more* was
pressed — pointing the reader at the wrong statement, which is worse than not surfacing it. The
number shown is the item's index in the section, and the D13 regression test asserts that number,
not merely that the row rendered.

**D14 · The open set follows the mark while the log is live.** The viewer polls a running task every
five seconds (`useTaskRunLogData.ts:101`), and `section.items` changes without `datasetKey`
changing. Applying D4's mark only when the component mounts would therefore miss the case the
feature exists for — a deploy watched from the plan page, succeeding command by command, that then
fails — because the failure arrives on a later poll into an already-mounted section, and nothing
would open it. So the open set is derived rather than initialised: a row is open when it is marked
or the reader opened it, and folded when the reader folded it. An explicit toggle outranks the mark
for as long as the dataset lasts, so a failure the reader folded stays folded through the next
poll, and a mark that moves off a row — a transient failure that turns out to have been retried —
takes its auto-open with it.

They also cannot be keyed by the row's render key. That key is positional —
`` `${idPrefix ?? "section"}-${groupIndex}-${entryIndex}` `` (`model.ts:481`) — and both halves move
underneath it. `useTaskRunLogSections` sets `idPrefix` to the replica id once a run has more than
one replica (`:131`, `:148`, `:158`), so the first replica's rows are `section-…` while it is alone
and `<replicaId>-…` the moment a second replica's first entry lands; and `groupIndex` shifts for
every later group when a retry marker splits a section in two. Either renumbering silently drops
the reader's folds on a dataset that never changed. So a row is keyed by the entry's own identity,
not by where it currently sits.

An earlier draft of this decision kept the positional key for rendering and gave the override map a
second, stable one. That was a hedge, and it was the wrong call: it leaves two keys on one row and
makes every future consumer pick correctly, when picking wrong is the entire bug. `DisplayItem.key`
becomes the identity itself. Nothing reads its shape — `SectionContent` passes it to React and
that is all (`SectionContent.tsx:42`) — and React gains from the change too, since rows currently
remount for no reason when a replica appears and every key is rewritten.

That identity needs care, because the obvious tuple is not unique. `task_run_log` has no primary
key *by design* — "entries for one task run can legitimately share a `created_at` microsecond"
(`backend/migrator/migration/LATEST.sql:705-706`) — so time alone ties. A statement's byte offset
cannot break the tie either: `LogCommandExecute` writes `Statement` **or** `Range` and never both
(`backend/plugin/db/driver.go:297-302`), and SDL sets `LogCommandStatement = true`
(`database_migrate_executor.go:730`), so in exactly the migrations this design exists for, command
entries carry no offset at all. Two tied rows would then share one key, and folding either would
fold both.

The identity is therefore the replica id, the log time at **full precision** — the proto carries
seconds and nanos, while `getTimestampMs` throws the sub-millisecond part away — the entry type,
and an ordinal among the entries that match all three. The ordinal comes from the run's entry list
as the API returns it, which is append-only and chronological, so a tie-group's existing members
never reorder; and none of the four terms mentions a section, a replica grouping or a render
position, which is what makes it survive the regrouping above.

Where that ordinal is computed decides whether it works. It has to be assigned **once, over the
run's whole entry list, before any filtering** — and then travel with the entry. Computing it
inside `buildSectionsFromEntries` would reset it per call, and `buildReleaseFileGroups` calls that
builder once per file with only that file's entries (`model.ts:569-606`), so two tied entries in
different release files would both be ordinal 0 and collide on a key that is supposed to be unique
across the viewer. Folding a row in one file would fold a row in another. The builders receive
entries that already know their identity; they never mint one.

Those overrides cannot live in `SectionContent` either, because it is mounted conditionally: collapsing an
enclosing section unmounts it and reopening builds a fresh one (`TaskRunLogViewer.tsx:195-201`), and
a live run also swaps the sole-section rendering for the multi-section tree the moment a second
entry type arrives. Either remount would discard the reader's folds and pop a marked failure back
open on a dataset that never changed. The overrides therefore sit in `TaskRunLogViewer`, above the
conditional mount, keyed by row and cleared when `taskRunName` changes — the same moment
`datasetKey` already clears `showAllItems`. `showAllItems` keeps its remount-local behavior;
re-hiding the tail of a long list on collapse is not a promise anyone made.

It also stops at a collapsed ancestor, deliberately. The viewer already expands a section the
moment its status turns to error — unless the reader collapsed that section themselves, which
`userCollapsedSections` records and honours (`useTaskRunLogSections.ts:234-247`), with the same
treatment for replicas and release files. A mark arriving into a section the reader has shut does
not reopen it. Doing so would break the precedence this decision just established, where an
explicit toggle outranks the mark, and it would override the reader at the one moment they have
most clearly said what they want on screen. Nothing is hidden by this: a collapsed section still
carries its own status, so the header turns to the error glyph and says a failure is inside. When
the reader opens it, the marked row is already unfolded and the section is already scrolled to it,
because D13's scroll fires on mount whenever a marked row is present, not only when a mark lands on
a row that is already showing.

The promise stops at the phase boundary too, deliberately. Both deploy surfaces key the viewer on the
run's status (`` key={`logs-${taskRun.name}-${taskRun.status}`} ``), so a `RUNNING` → terminal flip
remounts the whole viewer and takes the overrides with it. That key is not an accident — the
comment above it says the remount exists "for a fresh disclosure state on the new phase", and it is
also what invalidates an in-flight `RUNNING` log request before it can be cached as complete.
Persisting overrides across it, in a module-level map keyed by task-run name, would work and would
quietly undo that intent. And the behavior it produces is the one this doc wants anyway: the flip
that reopens a folded failure is the flip to the phase where that failure is the outcome. So folds
survive polls, section collapse and the sole-to-multi swap *within* a phase, and a new phase starts
fresh.

**D15 · The columns before the statement reserve their width.** The relative-time column has none,
and it is rendered only when it has a value (`SectionContent.tsx:55-59`). So `+0ms`, `+12ms` and
`+506ms` each push everything after them by a character, and the first row of a section — which has
no relative time at all — drops the column and jumps a further seven. Measured on the mockups, the
statement starts at three different x positions inside one section: 228px, 235px and 242px. The log
has always been ragged this way and nobody noticed, because ragged text still reads as text; a
column of identical chevrons beside it does not. So the span renders always, right-aligned, keeping
its `tabular-nums`, with a minimum width of seven characters — enough for every value below 100
seconds, and longer ones push as they do today.

That width is a named constant beside `ITEM_HEIGHT` and applied inline, not an arbitrary
`min-w-[7ch]` class. `SectionContent` already sets its cap that way
(`style={{ maxHeight: \`${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px\` }}`), so the measurement sits with
its neighbours and stays legible next to them. It does not belong in
`components/ui/styles.stylex.ts`: that file holds the shared control-size scale every primitive
reads, and `frontend/AGENTS.md` sends *repeated* measurements there — this one has a single
consumer, and filing a log-viewer column width in the design system would make it look like a token
other components should reach for.

A per-row minimum is not enough, and an earlier draft of this decision got that wrong twice. It
moved the index from `w-6` to `min-w-6` so four digits would not clip, and gave the relative time a
minimum of seven characters — but each row is its own flex container, so a minimum only sets a
floor. A section that reaches row 1000, or one whose run lasted long enough for `+1234.56s`, widens
*that row's* cell and pushes its chevron and statement right, while every other row stays where it
was. The result is the ragged edge this decision exists to remove, reintroduced by the fix for it,
and worst in D13's sparse window, where a four-digit failure sits directly beneath three-digit
neighbours.

So both columns are sized **once per section, from the widest value that section will actually
render**, and every row uses that width — not a minimum. `SectionContent` already knows the numbers
it is about to draw, including the real index D13 gives the marked row, so it measures them, writes
the two widths onto the scroll box as custom properties, and the cells read them. One write per
render, no per-row variation possible. The floors stay as floors: 24px for the index, seven
characters for the time, so a short section looks exactly as it does now.

The UX contract already asks for right-aligned numerics in tables; these are the same columns in a
different frame. This straightens the statement column too, which is the part of the fix that
improves the log as it stands today.

## States

Mockups A–E are in the PR description. Product typography, spacing and semantic colors are taken
from the live component.

| | State | What it settles |
|---|---|---|
| A | Today | The 80-character cut, a failed row that is error-only, and the ragged left edge of D15 |
| B | Folded, hover | D2, D3, D5, D6, D11, D15 — one meaning per control, reserved slots, a straight column |
| C | Unfolded | D8, D9, D10 — verbatim formatting, copy in the block, raised cap, one scrollbar |
| D | A failed command | D1, D3, D4, D10 — the error keeps the line, the failed statement and its copy sit beneath it |
| E | Narrow container | D2 — the same rows in the deploy sheet, clamped by its width |

## Scope

Frontend only. The viewer is embedded by `DatabaseChangelogDetailPage`, `RevisionDetailPanel`,
`DeployTaskRunHistorySheet` and `DeployLatestTaskRunInfo`; all four inherit the change.

| File | Change |
|---|---|
| `task-run-log/types.ts` | `statement?`, `error?` and `defaultOpen?` on `DisplayItem` |
| `task-run-log/model.ts` | Delete the `substring`; read the statement for failed commands too; return all the new fields; pick the auto-open row in `buildSectionsFromEntries`, over the whole entry sequence rather than per section; build `key` from the entry's identity instead of `idPrefix-groupIndex-entryIndex` (D14) |
| `task-run-log/useTaskRunLogSections.ts` | The hook owns every builder call — flat, per-replica, release-file and orphan — so it forwards `taskRunStatus` into all of them; nothing else invokes the builders, and a guard that stops here is a guard that never runs |
| `task-run-log/SectionContent.tsx` | Fold control, copy button, CSS clamp, default-open failed rows, the marked row rendered and scrolled to past the 50-item window, section cap, `ITEM_HEIGHT` 20 → 28, the reserved timestamp and index columns (D15), and one `ResizeObserver` on the scroll box deciding which rows are foldable (D5) |
| `task-run-log/TaskRunLogViewer.tsx` | The reader's fold overrides, held above the conditional mount and cleared with `taskRunName` (D14) |
| `locales/en-US.json` | Two accessible names |

Tests, none of which exist today — which is how #21276 passed a clean suite while changing the
behavior of this function:

- `model.test.ts`: a multi-line statement yields a collapsed `detail` and a verbatim `statement`; a
  statement past 80 characters is not truncated (regression); a failed command yields the error as
  `detail` *and* the failed statement in `statement`, including when it has to come from `range`;
  an entry with no statement yields `"-"` and no `statement`.
- `model.test.ts` again for the auto-open pick, which is where the retry shape has to be locked
  down: entries for two attempts separated by a `RETRY_INFO`, the first failing and the second
  succeeding, mark **no** row to open even though the failure is last in its own section; the same
  entries with the second attempt failing mark only the second failure; a failure under one replica
  is not silenced by another replica's success; and a `DONE` `taskRunStatus` marks nothing at all,
  whatever the entries say.
- `SectionContent`: a foldable row toggles and reports `aria-expanded`; a row marked to open starts
  unfolded and can be folded; a section of 60 entries whose marked failure is the last one renders
  that row without pressing *Load more*, still reports the hidden count, **numbers it 60, not 51**,
  and leaves the section scrolled to it rather than at the top (D13); copy receives the verbatim statement, never the line and never the error; a failed row
  carries no copy button on its error line and one inside its block; a row with no recoverable
  statement carries none at all; a single-line statement that fits its width has **no** fold
  control, the same row in a container narrow enough to clamp it has one, and a multi-line
  statement has one at any width (D5); a failed row whose statement is as short as `SELECT 1` is
  foldable anyway, and folding then unfolding it brings the statement and its copy button back; and
  a clamped one-line statement **keeps** its chevron after opening — drive an observer callback
  while it is open, which is what raising the cap does in the product, and assert the control
  survives and still closes the row; a marked failure with no recoverable statement is open yet
  carries no chevron and no row toggle.
- Collapsed ancestors (D14), same `datasetKey`: a failure arriving into a section the reader
  collapsed leaves it collapsed, and expanding it afterwards shows the marked row already unfolded
  and scrolled to.
- Clicking (D7): a click on the index, the timestamp or the row's empty space toggles; a click on
  the clamped line or inside the unfolded block does not, so a double-click selects a word without
  the row moving under it.
- `model.test.ts` for the marking edges: a failed command with neither `statement` nor a usable
  `range` is still marked, so D13 renders and scrolls to its error, and no cap change follows
  because it has no block.
- Column widths (D15): a section containing row 1000, and one containing a `+1234.56s`, put every
  row's fold control at the same x — the wide value sets the column for all of them, including in
  D13's sparse window where a four-digit index sits under three-digit neighbours.
- Row identity (D14): two `COMMAND_EXECUTE` entries from the same replica sharing a timestamp and
  carrying `statement` rather than `range` — the SDL shape — get distinct keys, so folding one
  leaves the other open; the same holds when the two sit in **different release files**, which is
  the case an ordinal computed inside the builder would break; and a row's key is unchanged by a
  second replica appearing or a retry marker splitting its section.
- Live updates (D14), all on an unchanged `datasetKey`, and including the regrouping case: a run
  that starts with one replica and gains a second keeps a folded row folded **and keeps that row's
  `key` byte-for-byte identical**, so nothing remounts; the same holds when a retry marker splits a
  section in two between polls. Asserting the key's stability is the point of the test — a version
  of it that tolerated a changing key would be asserting the bug.
- Live updates (D14), all on an unchanged `datasetKey`: a section rerendered with a newly marked
  failure opens it without remounting; a row the reader folded stays folded when the next poll
  arrives; a row whose mark moves away folds again if the reader never touched it; a folded row is
  **still folded after collapsing and reopening its enclosing section**, and after the sole-section
  rendering gives way to the multi-section tree; and `taskRunName` changing clears those toggles,
  as `datasetKey` already clears `showAllItems`.

## Not in this PR

The implementation, which follows separately. Also deliberately out:

- **A smaller shared control size.** The 8px the row gains is the cost of the shared size contract
  (D11). A 20px tier is a design-system change, not a log-viewer one.
- **The CockroachDB retry that is logged as a failure.** One `COMMAND_EXECUTE` and a response per
  attempt, of which the converter keeps the first, so a statement that succeeded on retry shows a
  red ✗ and an error in every task-run log today, and marks its whole section failed. It is a
  backend defect, not a viewer one — the successful response never reaches the API — and it wants
  either a `COMMAND_EXECUTE` per attempt, matching the Postgres shape, or a converter that replaces
  rather than drops. D4's `DONE` guard keeps this design from making it louder; it does not fix it.
- **Giving the deploy sheet's body the scroll.** It would remove the one nested scroll region the
  viewer sits in (D9), but it buys a second layout mode on a surface whose nesting is already the
  status quo. Worth revisiting if the sheet is where people actually read long statements.
- **Syntax highlighting in the unfolded block.** A Monaco instance per log row is far past what
  this surface can afford. The statement is mono, preformatted and copyable; the editor is one
  click away on the pages that embed the viewer.
- **The other entry types.** `SCHEMA_DUMP`, `DATABASE_SYNC` and friends carry status words, not
  payloads. They gain the new fields only if they ever carry one.
