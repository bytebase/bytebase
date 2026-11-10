# Plan comment SQL excerpt

Status: proposal · 2026-11-10

In the Activity timeline, a comment thread on lines of SQL shows those lines above the comment, so
a reader can tell what the comment is about without opening the statement. Today every commented
line in that excerpt is tinted, whatever the comment says, and the comment starts on the same left
edge as the line numbers, so it reads as more lines of the statement. The rule this doc lands on:
**the excerpt marks the commented lines with a bar beside the line numbers, paints nothing behind
the code, and sits inset from the comment below it.**

This is the second of three docs, each in its own PR. [Plan comment card](plan-comment-card.md)
covers the card the excerpt sits in. The third, Plan review findings group, covers how findings
sit under the review that posted them.

## Problem

`SqlExcerpt` in
`frontend/src/routes/project/plan-detail/components/threads/StatementAnchorContext.tsx` renders the
recorded lines of the sheet revision a thread was made on.

- **Every commented line is tinted.** A line inside the anchored range gets `bg-accent/12`. The
  Changes section tints by severity and Activity is always accent, so the same finding has two
  colors in two places. [Plan comment card](plan-comment-card.md) removes the tints in the Changes
  section; this doc removes the last one.
- **The tint is the only range marker.** The excerpt starts two lines above the range
  (`CONTEXT_LINES_BEFORE`), so without the tint nothing separates context from the commented lines.
- **The comment runs on from the SQL.** Each excerpt row is padded `px-3`, and the comment below is
  padded `px-2 sm:px-3`, so the line numbers and the author's avatar start on the same edge.
- **An outdated excerpt is faded.** When the thread's lines have changed or cannot be placed, the
  excerpt renders at `opacity-60`. That takes the syntax colors under the 4.5:1 contrast minimum:
  a quoted identifier falls to about 3.4:1 against the background.

## Principle

> The SQL in a comment card is a quotation. Mark what the comment is about with a rule in the
> margin, the way a quotation is marked, and keep the quotation visibly apart from the comment.

## Decisions

**D1 · Two lines of context stay, in normal colors.** The excerpt keeps `CONTEXT_LINES_BEFORE = 2`
and the context lines keep their syntax colors. Rejected: showing only the commented lines. A range
that starts mid-statement, such as an `ADD COLUMN` line, is hard to read without the `ALTER TABLE`
above it. Rejected: grey context lines with no other marker. The reader has to compare text colors
to find the range, and a context of one short line barely differs.

**D2 · A bar beside the line numbers marks the range.** Rows inside the range carry a 3px bar in
`control`, 8px left of the line numbers. Nothing is painted behind the code. The title bar of the
card names the same lines in words. VS Code marks commented lines the same way, with a bar in the
margin. Rejected: darker, bolder line numbers for the range and no bar. It is the quietest option
and the easiest to miss.

**D3 · The SQL is inset from the comment.** The excerpt's left padding becomes the start of the
comment's text column once [Plan comment card](plan-comment-card.md) D7 gives the avatar its own
column: the comment's own left padding, a 20px avatar and an 8px gap. The comment is padded
`px-2 sm:px-3`, so the inset is 36px below the `sm` breakpoint and 40px from it up
(`pl-9 sm:pl-10`), where today it is 12px at every width. The line numbers then line up with the
comment's text and the avatar column stays clear. Only the SQL moves. The comment below it, its
replies and the reply footer stay where [Plan comment card](plan-comment-card.md) puts them.
GitHub's conversation view does the same thing: the comment starts at the box's left edge and the
code sits behind a number gutter about 64px wide. Rejected: a bordered comment
box that starts at the code column. Inside a review's group it would be a third frame, which
the Plan review findings group doc rules out. Rejected: indenting the
comment instead of the SQL. The comment loses width and still sits in the code's column.

**D4 · An outdated or unavailable excerpt is grey, not faded.** `opacity-60` goes. The lines render
in `text-control-light` without syntax colors, which is 4.8:1 on the background. The bar from D2
still marks the range.

## States

The excerpt follows the anchor state `StatementAnchorContext` already computes. When it renders
does not change: it renders once the recorded sheet has loaded, with whatever lines of the range
the sheet's preview holds. This doc changes only how it is drawn.

| Anchor state | Excerpt |
| --- | --- |
| `CURRENT` | Context and range in syntax colors, bar on the range |
| `OUTDATED` | The recorded lines in grey, bar on the range |
| `UNAVAILABLE`, the recorded sheet loaded | The recorded lines in grey, bar on the range |
| `UNAVAILABLE`, a large sheet's preview ends inside the range | The lines the preview holds, in grey, as today |
| `UNAVAILABLE`, the recorded sheet did not load | No excerpt |
| `PENDING` | No excerpt until the sheet loads; it loads when the card nears the viewport, as today |
| Range starts on line 1 or 2 | Fewer than two context lines, as today |

## Scope

Frontend only. One PR. It depends on the comment column slice of
[Plan comment card](plan-comment-card.md), the one that builds D7 there. That slice sets the
comment's text column, which D3 here lines up with. The slice is named by what it builds because
its number can change while that doc is in review.

1. **Excerpt (D1 to D4).**
   - Files: `StatementAnchorContext.tsx` (`SqlExcerpt`): drop `bg-accent/12` and `opacity-60`, add
     the bar and the `pl-9 sm:pl-10` inset, and render a dimmed excerpt as plain text in
     `text-control-light` instead of the colorized HTML. The condition that mounts `SqlExcerpt`
     does not change.
   - Tests: `StatementAnchorContext.test.tsx`: no row carries a background class; rows inside the
     range carry the marker and rows outside do not; a dimmed excerpt uses the grey text class, no
     opacity and no colorized HTML; rows carry the inset class pair.

## Not in this PR

- Starting the context at the first line of the statement the range belongs to, instead of a fixed
  two lines above it.
- Showing how the statement differs now from the revision an outdated comment was made on.
- Limiting the height of a very long range.
- Hiding the excerpt when a large sheet's preview ends inside the commented range. Today it shows
  the lines the preview holds, and this doc leaves when the excerpt renders as it is.
