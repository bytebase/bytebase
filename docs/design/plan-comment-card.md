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

## The card

One component draws a comment thread in both places. It has four parts, top to bottom.

| Part | What it is |
| --- | --- |
| Frame | The box: `border-block-border`, `rounded-sm` and `shadow-sm` on `bg-background` |
| Bar | One line on `bg-control-bg` in 12/16 `text-control`: the fold control, a title, state labels and, in Activity, the lines and a link |
| Body | In Activity the SQL excerpt, then the first comment and its replies |
| Footer | The reply field and the Resolve or Reopen button |

What a part shows depends on a few facts about the thread, and the code already has each of them:
the place the card is drawn in, what the thread is on, whether its lines are still in the
statement, whether it is resolved, what the reader may do, and whether it is folded. The sections
below give each part's content from those facts. Together they cover every card the product can
draw, so a case is never decided one at a time.

### Invariants

These hold for every card, in both places and in every state. A change that breaks one is wrong
even where it follows the tables.

1. **No thread paints the SQL.** Nothing about a thread's existence, state or priority colors a
   line. The one highlight is the reader's own selection while writing a new comment. It is
   neutral grey and ends with the selection.
2. **Every card has the frame and a bar.** That includes the new-comment composer and a thread that
   is not on lines.
3. **The bar is the same folded and unfolded.** Folding hides the body and the footer and changes
   nothing else. So a folded card says where a thread is and what state it is in, and nothing about
   what it says: not its author, its priority, its text, its replies or the databases it applies
   to. Each of those is one click away.
4. **The fold control is named by the bar's text.** Its accessible name is the title, followed by
   the lines where the bar shows them apart from the title. Whatever the title is, the control has
   a name.
5. **State is words in the bar.** "Resolved", "Outdated" and "Statement unavailable" are text with
   an icon, never a filled badge.
6. **The footer is there only when the reader can act.** A reader who can neither reply nor resolve
   sees none.
7. **A place adds content, not a layout.** The Changes section and Activity fill the same bar and
   body differently. Neither draws a different card.

### Title

| Place | The thread is on | Title | Name of the fold control |
| --- | --- | --- | --- |
| Changes section | Lines | "Comment on line 3" or "Comment on lines 1–7" | The title |
| Changes section | A draft, in the composer | "Add a comment on line 3" or "Add a comment on lines 9 to 10", as today | No fold control |
| Activity | Lines of a change | The change's label, as its tab in the Changes section shows it | The label, then the lines: "alpha_prod, lines 9–10" |
| Activity | Lines of a change that was deleted | "Deleted change", as today | "Deleted change, lines 9–10" |
| Activity | No lines | "Comment on the plan" | The title |

The Changes section never draws the last two rows. It shows a thread only while its lines are in
the current statement, which `selectEditorThreads` decides and this doc does not change.

### State labels, lines and link

| Fact | The bar shows |
| --- | --- |
| The thread is resolved | "Resolved" |
| Its lines changed after the comment (anchor state `OUTDATED`) | "Outdated" |
| The recorded statement cannot be placed, or the change was deleted (`UNAVAILABLE`) | "Statement unavailable" |
| In Activity, the thread is on lines | The line range, on the right |
| In Activity, its lines are still in the current statement (`CURRENT`) | "View in Statement", after the lines |
| The anchor state is not known yet (`PENDING`) | Neither "Outdated", "Statement unavailable" nor the link, until it settles |

Labels combine: a thread that is outdated and resolved shows both. In the Changes section only
"Resolved" can appear, because a thread drawn there has its lines in the statement.

### Body and footer

| Part | Shows |
| --- | --- |
| Excerpt | In Activity, for a thread on lines, once the recorded sheet has loaded. The Changes section has none, since the statement is directly above the card |
| First comment | An avatar column, then the author row, the "Applies to" row when a finding applies to only some of the change's databases, the priority and title of a finding, and the text |
| Replies | Indented by the avatar column and joined to the first comment by a line. With more than four, the middle ones fold behind "N replies hidden · Show all", as today |
| Footer | Only when the reader can reply or resolve: the reply field if they can reply, and Resolve or Reopen if they can resolve. The open reply editor keeps its "Resolve thread" checkbox |

### Folding

| Place | Starts | Changes when | Who keeps the state |
| --- | --- | --- | --- |
| Changes section | The first unresolved thread is unfolded. Every other thread is a margin marker. Clicking a marker opens its line with one thread unfolded and the line's other threads folded | The reader folds or unfolds it, or resolves it themselves, which folds it | `StatementThreadsLayer`, as today. Unfolding also selects the thread in the stepper and scrolls to it, and folding clears that selection. The card reports the click and keeps no state |
| Activity | Unfolded when open, folded when resolved | The reader folds or unfolds it. It folds when it becomes resolved and unfolds when it is reopened | The card, as today |

### On the statement

| What | In this design |
| --- | --- |
| Margin marker | Unchanged. It is the only mark on the statement at rest |
| Line tint for a thread | None, by invariant 1 |
| Highlight of the reader's selection | Neutral grey, while selecting lines or writing the comment |
| Width of a card | At most 48rem, and 16px clear of the editor's right edge |

## Decisions

**D1 · No color on commented lines.** The four at-rest tints go: grey, amber, red, and the accent
tint of an unfolded thread. The highlight of the reader's selection (`bb-thread-line--selecting`)
stays, because it is feedback on an action and ends with it. Its color changes from
`--color-info`, a blue, to the neutral `--color-control-bg-hover`. Severity stays where it already
is, on the P0, P1 or P2 badge inside the comment. Rejected: tinting only P1 lines, as BYT-10499
asks. Severity would still be painted on the SQL, and the reader would still have to learn a color
code. Rejected: one neutral tint for every commented range. It still paints the statement, and the
title names the lines in words.

**D2 · The box has a width limit.** In the Changes section the card is capped at `max-w-3xl`
(48rem) and keeps 16px clear of the right edge, where today it is 8px. `shadow-sm`, which only the
Changes section gives the card today, becomes part of the frame, so the Activity card and the
composer gain it. The Activity card keeps its full width. Rejected: keeping the full width in the
Changes section. A box as wide as the code reads as a band of the statement.

**D3 · A title bar names the lines.** Every card starts with a bar, and its title says what the
thread is on. The bar's text is `text-control`, not `text-control-light`: light text on
`bg-control-bg` is 4.4:1, under the 4.5:1 minimum. In Activity the lines stay on the right, before
"View in Statement", which is the order the Activity header has today. Rejected: the range label
that appears only when two threads' ranges differ. With D1 it would leave most threads with
nothing saying which lines they are on.

**D4 · A folded thread is its title bar.** `CollapsedThreadRow` and `ThreadSummary`, the two folded
rows of today, are both deleted. A folded finding no longer shows its priority. Rejected: keeping
the priority badge on the folded bar. The Automated review section above the statement already
counts open findings by priority, and one colored badge per folded thread is the density this doc
removes.

**D5 · Any thread can be folded.** The fold control is an icon-only `Button size="xs"` at the left
of the bar, with `aria-expanded`. It is the only part of the bar that folds, because in Activity
the bar also holds a link. In the Changes section it replaces the close button in the author row.
Rejected: folding only resolved threads. A long open finding cannot be put away while reading the
rest of the timeline.

**D6 · State is a quiet text label in the bar.** "Resolved" moves from the footer into the bar and
keeps the form `ThreadStateBadge` has today: an icon and text, no fill. "Outdated" and "Statement
unavailable" take the same form in the same place. Today "Outdated" is an amber badge, which D1
would leave as the only strong color in the card. Rejected: "Resolved by" with the name.
`IssueComment` records `thread_state` and nothing about who set it, so the name needs a backend
field first.

**D7 · Comment text sits in its own column.** The avatar gets a 20px column, and the author row,
the "Applies to" row, the priority and title, and the text sit beside it. The
`bg-control-bg/50` band behind replies goes, since the indent now sets them apart. Rejected:
GitHub's current layout, which indents replies only and keeps the first comment's text flush with
its avatar. A finding stacks four rows, and they scan better with the avatar out of the way.

**D8 · The composer is the same card.** It has the frame and a bar, by invariant 2. Its avatar
moves from the bar into the body's avatar column beside the editor, so a comment does not change
shape when it is posted.

**D9 · The "Applies to" row is unchanged, and stays in the body.** The bar does not name or count
a finding's databases. They are part of what the finding says, like its priority in D4, so by
invariant 3 they show when the card is unfolded and are hidden when it is folded, whether the
thread is open or resolved. Rejected: a count or the database names in the bar. The bar stops
being simple and long lists do not fit. BYT-10514 tracks the one signal worth adding there later,
a label for a comment that involves a production database.

**D10 · A thread that is not on lines has a bar too.** A thread is on lines when it has a
statement anchor, and the anchor is also what names its change. A thread without one can come
from an API client, or from the rule review for a syntax error that has no position, which the
parsers do not report today. The product's own comment box does not create one. Whoever posted
it, it is titled "Comment on the plan", because the card has no change to name. So invariant 2
has no exception and the card has one layout. Rejected: keeping today's card without a bar for it.
Every other rule would then need a second version. Rejected: never folding it. A resolved one
could not be put away.

## Scope

Frontend only. Five PRs. Each leaves main shippable, names what it depends on, and is opened after
the one before it merges.

A slice is finished when the rows it names hold and no invariant is broken. This plan names each
slice's files so its size can be judged, and its tests by the rows they assert. Which props and
helpers carry the rules is decided in the slice's own PR and reviewed there.

1. **Bar and folding in the Changes section (D3, D4, D5, and "Resolved" from D6).** Depends on
   nothing.
   - Delivers, for a card in the Changes section: the first row of Title; "Resolved" from State
     labels; the Footer row; the Changes section row of Folding; invariants 3, 4, 5 and 6.
     `CollapsedThreadRow` goes. Until slice 2 the Activity card is untouched.
   - Files: `CommentThreadCard.tsx`, `ThreadStack.tsx`, `StatementThreadsLayer.tsx`, and the five
     locale files for the title.
   - Tests: `CommentThreadCard.test.tsx`, `ThreadStack.test.tsx` and
     `StatementThreadsLayer.test.tsx`, one case for each row delivered. They include: the bar is
     the same folded and unfolded, and a folded card shows nothing from the body, by invariant 3;
     folding from the bar updates the layer's state and the stepper's
     selection; a resolved thread whose reader can neither reply nor resolve has no footer. E2E:
     `frontend/tests/e2e/plan-detail/plan-detail.page.ts` reads `thread-row`, which this slice
     removes, and `plan-detail-review.spec.ts` covers folding and resolving in the Changes section.
2. **The Activity card takes the bar (D2's shadow; D3, D4 and D5 in Activity; the rest of D6;
   D10).** Depends on slice 1.
   - Delivers, for a card in Activity: the three Activity rows of Title; every row of State
     labels, lines and link; the Excerpt and Footer rows, with the excerpt still in the body and
     drawn as today; the Activity row of Folding; the frame's shadow. After this slice no card is
     drawn without a bar, so invariants 2 and 7 hold in full and `ThreadSummary` goes.
   - Files: `CommentThreadCard.tsx`, `StatementAnchorContext.tsx`, `ReviewActivityTimeline.tsx`,
     `ThreadStack.tsx`, and the five locale files for "Comment on the plan".
   - Tests: `CommentThreadCard.test.tsx` and `StatementAnchorContext.test.tsx`, one case for each
     row delivered. They include: a deleted change and a thread that is not on lines; two folded
     cards whose fold controls have different names; the excerpt renders in the body and not in the
     bar. E2E: `plan-detail-review.spec.ts` covers folding and resolving in Activity.
3. **No line tints, and the width limit (D1, D2's width).** Depends on slice 1, which gives every
   thread in the Changes section a title that names its lines.
   - Delivers: invariant 1 and every row of On the statement.
   - Files: `StatementThreadsLayer.tsx`, `statementThreads.css`.
   - Tests: `StatementThreadsLayer.test.tsx`: no line is decorated for an open, an unfolded or a
     resolved thread; a selection and an open composer are still highlighted. E2E:
     `plan-detail-review.spec.ts` asserts that a selected line's background is the grey token.
4. **Comment column and replies (D7).** Depends on nothing.
   - Delivers: the First comment and Replies rows of Body and footer.
   - Files: `ThreadComment.tsx`, `CommentThreadCard.tsx`, `CommentThreadCard.css`.
   - Tests: `ThreadComment.test.tsx`, `CommentThreadCard.test.tsx`.
5. **Composer (D8).** Depends on slice 1, for the bar.
   - Delivers: the composer row of Title, and invariant 2 for the composer.
   - Files: `InlineThreadComposer.tsx`.
   - Tests: a new `InlineThreadComposer.test.tsx`: the composer has the frame, including the
     shadow, and its avatar is in the body.

D9 changes nothing, so no slice carries it.

The two sibling docs add three more PRs. Across all three docs the order is:

| PR | Depends on |
| --- | --- |
| Card 1: bar and folding in the Changes section | Nothing |
| Card 2: the Activity card takes the bar | Card 1 |
| Card 3: no line tints, and the width limit | Card 1 |
| Card 4: comment column and replies | Nothing |
| Card 5: composer | Card 1 |
| Excerpt 1: the SQL excerpt | Card 4, for the comment's text column |
| Group 1: Show and Hide in the link color | Nothing |
| Group 2: findings grouped under their review | Card 2, for the finding card's bar |

## Not in this PR

- The SQL excerpt inside an Activity card, which the Plan comment SQL excerpt doc covers.
- Findings grouped under their review in Activity, which the Plan review findings group doc
  covers.
- Opening a long "Applies to" list in a popover (BYT-10512), ordering it by environment
  (BYT-10513), and a label in the bar for a comment that involves a production database
  (BYT-10514).
- Naming who resolved a thread. It needs a field on the thread first.
- Creating a thread that is not on lines from the product. D10 gives such a thread a bar and adds
  no way to create one. The comment box in Activity keeps posting a plain comment, and the Changes
  section keeps creating threads on selected lines only.
- A title that names the change for a review finding without lines. The finding would first need
  to record its change, and no review posts one today.
- Highlighting a thread's lines while the pointer is on its title bar.
- The thread stepper (`StatementThreadWalker`) and the margin markers, which do not change.
