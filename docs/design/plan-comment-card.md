# Plan comment card

Status: proposal · 2026-11-10

On a plan, people and the automated reviewers comment on lines of SQL. A comment thread shows in
two places: under its lines in the statement editor of the Changes section, and as an entry in the
Activity timeline. A comment is auxiliary information about the statement. The statement is what
gets reviewed, approved and run, and a comment helps someone judge it. Today the page marks a
comment with its strongest signals instead: the commented lines are tinted, a folded thread is a
dense status row, and the box is as wide as the statement. The two places also draw the thread
differently. The rule this doc lands on: **a comment thread is one quiet card in both places, with
no color on the SQL, a title bar that names its lines, and the same folded form everywhere.**

This is the first of three docs, each in its own PR. The second, Plan comment SQL excerpt,
covers the SQL shown inside an Activity card. The third, Plan review findings group, covers how
findings sit under the review that posted them.

## Problem

Internal testing of SQL review reported that inline comments do not read as comments (BYT-10500),
that the line background colors are confusing (BYT-10499), and that several reviews on one line
overwhelm the statement (BYT-10373). The causes are in how the thread is drawn, under
`frontend/src/routes/project/plan-detail/components/threads/`:

- **Color sits on the SQL.** `StatementThreadsLayer.tsx` paints every line an open thread covers:
  `bb-thread-line--passive` (grey) for a comment from a person, `--attention` (amber) for a P1 or
  P2 finding, `--blocking` (red) for a P0. Unfolding a thread repaints its lines with
  `bb-thread-line` (accent), so a finding's color changes when it is opened.
- **The box reads as part of the statement.** The view zone starts at the code column and is
  padded `py-2 pr-2`, so the card is as wide as the code and starts where the code starts. It has
  no title; its first row is the author.
- **Only color says which lines a thread is on.** `ThreadStack` shows a range label only when two
  threads ending on one line cover different ranges (`showRanges`).
- **A folded thread is dense.** `CollapsedThreadRow` shows the author, a priority badge, a summary,
  the reply count, the time and the state.
- **Comment text has no structure.** `ThreadComment` puts the avatar in the header row, so the
  avatar, the priority badge and the text share one left edge. Replies differ from the first
  comment only by a `bg-control-bg/50` band.
- **Two places, two cards.** The Activity card takes its header from `StatementAnchorContext`. It
  can be folded only when it is resolved, and its folded row (`ThreadSummary`, a second
  implementation of the collapsed row) shows the author and a summary but not where in the SQL the
  thread is.

## Principle

> A comment is auxiliary information about the statement. Mark it with structure and words, never
> with color on the SQL, and draw it the same way wherever it appears.

GitHub's review thread is the reference where it has an answer: a bordered box of limited width, a
title bar that names the lines and holds the fold control, a quiet "Resolved" label, and indented
replies. Where this design differs from it, the decision says why.

## Decisions

**D1 · No color on commented lines.** The four at-rest tints go: grey, amber, red, and the accent
tint of an unfolded thread. The margin marker stays as the only mark on the statement at rest. The
highlight while a reader selects lines for a new comment (`bb-thread-line--selecting`) stays,
because it is feedback on an action and ends with it. Its color changes from `--color-info`, a
blue, to the neutral `--color-control-bg-hover`, so that no comment state paints the SQL in a hue. Severity stays where it
already is, on the P0, P1 or P2 badge inside the comment. Rejected: tinting only P1 lines, as
BYT-10499 asks. Severity would still be painted on the SQL, and the reader would still have to
learn a color code. Rejected: one neutral tint for every commented range. It still paints the
statement, and D3 names the lines in words.

**D2 · The box has a width limit.** The thread stack is capped at `max-w-3xl` (48rem), and the view
zone's right padding grows from `pr-2` to `pr-4`, so on a narrow editor the card keeps 16px clear
of the right edge. The border (`border-block-border`) and corners (`rounded-sm`) stay. `shadow-sm`,
which only the Changes section passes to the card today, becomes part of the card, so the Activity
card gains it; the Activity card keeps its full width. Rejected: keeping the full width. A box as wide as the code reads as a band of the statement.

**D3 · A title bar names the lines.** Every card starts with a bar on `bg-control-bg`, in 12/16
text. In the Changes section it reads "Comment on line 1" or "Comment on lines 1–7". In Activity it
shows the change the thread belongs to on the left, and the line range followed by "View in
Statement" on the right, which is the order the Activity header has today. The bar's text is
`text-control`, not `text-control-light`: light text on `bg-control-bg` is 4.4:1, under the 4.5:1
minimum. The new-comment composer already has a bar of this kind ("Add a comment on lines 9 to
10"), so a thread keeps one look from draft to posted. Rejected: the range label that appears only
when ranges differ. With D1 it would leave most threads with nothing saying which lines they are
on.

**D4 · A folded thread is its title bar.** Folding hides the body and changes nothing else. The bar
is the same folded and unfolded, so nothing moves sideways when a thread opens.
`CollapsedThreadRow` and `ThreadSummary` are both deleted. A folded finding no longer shows its
priority. Rejected: keeping the priority badge on the folded bar. The Automated review section
above the statement already counts open findings by priority, and one colored badge per folded
thread is the density this doc removes.

**D5 · Any thread can be folded.** The fold control is an icon-only `Button size="xs"` at the left
of the bar, with `aria-expanded`, named by the bar's title. It is the only part of the bar that
folds, because in Activity the bar also holds a link. In Activity an open thread starts unfolded
and a resolved one starts folded, as today, and the reader can now fold an open one. In the
Changes section the control replaces the close button in the author row, and the default is
unchanged: the first unresolved thread is unfolded and every other thread is a margin marker until
clicked (`defaultExpandedThread`). Rejected: folding only resolved threads. A long open finding
cannot be put away while reading the rest of the timeline.

Who owns the fold does not change. In the Changes section it is `StatementThreadsLayer`: it keeps
`expandedRoots`, `expandThread` also selects the thread in the stepper and scrolls it into view,
and `collapseThread` also clears that selection. The card is controlled there: the bar's fold
control calls those two functions and the card keeps no fold state of its own. In Activity the card
owns the fold, as it does today. Resolving keeps its rules too: in the Changes section only the
reader's own Resolve folds the card, and in Activity a thread folds when it becomes resolved.

**D6 · State is a quiet text label in the bar.** "Resolved" moves from the footer into the bar and
keeps the form `ThreadStateBadge` has today: an icon and text, no fill. "Outdated" and "Statement
unavailable", which Activity shows when a thread's lines have changed or cannot be loaded, take the
same form in the same place. Today "Outdated" is an amber badge, which D1 would leave as the only
strong color in the card. Rejected: "Resolved by" with the name. `IssueComment` records
`thread_state` and nothing about who set it, so the name needs a backend field first.

**D7 · Comment text sits in its own column.** `ThreadComment` becomes a two-column grid: a 20px
avatar column, then the author row, the "Applies to" row, the priority and title, and the text.
Replies are indented by that column and joined to the first comment by a vertical line. The
`bg-control-bg/50` band behind replies goes, since the indent now sets them apart. Rejected:
GitHub's current layout, which indents replies only and keeps the first comment's text flush with
its avatar. A finding stacks four rows, and they scan better with the avatar out of the way.

**D8 · The composer is the same card.** `InlineThreadComposer` keeps its title ("Add a comment on
lines 9 to 10") in the bar, and its avatar moves from the bar into the body's avatar column beside
the editor.

**D9 · The "Applies to" row is unchanged.** A finding that applies to only some of a change's
databases lists them under the author row, as today (`ReviewTargets`). The bar does not name or
count them: an open finding is unfolded, so the row is on screen, and a resolved one only needs to
say it is resolved. Rejected: a count or the database names in the bar. The bar stops being simple
and long lists do not fit.

## States

| State | Changes section | Activity |
| --- | --- | --- |
| Open, lines in the current statement | Margin marker; the first open thread is unfolded | Card, unfolded |
| Resolved, lines in the current statement | Margin marker in grey; folded bar with "Resolved" once its line is opened | Card, folded, "Resolved" |
| Several threads ending on one line | Marker with a count; one card per thread, open ones first | One entry per thread |
| Outdated: the commented lines changed | Not shown | Card with "Outdated" and no link into the statement |
| Statement unavailable | Not shown | Card with "Statement unavailable" |
| Thread on a deleted change | Not shown | Card titled "Deleted change" |
| More than four replies | Middle replies fold behind "N replies hidden · Show all" | Same |
| Reply editor open | Editor with the "Resolve thread" checkbox, in the footer | Same |
| Reader cannot reply or resolve | No footer | No footer |
| New comment | Selected lines highlighted in grey; composer card | Does not apply |

Which threads the Changes section shows is unchanged and is decided by placement
(`selectEditorThreads` in `threadModel.ts`): a thread shows there only while its lines are in the
current statement.

## Scope

Frontend only. Four PRs, in order. Each leaves main shippable. Every class, token or string that a
decision changes is named in one slice below, with the test that pins it, and so is the owner of
any state a slice rewires.

1. **Title bar and folding (D2 shadow, D3, D4, D5, D6).**
   - `CommentThreadCard.tsx`: add the bar on `bg-control-bg` with `text-control` text; add the fold
     control and drop the `thread.resolved` condition on folding; delete `ThreadSummary`; move
     `ThreadStateBadge` from the footer into the bar; take `shadow-sm` into the card's own classes.
     The fold takes an optional controlled pair, `expanded` and `onExpandedChange`, which replaces
     `onClose`. Without it the card keeps its own state, which is what Activity uses.
   - `ThreadStack.tsx`: delete `CollapsedThreadRow` and `showRanges`, and stop passing
     `shadow-sm`. Every thread renders the card, folded or not, with `expanded` from
     `expandedRoots` and `onExpandedChange` calling the layer's `onExpand` and `onCollapse`.
   - `StatementAnchorContext.tsx`: its header becomes the bar's Activity content, and
     `AnchorStatePill` renders "Outdated" and "Statement unavailable" as text labels in place of
     `Badge variant="warning"` and `variant="default"`.
   - The five locale files, for the two title strings.
   - Tests: `CommentThreadCard.test.tsx` (an open thread folds and unfolds; the bar is the same in
     both states; "Resolved" renders in the bar; a controlled card does not fold on its own),
     `ThreadStack.test.tsx` (a folded thread renders the card's bar and no `thread-row`),
     `StatementThreadsLayer.test.tsx` (folding from the bar removes the thread from
     `expandedRoots` and clears the stepper's selection; unfolding from the bar adds it and selects
     it), `StatementAnchorContext.test.tsx` (no `Badge` for the
     outdated and unavailable states). E2E: `frontend/tests/e2e/plan-detail/plan-detail.page.ts`
     reads `thread-row`, which this slice removes, and `plan-detail-review.spec.ts` asserts the
     folded and resolved states.
2. **No line tints, and the width limit (D1, D2).** Depends on slice 1, which gives every thread a
   title that names its lines.
   - `StatementThreadsLayer.tsx`: delete `commentedLines` and `LINE_CLASS_BY_WEIGHT`; delete the
     `bb-thread-line` paint over an unfolded thread's range; keep the two
     `bb-thread-line--selecting` paints. Both view-zone wrappers, the thread stack's and the
     composer's, go from `py-2 pr-2` to `py-2 pr-4` with `max-w-3xl`.
   - `statementThreads.css`: delete the `--passive`, `--attention`, `--blocking` and
     `bb-thread-line` rules; `bb-thread-line--selecting` changes from `--color-info` to
     `--color-control-bg-hover`.
   - Tests: `StatementThreadsLayer.test.tsx` (no decoration carries `bb-thread-line`,
     `--passive`, `--attention` or `--blocking` for an open or an unfolded thread; a pending
     selection and an open composer still carry `--selecting`). E2E: `plan-detail-review.spec.ts`
     asserts that a selected line's computed background is the grey token.
3. **Comment column and replies (D7).**
   - `ThreadComment.tsx`: the two-column grid.
   - `CommentThreadCard.tsx` and `CommentThreadCard.css`: indent the replies, draw the joining
     line, and drop `bg-control-bg/50` from the replies block.
   - Tests: `ThreadComment.test.tsx`, `CommentThreadCard.test.tsx`.
4. **Composer (D8).**
   - `InlineThreadComposer.tsx`: the bar goes from `bg-control-bg/50` with `text-sm` to the card's
     bar, and the avatar moves into the body.

## Not in this PR

- The SQL excerpt inside an Activity card, which the Plan comment SQL excerpt doc covers.
- Findings grouped under their review in Activity, which the Plan review findings group doc
  covers.
- Opening a long "Applies to" list in a popover (BYT-10512), ordering it by environment
  (BYT-10513), and a label in the bar for a comment that involves a production database
  (BYT-10514).
- Naming who resolved a thread. It needs a field on the thread first.
- A title for a thread that is not on lines. The API allows one; the comment box in Activity posts
  a plain comment, so the product does not create one today.
- Highlighting a thread's lines while the pointer is on its title bar.
- The thread stepper (`StatementThreadWalker`) and the margin markers, which do not change.
