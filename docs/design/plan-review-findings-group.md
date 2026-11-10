# Plan review findings group

Status: proposal · 2026-11-10

When Standard rules or AI review finishes a run on a plan, the Activity timeline records it as one
line, such as "Standard rules reviewed the SQL · 2 findings, 1 open", with a Show or Hide control.
The findings that run posted are comment threads. Today each one is its own timeline entry, with
its own icon, at the same level as the review line that owns it, so nothing on the page says which
findings belong to which review. The rule this doc lands on: **a review and the findings it posted
share one bordered group in Activity, and a comment from a person stays its own entry.**

This is the third of three docs, each in its own PR. [Plan comment card](plan-comment-card.md)
covers the card each finding is drawn as. The second, Plan comment SQL excerpt, covers the SQL
shown inside it.

## Problem

This applies where SQL Review V2 is on (`sqlReviewV2Enabled()`), which is where a completed run is
one line. Under `frontend/src/routes/project/plan-detail/components/review/`:

- **A finding is a sibling of its review.** `buildTimelineEntries` (`timelineEvents.ts`) emits one
  `review-run` entry for the run and then one entry per finding. `ReviewActivityTimeline.tsx`
  renders the run with `ReviewRunActivityRow` and each finding with `ThreadActivityRow`, which
  gives it a timeline icon of its own and a card offset by `ml-3`.
- **Show and Hide have no visible scope.** Show reveals a run's findings as further entries below
  the line. With two reviews in a row, the reader cannot see where one review's findings end.
- **Show and Hide read as static text.** The control is a link `Button` whose color is overridden
  with `text-control-light`, so it looks like the timestamp beside it. BYT-10508 reports this.

## Principle

> In a timeline, an entry that owns other entries holds them. Ownership is shown by containment,
> one level deep, and never by indentation alone.

## Decisions

**D1 · A review and its findings share one group.** The review line becomes the first line of a
bordered group, and the finding cards sit inside it. The findings lose their own timeline icon; the
review keeps its icon on the timeline. Show and Hide keep their place on the review line and now
act on a visible group. Rejected: indenting the findings under their review line with nothing
joining them. It leaves two card edges on the page, the indented findings and a person's comment
further left, and reads as a misalignment. Rejected: a vertical line down the indented findings, which is what GitHub
draws for the threads of a review. It adds a second vertical line beside the timeline's own and
keeps two card edges. Rejected: giving every thread a line of its own, such as "commented on the
SQL", so that no card is indented. It costs a line for each comment from a person and shows
ownership less plainly than a group does. It is the fallback if D4 is not accepted.

**D2 · The group is a border and nothing else.** `border-block-border` and `rounded-sm`, with no
shadow, no fill and no title bar. The cards inside keep their title bar and shadow from
[Plan comment card](plan-comment-card.md). The two levels look different, so it stays clear that
the finding is the unit and the group only says where it belongs. The review line sits inside the
border with `px-3`, and the cards below it with 12px on each side and 8px between them.

**D3 · A person's comment stays a top-level entry.** Its card starts on the group's left edge, so
the page has one outer edge. Findings sit 12px inside it, and the border explains why.

**D4 · The UX guideline gains one exception.** `docs/agents/frontend-ux.md` says "Do not put cards
inside cards." The rule stays. The slice that builds D1 amends it to:

> Do not put cards inside cards, with one exception. A **group frame** MAY hold the cards that
> belong to one parent entry in a list or timeline when all of the following hold:
>
> - The frame's first line names the parent and carries the control that shows or hides the group.
> - The frame is a border only: `border-block-border` and `rounded-sm`, with no shadow, no fill and
>   no title bar. The cards inside keep their own.
> - Nesting stops there. A card inside a group frame MUST NOT contain another framed box.
> - A group with no cards showing renders as its first line alone, without the frame.

The first condition ties the frame to ownership, so it cannot be used for decoration. The second
keeps the two levels distinct. The third caps the depth at two. The fourth prevents empty boxes.
Rejected: dropping the rule. It exists to stop decorative nesting, and that reason still holds.

**D5 · Which cards a group shows is unchanged.** `buildTimelineEntries` already decides this, and
the group only changes where they are drawn:

- A run is open by default while one of its findings is still current, meaning no later run has
  superseded it. Show and Hide override the default.
- An open run shows the card of every finding it posted.
- A hidden run shows only the findings that would otherwise be nowhere (`needsActivityCard` in
  `threadModel.ts`): ones people replied to, and ones the Changes section cannot show. A finding
  that a later run superseded and that is resolved gets no card.

So a hidden review can still have a group, holding only those cards, and a review with no cards to
show is a plain line, as today.

**D6 · The group wraps whatever a review line is.** BYT-10505 proposes showing Standard rules and
AI review as one review. This design does not depend on the outcome: the group belongs to a review
line, however many reviewers the page distinguishes.

**D7 · Show and Hide look like actions.** The control drops the `text-control-light` override and
takes the shared link `Button`'s own color, the same as "View in Statement" on a finding's card.
Hover and keyboard focus come from the shared button. Rejected: a chevron in place of the word. It
would sit directly above the cards' own fold chevrons, and it is less clear which one folds what.
Rejected: an outlined button, which is heavier than the one line it sits on.

## States

| Review | Activity shows |
| --- | --- |
| No findings | The review line alone |
| Open, with findings | A group: the review line, then one card per finding |
| Hidden, no finding needs a card | The review line alone, with Show |
| Hidden, some findings need a card | A group with only those cards, and Show on the review line |
| A later review | The line says what changed: "1 new, 1 still found, 1 no longer found" |
| A finding a later run reports again | Its thread stays with the run that first posted it, as today |
| A comment from a person | Its own entry, outside any group |

## Scope

Frontend and one guideline file. One PR. It depends on slice 1 of
[Plan comment card](plan-comment-card.md), which gives a finding card its title bar.

1. **Group (D1 to D5, D7).**
   - `timelineEvents.ts`: a run's finding entries become children of its `review-run` entry.
   - `ReviewActivityTimeline.tsx`: `ReviewRunActivityRow` renders the group, and its Show and Hide
     control drops the `text-control-light` override. `ThreadActivityRow` keeps rendering a
     person's thread.
   - `foldTimeline.ts`: a group stays together when a long history folds, as an open run's line
     and cards do today.
   - `docs/agents/frontend-ux.md`: the exception in D4.
   - Tests: `timelineEvents.test.ts` (the kept-card rule in D5, unchanged cases plus the
     grouping), `ReviewActivityTimeline.test.tsx` (a finding renders inside its review's group and
     without a timeline icon; a hidden review with a kept finding still renders a group; a review
     with no cards renders no frame; the Show and Hide control carries no color override),
     `foldTimeline.test.ts`. E2E: `frontend/tests/e2e/plan-detail/plan-detail-review.spec.ts`.

## Not in this PR

- Grouping review events by SQL change, which is the other half of BYT-10508.
- Merging the two reviewers into one review. That is BYT-10505.
- The wording of review lines, which is BYT-10495.
