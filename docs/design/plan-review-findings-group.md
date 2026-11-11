# Plan review findings group

Status: proposal · 2026-11-11

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

**D2 · The group is a border and nothing else.** `border-block-border`, with the corners of the
cards it holds and with no shadow, no fill and no title bar. A comment card is `rounded-lg` in
[Plan comment card](plan-comment-card.md), so the group is too: a sharper frame around rounder
cards reads as two unrelated boxes. That doc changes the guideline and its scanner to allow the
corner in the files that draw a comment card, and the slice that builds the group adds the
group's file to them. The cards inside keep their title bar and shadow. The two
levels look different, so it stays clear that the finding is the unit and the group only says
where it belongs. The review line sits inside the border with `px-3 py-2`, so a folded group has
8px above and below its text and does not read as crowded. The cards sit below the line with 12px
on each side and 8px between them.

**D3 · A person's comment stays a top-level entry.** Its card starts on the group's left edge, so
the page has one outer edge. Findings sit 12px inside it, and the border explains why.

**D4 · The UX guideline gains one exception.** `docs/agents/frontend-ux.md` says "Do not put cards
inside cards." The rule stays. The slice that builds D1 amends it to:

> Do not put cards inside cards, with one exception. A **group frame** MAY hold the cards that
> belong to one parent entry in a list or timeline when all of the following hold:
>
> - The frame's first line names the parent and carries the control that shows or hides the group.
> - The frame is a border only: `border-block-border`, with the corners of the cards it holds and
>   with no shadow, no fill and no title bar. The cards inside keep their own.
> - Nesting stops there. A card inside a group frame MUST NOT contain another framed box.
> - Only a parent that has cards gets a frame. Hiding them folds the frame to its first line. A
>   parent with no cards is a plain line.

The first condition ties the frame to ownership, so it cannot be used for decoration. The second
keeps the two levels distinct. The third caps the depth at two. The fourth prevents a frame around
nothing, and keeps an entry the same shape when it is hidden and shown.
Rejected: dropping the rule. It exists to stop decorative nesting, and that reason still holds.

**D5 · Show and Hide fold the whole group.** The group is a card whose first line is the review
line, and Show and Hide are its fold control:

- A run is open by default while one of its findings is still current, meaning no later run has
  superseded it. Show and Hide override the default. This does not change.
- An open group shows the card of every finding the run posted.
- A hidden group is folded to its first line, inside its frame. It shows no finding card,
  whatever replies the findings have.
- A review with no findings has nothing to fold. It is a plain line with no frame and no Show.

Today a hidden run still shows the findings `needsActivityCard` in `threadModel.ts` keeps: one
people replied to, or one the Changes section cannot show. So when every finding of a review has
a reply, Hide changes nothing but its own word, and reads as broken. A hidden finding is not
lost: the review line still counts it, as in "2 findings, 1 open", and Show brings it back.
Rejected: keeping that rule. A control that sometimes hides is worse than one that always does.
Rejected: a hidden review as a plain line. The entry would change from a box to a sentence and
back each time it is toggled.

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
| Hidden, with findings | The group folded: its frame around the review line, with Show |
| A later review | The line says what changed: "1 new, 1 still found, 1 no longer found" |
| A finding a later run reports again | Its thread stays with the run that first posted it, as today |
| A comment from a person | Its own entry, outside any group |

## Scope

Frontend and one guideline file. Two PRs. Each leaves main shippable and names what it depends on.

1. **Show and Hide in the link color (D7).** Depends on nothing. It does not need the group or the
   new card.
   - `ReviewActivityTimeline.tsx`: the Show and Hide control in `ReviewRunActivityRow` drops the
     `text-control-light` override.
   - Tests: `ReviewActivityTimeline.test.tsx` (the control carries no color override).
2. **Group (D1 to D5).** Depends on the slice of [Plan comment card](plan-comment-card.md) that
   gives the Activity card its title bar and its corners. The slice is named by what it builds
   because its number can change while that doc is in review.
   - `timelineEvents.ts`: a run's finding entries become children of its `review-run` entry, and
     a hidden run has none. `needsActivityCard` in `threadModel.ts` then has no caller and goes.
   - `ReviewActivityTimeline.tsx`: `ReviewRunActivityRow` renders the group. `ThreadActivityRow`
     keeps rendering a person's thread.
   - `foldTimeline.ts`: a group stays together when a long history folds, as an open run's line
     and cards do today.
   - `docs/agents/frontend-ux.md`: the exception in D4.
   - `frontend/scripts/check-ui-guideline.mjs` and its test: the group's file joins the files the
     scanner allows `rounded-lg`.
   - Tests: `timelineEvents.test.ts` (the default from D5; an open run has every finding and a
     hidden run has none, with and without replies), `ReviewActivityTimeline.test.tsx` (a finding
     renders inside its review's group and without a timeline icon; a hidden review with findings
     renders its frame around the review line and no card; a review with no findings renders no
     frame), `foldTimeline.test.ts`. E2E:
     `frontend/tests/e2e/plan-detail/plan-detail-review.spec.ts`.

## Not in this PR

- Grouping review events by SQL change, which is the other half of BYT-10508.
- Merging the two reviewers into one review. That is BYT-10505.
- The wording of review lines, which is BYT-10495.
