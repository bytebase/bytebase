# Task-run log — a statement is a payload, not a line

Status: proposal · 2026-09-11 (revised 2026-09-21)

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
- Nothing starts unfolded. The row that explains the outcome is brought into view; opening it is
  the reader's move, and a toggle never moves the line that was clicked.

## Decisions

**D1 · Both payloads on the item.** `DisplayItem` (`types.ts`) grows two optional fields beside the
existing `detail: string`:

```ts
key: string;          // the entry's identity, not its position — see D14
detail: string;       // the text the row shows
statement?: string;   // the statement this row ran, as the sheet stored it
error?: string;       // the error the command returned
marked?: boolean      // the row D4 picked: the failure D13 brings into view
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
that came back partial — still gets marked, because the mark's job is to say *this is the row that
explains the outcome*, and D13 has to render and scroll to it or the error itself stays hidden
behind *Load more*. What it does not get is a fold control: there is no block to reveal, so D5
gives it nothing to open with.

`marked` is the fourth, and it exists because the view cannot work it out. D4's pick is made
across a whole execution context, and `SectionContent` sees one section at a time; nor can it infer
the pick from `error`, since every transient failed attempt carries one and surfacing the first of
them is the thing D4 exists to prevent. So `buildSectionsFromEntries` decides and marks the row it
chose, and D13's render window and scroll read that flag rather than re-deriving a judgement they
lack the inputs for.

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
copy button appears on it; that row's copy is in its block, one click on the row away. The error
gets none: wanting an error in the clipboard is rare next to wanting the SQL, and it does not
justify a second control on the densest surface in the product — it stays selectable text, as it
is today. A row whose statement could not be recovered at all (no `statement`, no usable `range`,
or a sheet that came back partial) gets no copy button rather than a fallback to something else,
so the control never means two things. One meaning, one place: copy is the SQL, beside the SQL.

**D4 · Nothing unfolds by default; the failure that explains the outcome is marked.** Every row
starts folded, the failed one included. An earlier revision opened the failed command by default,
on the reasoning that it is what the log was opened for, and in use that was wrong twice over. The
pages that embed the viewer already show the failed task's statement directly above the log, so the
block repeated it. And it made the reader's first act on a failed log a *fold* — of a row they had
never opened, in a box whose cap D8 had raised and whose scroll D13 had moved — which is exactly the
state in which a fold displaces the line being clicked (D16). The error is the news and it is
always on the line; the statement is one click away.

What the failure does get is the **mark**: D13 renders it past the *Load more* window and brings it
into view. But only the failure that explains the outcome, which is not the same as every failed
row. A run can hold several: on a lock timeout the
Postgres driver retries the whole command list up to `MaximumRetries`
(`backend/plugin/db/pg/pg.go:463-485`), re-logging every command on each attempt, so the same
statement can fail three times and the run still succeed. Marking all of them would bring the first
attempt into view, not the outcome.

So the rule is: **mark the last failed command row in an execution context, and only when no
command after it succeeded.** The context is the entry sequence `buildSectionsFromEntries` is
handed — one replica, one release file — and the test runs over that sequence *before* it is
grouped. It cannot be section-local: sections are typed groups, and `groupEntriesByType` starts a
new one whenever the entry type changes, while a retry emits a `RETRY_INFO` entry between attempts
and, in transaction mode, a rollback and a fresh begin as well. Every transient failure would
therefore be the last failure in a section of its own, and all of them would be marked. Scanned across
the context instead, a run that recovered has successful commands after its failure and marks
nothing, while a run that really failed ends at the failure and marks exactly that row. Per
context, not per run, so a failure on one replica is never silenced by another replica's success.
A live log is a prefix of the finished one, so the test also treats a failure followed by a
`RETRY_INFO` entry as not terminal. The driver writes that marker before it re-runs
(`backend/plugin/db/driver.go:357-367`, called at `pg.go:470`), so the marker is the standing
signal that another attempt is coming; without it, the five-second poll
(`useTaskRunLogData.ts:101`) landing between a failed attempt and its retry would mark a failure
that is about to be superseded. What remains is the ~200ms between the failure and the marker,
and that window is left open deliberately. A poll landing inside it — roughly one chance in
twenty-five, and only for someone watching a retrying run live — scrolls to a failure that the next
poll unmarks when the marker arrives, so the view can rest on a superseded attempt for up to five
seconds. Suppressing the mark while the run is still `RUNNING` would remove that and take the
mark's point with it, since the case the mark exists for *is* a deploy watched from
the plan page that fails while you are looking at it. The transient state is not false either: that
command really did fail at that moment; it is only about to be tried again.

The test reads only the entries, which matters because `taskRunStatus` is an optional prop that the
changelog and revision pages do not pass. One guard uses it where it exists: when a caller passes a
status of `DONE`, nothing is marked at all. That is cheap, it costs the live case nothing — that is a run
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
**already** reads as a failure today, before this design touches it. Scrolling to a wrong row makes
it more prominent; it does not make it wrong. The converter fix below closes it on all four surfaces at once,
which is why that is where it belongs.

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
statement has no newlines and its line, being the error, is never clamped — so the row would get
no control at all, and the statement and the copy button that lives with it, both inside the block,
could never be reached — on exactly the row whose SQL the reader wants. A row holding two payloads
is always foldable.

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
exception exists to let a reader close a block that is on screen, so it reaches exactly as far as a
block does: an override outlives the poll it was made on, and a row whose statement can no longer
be recovered has no block and gets no chevron. Beyond that it sounds like
a truism and is in fact the subtlest case here. A successful single-line statement qualifies only
by the clamp, and D10 has the block *replace* the line when it opens, so the thing the clamp was
measured on no longer exists. Worse, opening a row in a box still below its cap grows the box, which
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

**D7 · Clicking a row's line toggles it; the unfolded block does not.** Click-anywhere is what
makes the affordance usable at 12px, and it is the convention of every CI log, so on a row that has
a fold control the whole line toggles: the index, the timestamps, the glyph, the empty space, the
clamped statement and the error. One rule, the same on every row that has a chevron — a reader does
not have to learn which part of a line is live. The unfolded block is the exception. It is not a
line but the content the line opened: a reader reads and selects from it, it can be hundreds of
pixels tall, so a stray click would collapse what is being read, and the chevron sits beside it.

That split also settles the double-click. A selection guard catches a drag, where the selection is
already non-collapsed at mouse-up, but not a double-click, whose first click arrives collapsed and
is indistinguishable from a single one at that instant. On the block, where selecting an identifier
is the point and the first click would remove the very text being selected, exclusion answers it.
On the clamped line the first click expands the row, which is what a click there meant anyway, and
the word is still there in the block underneath.

The error line pays a small, accepted price for the uniform rule. It has no copy button (D3), so it
is taken by selecting it. A drag never toggles. A double- or triple-click selects as usual and also
flips the block once, and in a section scrolled to its end that fold can shift the text under the
pointer mid-gesture. D3 already judged wanting the error in the clipboard rare; a dead zone across
the widest part of the row that matters most is the larger cost.

So: clicks inside a button are ignored (D6 owns those), clicks inside the unfolded block are
ignored, a click that ends a selection is ignored, and everything else toggles. A row that is not
foldable (D5) has nothing to toggle, so its line is inert — including the error of a failure whose
statement could not be recovered. Mouse convenience only — assistive technology sees D6's controls.

**D8 · The section's height never changes because a row unfolds.** The scroll box is capped at
`ITEM_HEIGHT * MAX_VISIBLE_ITEMS` — ten rows, 280px — whether or not a block is open, and a block
taller than what remains scrolls inside it. An earlier revision raised the cap to `max(280px, 60vh)`
while a block was on screen, so that a 630px statement would not be read through a keyhole. In use
the cost was worse than the keyhole: a full box grew by 260px the moment a row was clicked, moving
every section below it, and shrank again on the fold, which is where the page and the box's own
scroll got clamped out from under the reader (D16). Layout stability wins: the box the reader
scanned is the box they unfold in. Two things follow. The block's copy button is sticky to the top
of the box while the block is in view (D10), so a statement taller than the box can still be
copied from wherever the reader has scrolled to. And unfolding a row near the bottom of the box has
to reveal what it opened, since the block lands below the box's visible edge — that is D16's job.

**D9 · The viewer owns one scroll context.** The unfolded block never gets its own `overflow` — it
grows to its natural height and the section scrolls, so nothing the viewer renders stacks a
scrollbar inside a scrollbar. It cannot claim the same for the page it sits on:
`DeployTaskRunHistorySheet` renders the viewer inside `SheetBody`, which is `overflow-y-auto`
(`components/ui/sheet.tsx:161-167`), so on that one surface the section's box has always sat inside
an outer scroller, and an unfolded block now scrolls inside it. The nesting predates this
change and the sheet's `overscroll-contain` keeps it from chaining. Handing the scroll to the host
— no cap when the viewer is inside a sheet, `SheetBody` scrolling the expanded statement — would
remove it, at the price of a second layout mode for the viewer to carry, test, and keep consistent
with `MAX_RENDERED_ITEMS`. Recorded as the option and not taken: the nesting is pre-existing and
mild, the mode would be permanent.

**D10 · The unfolded block takes the line's place, or sits under it.** Same cell, same left edge;
`whitespace-pre-wrap break-words` on the verbatim statement, in the row's mono face, on
`bg-background` inside a `border-block-border` block — the block is a framed content region, and
`docs/agents/frontend-ux.md:143-144` keeps `border-control-border` for controls — with D3's copy
button beside it, in a column of its own at the block's right edge. A column rather than an
overlay because the contract says text "must not overlap adjacent controls"
(`frontend-ux.md:113-114`), and the rows D5 newly makes foldable are precisely the ones whose first
wrapped line runs the full width of the block. The button is `sticky` to the top of the section's
scroll box: D8 keeps the box at ten rows, so a long statement scrolls under it, and a control pinned
to the block's top corner would be gone before the reader reached the end of what they want to
copy. It stays put for as long as any of the block is in view, and leaves with the block. On a successful row the block replaces the line, so one
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

**D13 · The marked row has to be rendered, numbered and in view.** `SectionContent`
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
the marked row, inside a box capped at 280px whose `scrollTop` starts at zero, so the
row would be mounted and off-screen — the promise kept in the DOM and broken on the
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

**D14 · The reader's folds are the only open state, and they outlive everything but the run.** The
viewer polls a running task every five seconds (`useTaskRunLogData.ts:101`), and `section.items`
changes without `datasetKey` changing. A row is open when the reader opened it and folded
otherwise; nothing else writes that state, so a poll can neither open a row nor close one. The mark
rides the same polls but touches only D13: a failure arriving on a later poll into an
already-mounted section is rendered and scrolled to then, not only at mount, and a mark that moves
off a row — a transient failure that turns out to have been retried — takes nothing with it,
because it never opened anything.

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
entry type arrives. Either remount would discard the reader's folds on a dataset that never
changed. The overrides therefore sit in `TaskRunLogViewer`, above the
conditional mount, keyed by row and cleared when `taskRunName` changes — the same moment
`datasetKey` already clears `showAllItems`. `showAllItems` keeps its remount-local behavior;
re-hiding the tail of a long list on collapse is not a promise anyone made.

The mark stops at a collapsed ancestor, deliberately. The viewer already expands a section the
moment its status turns to error — unless the reader collapsed that section themselves, which
`userCollapsedSections` records and honours (`useTaskRunLogSections.ts:234-247`), with the same
treatment for replicas and release files. A mark arriving into a section the reader has shut does
not reopen it. Doing so would override the reader at the one moment they have most clearly said
what they want on screen. Nothing is hidden by this: a collapsed section still
carries its own status, so the header turns to the error glyph and says a failure is inside. When
the reader opens it, the section is already scrolled to the marked row,
because D13's scroll fires on mount whenever a marked row is present, not only when a mark lands on
a row that is already showing.

The promise stops at the phase boundary too, deliberately. Both deploy surfaces key the viewer on the
run's status (`` key={`logs-${taskRun.name}-${taskRun.status}`} ``), so a `RUNNING` → terminal flip
remounts the whole viewer and takes the overrides with it. That key is not an accident — the
comment above it says the remount exists "for a fresh disclosure state on the new phase", and it is
also what invalidates an in-flight `RUNNING` log request before it can be cached as complete.
Persisting overrides across it, in a module-level map keyed by task-run name, would work and would
quietly undo that intent. A new phase is also a fair place to start clean: the terminal log is a
different document from the one that was streaming. So folds
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

**D16 · A toggle never moves the clicked line except to reveal what it opened, and folding puts it
back.** With D8's cap fixed, a toggle changes only the box's scroll range: unfolding lengthens it,
folding shortens it back to what it was. Two motions follow, and both are the reader's doing.

Unfolding reveals. The marked row is usually the last one, and D13 parks it at the bottom of the
box, so its block opens below the visible edge; a click that visibly changed nothing but the
chevron would read as a click that did nothing. So the section scrolls itself by the least that
shows the row: to its bottom edge when the row fits, and to its top edge — the line at the top of
the box, the block filling the rest — when the row is taller than the box. The section's own
scroll, never the page's: the block is clipped by the box, and a page scroll cannot reveal what is
inside it, so scrolling the page would only move the log under the reader.

Folding puts it back. The folded content is exactly the content before the unfold, so the box's
scroll range is exactly what it was, and the browser's own clamp lands the line where it stood
before the reveal — under the pointer when the reveal moved nothing, and back where it was clicked
from when it did. Nothing compensates for this: a page scroll that kept the line under the pointer
would slide the whole log up the page instead, which is the jump the fixed cap exists to remove.
Row 38 of 38, at the bottom of a full box, goes to the top on unfold and comes back to the bottom
on fold; the page does not move for either.

An earlier revision compensated with the section's and then the page's scroll after every toggle.
All three displacements it corrected — the cap dropping around the row, the section's scroll
clamping when the cap rose, the page clamping when the box shrank on a full log — were caused by
the cap moving, and left with it.

One case is left, and accepted: a box still below its cap grows on unfold and shrinks on fold, and
a reader who scrolls the page into that growth and then folds is clamped by the page, the way every
disclosure at the bottom of a page is. Holding the line there would mean reserving the vacated
height as blank space, state lingering on screen after the click that caused it. Recorded as the
option and not taken.

## States

Mockups A–E are in the PR description. Product typography, spacing and semantic colors are taken
from the live component.

| | State | What it settles |
|---|---|---|
| A | Today | The 80-character cut, a failed row that is error-only, and the ragged left edge of D15 |
| B | Folded, hover | D2, D3, D5, D6, D11, D15 — one meaning per control, reserved slots, a straight column |
| C | Unfolded | D9, D10 — verbatim formatting, copy in the block; the mockup's raised cap predates D8's revision, the box stays at ten rows |
| D | A failed command, unfolded | D1, D3, D10 — the error keeps the line, the failed statement and its copy sit beneath it |
| E | Narrow container | D2 — the same rows in the deploy sheet, clamped by its width |

## Scope

Frontend only. The viewer is embedded by `DatabaseChangelogDetailPage`, `RevisionDetailPanel`,
`DeployTaskRunHistorySheet` and `DeployLatestTaskRunInfo`; all four inherit the change.

| File | Change |
|---|---|
| `task-run-log/types.ts` | `statement?`, `error?` and `marked?` on `DisplayItem` |
| `task-run-log/model.ts` | Delete the `substring`; read the statement for failed commands too; return all the new fields; pick the marked row in `buildSectionsFromEntries`, over the whole entry sequence rather than per section; build `key` from the entry's identity instead of `idPrefix-groupIndex-entryIndex` (D14) |
| `task-run-log/useTaskRunLogSections.ts` | The hook owns every builder call — flat, per-replica, release-file and orphan — so it forwards `taskRunStatus` into all of them; nothing else invokes the builders, and a guard that stops here is a guard that never runs |
| `task-run-log/SectionContent.tsx` | Fold control, copy button, CSS clamp, the marked row rendered and scrolled to past the 50-item window, section cap, `ITEM_HEIGHT` 20 → 28, the reserved timestamp and index columns (D15), one `ResizeObserver` on the scroll box deciding which rows are foldable (D5), and the reveal on unfold (D16) |
| `task-run-log/TaskRunLogViewer.tsx` | The reader's fold overrides, held above the conditional mount and cleared with `taskRunName` (D14) |
| `locales/en-US.json` | Two accessible names |

Tests, none of which exist today — which is how #21276 passed a clean suite while changing the
behavior of this function:

- `model.test.ts`: a multi-line statement yields a collapsed `detail` and a verbatim `statement`; a
  statement past 80 characters is not truncated (regression); a failed command yields the error as
  `detail` *and* the failed statement in `statement`, including when it has to come from `range`;
  an entry with no statement yields `"-"` and no `statement`.
- `model.test.ts` again for the mark, which is where the retry shape has to be locked
  down: entries for two attempts separated by a `RETRY_INFO`, the first failing and the second
  succeeding, mark **no** row even though the failure is last in its own section; the same
  entries with the second attempt failing mark only the second failure; a failure under one replica
  is not silenced by another replica's success; and a `DONE` `taskRunStatus` marks nothing at all,
  whatever the entries say.
- `SectionContent`: a foldable row toggles and reports `aria-expanded`; a marked row starts
  folded like any other; a section of 60 entries whose marked failure is the last one renders
  that row without pressing *Load more*, still reports the hidden count, **numbers it 60, not 51**,
  and leaves the section scrolled to it rather than at the top (D13); copy receives the verbatim statement, never the line and never the error; a failed row
  carries no copy button on its error line and one inside its block; a row with no recoverable
  statement carries none at all; a single-line statement that fits its width has **no** fold
  control, the same row in a container narrow enough to clamp it has one, and a multi-line
  statement has one at any width (D5); a failed row whose statement is as short as `SELECT 1` is
  foldable anyway, and folding then unfolding it brings the statement and its copy button back; and
  a clamped one-line statement **keeps** its chevron after opening — drive an observer callback
  while it is open, which is what raising the cap does in the product, and assert the control
  survives and still closes the row; a marked failure with no recoverable statement
  carries no chevron and no row toggle.
- Collapsed ancestors (D14), same `datasetKey`: a failure arriving into a section the reader
  collapsed leaves it collapsed, and expanding it afterwards shows the section already scrolled to
  the marked row, which is still folded.
- Clicking (D7): on a foldable row a click on the index, the timestamp, the empty space, the clamped
  line or the error line toggles; a click inside the unfolded block does not, so an identifier can
  be double-clicked there without the row moving; a click that ends a drag across the clamped line
  or the error does not toggle; the line of a row that is not foldable does nothing, the error of a
  failure with no recoverable statement included; and two clicks in quick succession on the clamped
  line leave the row open, because the second lands on the block.
- A fixed box (D8): the scroll box's cap is 280px before, during and after a row is unfolded, and
  the block's copy button is a sticky sibling of the SQL rather than an overlay on it.
- Revealing and putting back (D16): with the row's geometry stubbed, unfolding a row whose block
  already fits scrolls nothing; unfolding one at the bottom of the box scrolls the section by
  exactly the block's overflow, so the line moves up by that and no more; unfolding a row taller
  than the box puts its line at the box's top; the page is never scrolled by either; folding
  scrolls nothing itself, and a toggle in one section leaves another section's scroll alone. jsdom
  has no layout, so the real motion is checked in a browser and in the journey below: a full
  box keeps its height through unfold and fold, the last row's line goes to the top and comes
  back, the page does not move, and the sticky copy button is still in view — and still copies the
  whole statement — after the box has been scrolled to the end of a block taller than itself.
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
  failure scrolls to it without remounting and does not open it; a row the reader unfolded stays
  unfolded when the next poll arrives, and when its mark moves away; an unfolded row is
  **still unfolded after collapsing and reopening its enclosing section**, and after the sole-section
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
