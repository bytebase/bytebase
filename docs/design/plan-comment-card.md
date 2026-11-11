# Plan comment card

Status: proposal · 2026-11-11

On a plan, people and the automated reviewers comment on lines of SQL. A comment thread shows in
two places: under its lines in the statement editor of the Changes section, and as an entry in the
Activity timeline. A comment is auxiliary information about the statement. The statement is what
gets reviewed, approved and run, and a comment helps someone judge it. Today the page marks a
comment with its strongest signals instead: the commented lines are tinted, a folded thread is a
dense status row, and the box is as wide as the statement. The two places also draw the thread
differently. The rule this doc lands on: **a comment thread is one white card in both places, set
on a gray statement, with no color on the SQL, a title bar that names its lines and says how
urgent the thread is, and the same folded form everywhere.**

This is the first of three docs, each in its own PR. The second, Plan comment SQL excerpt,
covers the SQL shown inside an Activity card. The third, Plan review findings group, covers how
findings sit under the review that posted them.

## Problem

Internal testing of SQL review reported that inline comments do not read as comments (BYT-10500),
that the line background colors are confusing (BYT-10499), and that several reviews on one line
overwhelm the statement (BYT-10373). The causes are in how the thread is drawn, under
`frontend/src/routes/project/plan-detail/components/threads/`:

- **Color sits on the SQL.** `StatementThreadsLayer.tsx` paints every line an open thread covers:
  `bb-thread-line--passive` (gray) for a comment from a person, `--attention` (amber) for a P1 or
  P2 finding, `--blocking` (red) for a P0. Unfolding a thread repaints its lines with
  `bb-thread-line` (accent), so a finding's color changes when it is opened.
- **The box reads as part of the statement.** The view zone starts at the code column and is
  padded `py-2 pr-2`, so the card is as wide as the code and starts where the code starts. It has
  no title; its first row is the author. The card and the code are on the same white.
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

> A comment is auxiliary information about the statement. Set it apart by its ground, its
> structure and its words, never by color on the SQL, and draw it the same way wherever it
> appears. Keep color for the few things that should change what the reader does next.

GitHub's review thread is the reference where it has an answer: a bordered box of limited width, a
title bar that names the lines and holds the fold control, a quiet "Resolved" label, and indented
replies. Where this design differs from it, the decision says why.

## The card

One component draws a comment thread in both places. It has four parts, top to bottom.

| Part | What it is |
| --- | --- |
| Frame | The box: `border-block-border`, `rounded-lg` and `shadow-md` on `bg-background` |
| Bar | One line in 12/16 `text-control`, with a border under it while the card is unfolded: the fold control, a title, markers, state labels and, in Activity, the lines and a link |
| Body | In Activity the SQL excerpt, then the first comment and its replies |
| Footer | The reply field and the Resolve or Reopen button |

What a part shows depends on a few facts about the thread, and the code already has each of them:
the place the card is drawn in, what the thread is on, whether its lines are still in the
statement, whether it is resolved, its priority and the databases it applies to when it is a
review finding, what the reader may do, and whether it is folded. The sections below give each
part's content from those facts. Together they cover every card the product can draw, so a case
is never decided one at a time.

### Invariants

These hold for every card, in both places and in every state. A change that breaks one is wrong
even where it follows the tables.

1. **No thread paints the SQL.** Nothing about a thread's existence, state or priority colors a
   line. The one highlight is the reader's own selection while writing a new comment. It is
   neutral gray and ends with the selection.
2. **SQL is on gray and the card is white.** The statement in the Changes section and the excerpt
   in an Activity card sit on a gray ground, the same on every line. The card's bar, body and
   footer are `bg-background`. The ground is what sets a comment apart from the code.
3. **Every card has the frame and a bar.** That includes the new-comment composer and a thread that
   is not on lines.
4. **The bar is the same folded and unfolded.** Folding hides the body and the footer and changes
   nothing else. So a folded card says where a thread is, what state it is in and how urgent it
   is, and nothing about what it says: not its author, its text, its replies or which databases
   it applies to. Each of those is one click away.
5. **The fold control is named by the bar's text.** Its accessible name is the title, followed by
   the lines where the bar shows them apart from the title. Whatever the title is, the control has
   a name.
6. **Color in the bar is kept for what the reader must act on.** The bar's words and icons are
   `text-control`. Besides the link, three things in it are colored: the badge of a P0 or P1
   finding, "Prod", and "Outdated". The two markers are badges with a light fill and no
   border. "Resolved" and "Statement unavailable" are text with an icon, with no color and no
   fill.
7. **The footer is there only when the reader can act.** A reader who can neither reply nor resolve
   sees none.
8. **A place adds content, not a layout.** The Changes section and Activity fill the same bar and
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
the current statement, which `selectEditorThreads` decides and this doc does not change. A thread
from a person and a review finding are titled the same way.

### Markers

Markers follow the title, in this order, in both places and whether the card is folded or not.

| Fact | The bar shows |
| --- | --- |
| The thread is a review finding with priority P0 | The "P0" badge, the one drawn beside the finding's title |
| The thread is a review finding with priority P1 | The "P1" badge |
| The thread is a P2 finding, or comes from a person | No badge |
| The thread is a review finding that applies to at least one database in a production environment | "Prod", with no database name |

A production environment is one `isProtectedEnvironment` accepts, the tag the product already
calls "Production environment". It needs the environment-tier feature, so without that feature no
thread shows "Prod". A thread from a person has no list of databases, so it never shows it.

### State labels, lines and link

State labels follow the markers. The lines and the link sit at the right end of the bar.

| Fact | The bar shows |
| --- | --- |
| The thread is resolved | "Resolved" |
| Its lines changed after the comment (anchor state `OUTDATED`) | "Outdated", in the warning color |
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

A click folds or unfolds a card when it lands on the bar, as follows. The fold control stays the
one focusable control for it, with the name from invariant 5.

| Place | Folds or unfolds | Does not |
| --- | --- | --- |
| Changes section | A click anywhere on the bar | The composer's bar, which has no fold |
| Activity | A click on the fold control, the title, a marker or a state label | A click on the lines or on "View in Statement" |
| Both | | A click on a control inside the bar, which keeps its own meaning, and a drag that selects text |

### On the statement

| What | In this design |
| --- | --- |
| Ground | `bg-control-bg/50` behind the whole statement, by invariant 2 |
| Margin marker | Unchanged. It is the only mark on the statement at rest |
| Line tint for a thread | None, by invariant 1 |
| Highlight of the reader's selection | Neutral gray, while selecting lines or writing the comment |
| Width of a card | At most 48rem, and 16px clear of the editor's right edge |

## Decisions

**D1 · No color on commented lines.** The four at-rest tints go: gray, amber, red, and the accent
tint of an unfolded thread. The highlight of the reader's selection (`bb-thread-line--selecting`)
stays, because it is feedback on an action and ends with it. Its color changes from
`--color-info`, a blue, to the neutral `--color-control-bg-hover`. Severity is said by the badge
of a finding, in the bar and beside its title, and never by the SQL. Rejected: tinting only P1
lines, as BYT-10499 asks. Severity would still be painted on the SQL, and the reader would still
have to learn a color code. Rejected: one neutral tint for every commented range. It still paints
the statement, and the title names the lines in words.

**D2 · The frame has a width limit, 8px corners and a shadow that lifts it.** In the Changes
section the card is capped at `max-w-3xl` (48rem) and keeps 16px clear of the right edge, where
today it is 8px. The Activity card keeps its full width. The frame is `rounded-lg` and
`shadow-md` in both places and for the composer. Today it is `rounded-sm`, and only the Changes
section gives it a shadow, `shadow-sm`. The rounder corner and the stronger shadow make the card
read as something laid on the statement and not as a panel of it. `docs/agents/frontend-ux.md`
asks for `rounded-sm` on ordinary framed surfaces, and `frontend/scripts/check-ui-guideline.mjs`
enforces it: `rounded-lg` is an off-scale radius there. The guideline says a rule changes in
the guideline, the scanner and the scanner's tests together, and never through the baseline.
So the slice that sets the frame changes all three. A comment card, and a group frame that
holds comment cards, use `rounded-lg`, and the scanner allows it in the files that draw them,
the way it already allows the lifecycle stamp its `rounded-full`. Every other surface keeps
the scale. The fields and buttons inside the card keep their own corners. Rejected: keeping the
full width in the Changes section. A box as wide as the code reads as a band of the statement.
Rejected:
12px corners. A folded thread becomes a pill, which in this product is a button or a tag.
Rejected: a pointer from the card to its line, a shape the product
has nowhere else. Rejected: `shadow-lg`. A folded thread then looks like a floating menu.

**D3 · A title bar names the lines.** Every card starts with a bar, and its title says what the
thread is on. The bar is white like the rest of the card, by invariant 2, with a border under it
while the card is unfolded. Its text is `text-control`, by invariant 6. In Activity the lines
stay on the right, before "View in Statement", which is the order the Activity header has today.
Rejected: the range label that appears only when two threads' ranges differ. With D1 it would
leave most threads with nothing saying which lines they are on.

**D4 · A folded thread is its title bar, and the bar says how urgent the thread is.**
`CollapsedThreadRow` and `ThreadSummary`, the two folded rows of today, are both deleted. After
the title the bar carries two markers, by the Markers table: the badge of a P0 or P1 finding, and
"Prod" when the finding applies to a production database. A P0 must change and a P1 must be
accepted by a person, so both need someone to act, and with several folded findings the markers
say which to open first. The badge is the one `ReviewResult` draws beside the finding's title.
That one stays, so an unfolded finding shows its priority twice. The bar cannot drop it on
unfold without breaking invariant 4. "Prod" takes the same form as the priority badge, a light
fill with no border: `Badge variant="secondary"`, the accent at 10% behind accent text. That is
also how the product draws an environment in its default color. Rejected: an outlined "Prod"
beside a filled priority badge. The two markers then read as different kinds of thing.
Rejected: no markers. Nothing would say which folded finding must be fixed. Rejected: a badge for
P2. It is advisory, and a marker on every finding marks nothing. Rejected: the database's name in
the label. It makes the bar a third longer and repeats the "Applies to" row. Rejected: the
author's avatar and name. Rejected: a word or an icon that says whether the thread is a finding
or a person's comment. The badge already marks a finding where it matters.

**D5 · Any thread can be folded, by a click on its bar.** The fold control is an icon-only
`Button size="xs"` at the left of the bar, with `aria-expanded`. In the Changes section it
replaces the close button in the author row. A click on the bar folds too, by the table under
Folding: a folded bar is a full-width row with one job, and a 24px target on it reads as broken.
In Activity the right end of the bar holds the lines and a link into the statement, so that end
does not fold. The bar's click is a pointer shortcut. It adds no second focus stop, and the fold
control keeps the name and the state for the keyboard and for screen readers. Rejected: the
fold control as the only part that folds. Rejected: the whole bar in Activity. A click meant
for the link that lands beside it would fold the card.
Rejected: folding only resolved threads. A long open finding cannot be put away while reading the
rest of the timeline.

**D6 · State is a text label in the bar, and "Outdated" is in the warning color.** "Resolved"
moves from the footer into the bar and keeps the form `ThreadStateBadge` has today, an icon and
text with no fill. Its color changes from `text-control-light` to the bar's `text-control`.
"Outdated" and "Statement unavailable" take the same form in the same place. "Outdated" is
`text-warning-hover`, the text color of the `attention` badge, which is 5:1 on the card. It is
the one state that tells the reader to doubt the comment, because the lines it was written about
have changed. "Statement unavailable" stays gray: it says the card cannot show where the thread
is, not that the comment may be wrong. Today "Outdated" is a filled amber badge. Rejected:
keeping the fill. A filled badge is the form of a priority, and "Outdated" is a state. Rejected:
gray "Outdated", like "Resolved". It is easy to miss on a comment that may no longer apply.
Rejected: "Resolved by" with the name. `IssueComment` records `thread_state` and nothing about
who set it, so the name needs a backend field first.

**D7 · Comment text sits in its own column.** The avatar gets a 20px column, and the author row,
the "Applies to" row, the priority and title, and the text sit beside it. The
`bg-control-bg/50` band behind replies goes, since the indent now sets them apart. Rejected:
GitHub's current layout, which indents replies only and keeps the first comment's text flush with
its avatar. A finding stacks four rows, and they scan better with the avatar out of the way.

**D8 · The composer is the same card.** It has the frame and a bar, by invariant 3. Its avatar
moves from the bar into the body's avatar column beside the editor, so a comment does not change
shape when it is posted.

**D9 · The "Applies to" row is unchanged, and stays in the body.** The bar does not name or count
a finding's databases. They are part of what the finding says, so by invariant 4 they show when
the card is unfolded and are hidden when it is folded, whether the thread is open or resolved.
The one thing the bar says about them is "Prod", from D4, which is the label BYT-10514 asks for.
Rejected: a count or the database names in the bar. The bar stops being simple and long lists do
not fit.

**D10 · A thread that is not on lines has a bar too.** A thread is on lines when it has a
statement anchor, and the anchor is also what names its change. A thread without one can come
from an API client, or from the rule review for a syntax error that has no position, which the
parsers do not report today. The product's own comment box does not create one. Whoever posted
it, it is titled "Comment on the plan", because the card has no change to name. So invariant 3
has no exception and the card has one layout. Rejected: keeping today's card without a bar for it.
Every other rule would then need a second version. Rejected: never folding it. A resolved one
could not be put away.

**D11 · The statement is on gray and the card is white.** Today the statement is on white, and
the card's replies and footer are on `bg-control-bg/50`. This swaps them. In the Changes section
the statement's ground is `bg-control-bg/50` on every line, commented or not, and the card is
`bg-background` from its bar to its footer. The card then stands out from the statement by its
ground, which is what lets invariant 1 hold with no tint on any line. The statement editor draws
its background from the editor theme. The SQL Editor already overrides that with a marker class
that makes the editor's base layers transparent, so that its wrapper's background shows
(`sqleditor--monaco-transparent` in `tailwind.css`). The statement editor takes its ground the
same way. Rejected: the stronger `bg-control-bg` behind the statement. It recolors the largest
surface of the page on every plan, and a gray that strong can read as a locked field. The
excerpt in an Activity card is a small band inside a white card, so it takes the stronger gray,
which the Plan comment SQL excerpt doc covers. Rejected: a gray bar on a white statement. The
bar then separates the card alone, and the card's body is the same white as the code.

## Scope

Frontend, with one exception in the UX guideline and its scanner. Six PRs. Each leaves main
shippable, names what it depends on, and is opened after the one before it merges.

A slice is finished when the rows it names hold and no invariant is broken. Its tests assert every
row and invariant it delivers, at least one case each, so a slice cannot pass while a row it
claims is missing. A slice that removes a signal the page has today delivers its replacement
in the same PR, so main never shows less than it did. Under Tests, a slice names where its
tests live and the cases that are easy to miss. That list is not the whole set. This plan names
each slice's files so its size can be judged. Which props and helpers carry the rules is decided
in the slice's own PR and reviewed there.

1. **Bar and folding in the Changes section (D3, D4's folded form and its priority badge, D5,
   and "Resolved" from D6).** Depends on nothing.
   - Delivers, for a card in the Changes section: the first row of Title; the three priority
     rows of Markers; "Resolved" from State labels; the Footer row; the Changes section row of
     Folding; the white bar and footer of invariant 2; invariants 5 and 7; invariant 4 apart
     from "Prod", which slice 6 adds; and invariant 6 for the badge and for "Resolved".
     `CollapsedThreadRow` goes, and it is what shows a folded finding's priority today, so the
     badge moves into the bar in this slice. Until slice 2 the Activity card is untouched,
     apart from the color of "Resolved", which both places draw with `ThreadStateBadge`.
   - Files: `CommentThreadCard.tsx`, `ThreadStack.tsx`, `StatementThreadsLayer.tsx`,
     `ThreadStateBadge.tsx`, `ReviewResult.tsx` for the badge the bar shares, and the five
     locale files for the title.
   - Tests: `CommentThreadCard.test.tsx`, `ThreadStack.test.tsx` and
     `StatementThreadsLayer.test.tsx`. They include: the bar is the same folded and unfolded, and
     a folded card shows nothing from the body, by invariant 4; a P0 and a P1 finding show the
     badge in the bar, folded and unfolded, and a P2 finding and a person's thread show none;
     folding from the bar updates the layer's state and the stepper's selection; a click on the
     bar's title folds and unfolds, and a drag that selects its text does not; a resolved thread
     whose reader can neither reply nor
     resolve has no footer; the bar and the footer are `bg-background`; nothing in the bar
     carries `text-control-light`. E2E: `frontend/tests/e2e/plan-detail/plan-detail.page.ts`
     reads `thread-row`, which this slice removes, and `plan-detail-review.spec.ts` covers folding
     and resolving in the Changes section.
2. **The Activity card takes the bar, and the frame (the corners and shadow of D2; D3, D4's folded
   form and D5 in Activity; the rest of D6; D10).** Depends on slice 1.
   - Delivers, for a card in Activity: the three Activity rows of Title; the three priority rows
     of Markers; every row of State labels, lines and link; the Excerpt and Footer rows, with the
     excerpt still in the body and drawn as today; the Activity row of Folding. For a card in
     both places: the frame, `rounded-lg` and `shadow-md`, with the exception in the guideline
     and in its scanner. After
     this slice no thread card is drawn without a bar or the frame, so invariant 8 holds in full,
     invariant 3 holds for every card but the composer, which slice 5 finishes, and invariant 6
     holds for every state label, and `ThreadSummary` goes.
   - Files: `CommentThreadCard.tsx`, `StatementAnchorContext.tsx`, `ReviewActivityTimeline.tsx`,
     `ThreadStack.tsx`, `docs/agents/frontend-ux.md`, `frontend/scripts/check-ui-guideline.mjs`
     with its test, and the five locale files for "Comment on the plan".
   - Tests: `CommentThreadCard.test.tsx` and `StatementAnchorContext.test.tsx`. They include: a
     deleted change and a thread that is not on lines; two folded cards whose fold controls have
     different names; a folded P0 finding shows its badge; a click on the change's label folds,
     and a click on the lines or on the link does not; the excerpt renders in the body and not
     in the bar; "Outdated" carries the warning color and no fill, and "Resolved" and "Statement unavailable" carry neither; the
     frame has the corners and the shadow in both places. `check-ui-guideline.test.mjs`: the
     scanner accepts `rounded-lg` in a file that draws a comment card and still rejects it in
     any other file. E2E: `plan-detail-review.spec.ts` covers folding and resolving in Activity.
3. **No line tints, the gray ground, and the width limit (D1, D2's width, D11).** Depends on
   slice 1, which gives every thread in the Changes section a title that names its lines.
   - Delivers: invariant 1, the statement's ground from invariant 2, and every row of On the
     statement.
   - Files: `StatementThreadsLayer.tsx`, `statementThreads.css`, the statement editor's wrapper,
     and `tailwind.css` for the rule that makes the editor's base layers transparent.
   - Tests: `StatementThreadsLayer.test.tsx`. They include: no line is decorated for an open, an
     unfolded or a resolved thread; a selection and an open composer are still highlighted; the
     wrapper of a thread and the wrapper of the composer both carry the width limit and the 16px
     clearance. E2E: `plan-detail-review.spec.ts` asserts that a selected line's background is
     the gray token, that the statement's background is its ground with and without a thread, and
     that an unfolded card is at most 768px wide and ends 16px or more before the editor's right
     edge.
4. **Comment column and replies (D7).** Depends on nothing.
   - Delivers: the First comment and Replies rows of Body and footer.
   - Files: `ThreadComment.tsx`, `CommentThreadCard.tsx`, `CommentThreadCard.css`.
   - Tests: `ThreadComment.test.tsx` and `CommentThreadCard.test.tsx`. They include: a reply is
     indented and the first comment is not.
5. **Composer (D8).** Depends on slice 2, for the bar and the frame.
   - Delivers: the composer row of Title, and invariants 2 and 3 for the composer.
   - Files: `InlineThreadComposer.tsx`, and the scanner's list of files if the composer is not
     on it yet.
   - Tests: a new `InlineThreadComposer.test.tsx`. They include: the composer has the frame,
     with its corners and shadow, and its avatar is in the body.
6. **"Prod" in the bar (D4's production marker).** Depends on slice 2, which puts the bar in
   both places. It adds a signal the page does not have today, so nothing waits on it.
   - Delivers: the "Prod" row of Markers, which completes invariant 4, and invariant 6 for
     "Prod".
   - Files: `CommentThreadCard.tsx`, a helper beside `threadModel.ts` that tells whether a
     finding's databases include one in a production environment, and the five locale files
     for "Prod".
   - Tests: `CommentThreadCard.test.tsx` and the helper's own test. They include: "Prod" shows
     when one of the finding's databases is in a production environment, and does not when
     none is, when the environment-tier feature is off, or on a person's thread; the label
     holds no database name.

D9 changes nothing, so no slice carries it.

The two sibling docs add three more PRs. Across all three docs the order is:

| PR | Depends on |
| --- | --- |
| Card 1: bar and folding in the Changes section | Nothing |
| Card 2: the Activity card takes the bar, and the frame | Card 1 |
| Card 3: no line tints, the gray ground, and the width limit | Card 1 |
| Card 4: comment column and replies | Nothing |
| Card 5: composer | Card 2 |
| Card 6: "Prod" in the bar | Card 2 |
| Excerpt 1: the SQL excerpt | Card 4, for the comment's text column |
| Group 1: Show and Hide in the link color | Nothing |
| Group 2: findings grouped under their review | Card 2, for the finding card's bar and corners |

## Not in this PR

- The SQL excerpt inside an Activity card, which the Plan comment SQL excerpt doc covers.
- Findings grouped under their review in Activity, which the Plan review findings group doc
  covers.
- Opening a long "Applies to" list in a popover (BYT-10512), and ordering it by environment
  (BYT-10513).
- Naming who resolved a thread. It needs a field on the thread first.
- Keeping an unsaved edit of a comment when its card folds. Today a reply draft survives a fold
  and an unsaved edit does not, and this design does not change that. The PR that builds folding
  settles it in its own review.
- Creating a thread that is not on lines from the product. D10 gives such a thread a bar and adds
  no way to create one. The comment box in Activity keeps posting a plain comment, and the Changes
  section keeps creating threads on selected lines only.
- A title that names the change for a review finding without lines. The finding would first need
  to record its change, and no review posts one today.
- Highlighting a thread's lines while the pointer is on its title bar.
- The color of the line numbers, in the statement and in the excerpt.
- The thread stepper (`StatementThreadWalker`) and the margin markers, which do not change.
