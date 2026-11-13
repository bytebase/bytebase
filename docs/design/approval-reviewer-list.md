# Approval step reviewer list

Status: proposal · 2026-11-10

Each current or pending step in a plan's approval flow shows an avatar stack and "N reviewers".
Hovering it lists the people who hold the step's role, but only the first 50, followed by "and N
more". A role bound to a large group can have more than a thousand holders, so the person a plan's
creator is looking for is often among the hidden ones, and the list has no search.
**Every holder of a step's role is reachable from that step's reviewer list: the list opens on
click, holds every holder, searches them by name or email, and lets you copy any reviewer's email.**

## Problem

A customer (2026-11-05) asked for two things: show all available reviewers, including those past
the first 50, and let users search the full list by name. The current list
(`NodeReviewers` in `frontend/src/routes/project/plan-detail/components/review/ReviewApprovalFlow.tsx`)
cannot do either:

- It renders `candidates.slice(0, 50)` and summarizes the rest as "and N more". Nothing reveals them.
- It opens on hover (`openOnHover`), and a hover popover closes when the pointer leaves, so it
  cannot hold a search box. Its `div` trigger already takes focus and opens on click, Enter or
  Space, but it shows no focus ring and a default cursor, so neither path is visible. The UX
  contract requires a visible keyboard focus state on interactive elements
  (`docs/agents/frontend-ux.md`, Borders, Corners, Focus, And Layers).
- The "+N" overflow avatar is a fixed `size-5` circle, so a count such as "+1200" spills out of it.

The data is not the limit. `useApprovalCandidates` (`frontend/src/routes/project/useApprovalCandidates.ts`)
already loads every active holder of the role into the browser, expanding groups, and sorts them:
the current user first, then people before service accounts and workload identities, then by title.
The 50-row cap is only a rendering limit.

The list renders in three places, all through `ReviewApprovalFlow`:

| Place | Renders |
| --- | --- |
| Review section of the plan page | `PlanReviewSection.tsx` → `ReviewApprovalFlow` |
| Bypass confirmation sheet | `bypassDeploy.tsx` → `ReviewApprovalFlow` |
| Header Plan status popover | `PlanStatusAction.tsx` → `ReviewApprovalFlow compact` → `CompactVerticalFlow` → `FlowNode` |

## Principle

> Make the safe path the easy path. If finding an approver is hard, people go around the approval
> flow and chase someone in chat instead. The reviewer list answers "who can approve this?" for
> every holder of the role, not the first 50.

## Decisions

**D1 · A popover that opens on click.** The trigger (avatar stack plus "N reviewers") stops
opening on hover. It opens the shared `Popover` on click, Enter or Space, as it already can, and
shows a focus ring and a pointer cursor. Focus moves into the popover, Escape closes it and returns
focus to the trigger, and a click outside closes it. Rejected: keeping hover, which closes when the
pointer leaves and so cannot hold a search box. Rejected: a `Dialog`, whose scrim hides the plan
while the user only looks someone up. Rejected: a `Sheet`, which the contract's surface table
gives to creating or editing a resource.

**D2 · The full list, with no cap and no "load more".** The popover lists every candidate in one
scrolling list and drops "and N more". Only the rows in view are mounted (virtualized, with
`@tanstack/react-virtual`, already a frontend dependency), so a role with thousands of holders
costs no more to open than one with fifty. Rejected: "load more" in pages of 50. The contract's
load-more pattern is for results paged from the server (frontend-ux.md, With Pagination Or Load
More), and every candidate is already in the browser, so a load-more button would only hold back
rows the page already has.

**D3 · Search by name or email, in the browser.** Above the list, a `SearchInput` filters
candidates by a case-insensitive substring match on the title or the email. It searches every
candidate, including service accounts and workload identities, and keeps the list's order. The
contract allows filtering in the browser when the full bounded set is loaded and the user is not
misled about server-wide search (frontend-ux.md, Without Pagination); the list is the step's
complete set. The search box appears only when the step has more than 10 candidates. When nothing
matches, the list says so and offers a button that clears the search.

**D4 · A copy button on every row.** Each row ends with the shared `CopyButton`, which copies the
row's email and confirms with the shared "Copied" toast, or "Copy failed". The button is always
visible, so it works on touch screens and needs no hover to discover. Its accessible name names
the row ("Copy email for Ana Patel"), since a list of identical "Copy" buttons is ambiguous to a
screen reader; `CopyButton` gains an optional label for this. Every row gets one, including
service accounts and workload identities, so the rule has no exceptions. Rejected: showing the
button only on hover or focus, which hides it on touch screens. Rejected: copying when the email
text is clicked, which nobody would discover and which fights text selection.

**D5 · One reviewer list in all three places.** The Review section, the bypass sheet and the
header Plan status popover render the same trigger and popover. In the header, the reviewer list
is a popover inside a popover: opening it or clicking inside it must not close the Plan status
popover, and Escape closes the reviewer list first. Rejected: making the count in the header plain
text, which would remove a list the header shows today.

**D6 · The overflow count grows to fit.** The "+N" count keeps its 20px height and widens when
the count needs more room, so "+1200" stays inside it. Once it can widen it is a text label, which
the UX contract gives `rounded-xs` corners; `rounded-full` stays with the avatars.

What does not change: the avatar stack shows the first three candidates; the order is the one
`useApprovalCandidates` produces; the last plan editor stays in the list with its "Can't approve"
line, and the warning under the step stays; a step with no holders still reads "No reviewers" and
has nothing to open.

**D7 · Nothing.** Just trying

## States

| Step's candidates | Trigger | Popover |
| --- | --- | --- |
| Project policy still loading | Nothing rendered | None |
| No holders | "No reviewers", not a button | None |
| 1 to 10 | Avatar stack and "N reviewers" | Every candidate, each with a copy button; no search box |
| More than 10 | Avatar stack and "N reviewers" | Search box, then every candidate, each with a copy button |
| Search with matches | Unchanged | Matching candidates in the same order |
| Search with no match | Unchanged | "No reviewers match" with a button that clears the search |
| Copy clicked | Unchanged | Shared "Copied" toast, or "Copy failed" when the browser refuses |

## Scope

One PR. It holds this doc and the frontend change, and needs no backend change.

1. **Reviewer list (D1–D6).**
   - Files: `ReviewApprovalFlow.tsx` (`NodeReviewers`: the trigger and the "+N" count); a new
     `ReviewerList.tsx` beside it for the popover body (search, list, rows);
     `frontend/src/components/ui/copy-button.tsx` (optional accessible label); the five locale files
     for the search placeholder, the empty-search message and the copy label, with the unused
     `and-n-more` keys removed.
   - Unit tests. `ReviewApprovalFlow.test.tsx`: the popover does not open on hover, opens on
     click, stays open when the pointer leaves, and lists every reviewer; the "+N" count shows
     every candidate past the third and widens to fit. `ReviewerList.test.tsx`: a holder past the
     50th is found by name or email; search keeps the order; no search box at 10 or fewer; the
     empty-search state clears; each row's copy button names its person and copies its email.
     `copy-button.test.tsx`: the label becomes the accessible name.
   - E2E: `frontend/tests/e2e/plan-detail/plan-detail-review.spec.ts` opens the reviewer list by
     clicking instead of hovering ("the step lists the service account as its reviewer"), and a new
     case opens the list from the header Plan status popover and closes it with Escape while the
     header popover stays open.

## Not in this PR

- Notifying a reviewer from the list. Bytebase already sends "Approval required" to the project's
  webhooks when a step starts, mentioning the people who can approve it, and direct-messages them
  where the webhook has direct messages on (`NotifyApprovalRequested` in
  `backend/component/review/evaluator.go`). A per-reviewer reminder needs a backend endpoint
  first: one that resends through that channel to one person, limits how often, reports delivery
  and writes the audit log.
- Choosing a specific reviewer for a plan. Approval is by role; any holder can approve.
- Grouping the list by the group that grants the role.
- The folded "N pending" chip's popover, which shows avatars and a count, not a list.
