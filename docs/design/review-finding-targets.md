# Review finding "Applies to" list

Status: in review · 2026-11-17

A Standard rules or AI review finding lists the databases it applies to in an "Applies to" row.
With five or more, the row showed three chips and "+N more", and "+N more" expanded every chip
inline, with no search and no way to fold the list back. The three chips were the first resource
names in alphabetical order, so a finding's production databases often sat behind the count, and
no chip showed its environment anyway.
**The row says in one line, at any size, which environments a finding touches, closest to
production first, and opens every database in a searchable popover.** This covers BYT-10512 and
BYT-10513.

## Problem

The row is `ReviewTargets` in
`frontend/src/routes/project/plan-detail/components/threads/ReviewResult.tsx`. Before this change:

- Expanding "+N more" put every database inline. On a change to dozens of databases it pushed the
  finding's heading and text several lines down, and it could not be folded back.
- The order was the server's: `slices.Sort` over full resource names in
  `backend/runner/reviewrun/rule_review.go` and `ai_review.go`, so instance ID, then database name.
  The environment played no part.
- A chip showed the engine, instance and database, never the environment. The title bar's
  production badge says a finding touches production, not which databases or how many.
- For a spec that targets a database group, the row never named the group. It was never hidden
  either: `equalsSpecDatabases` hides the row only when a hand-picked list equals the finding's
  databases, so a finding on every database of a group listed them all.

The row renders in three places, all through `ReviewTargets`:

| Place | Renders |
| --- | --- |
| Changes section of the plan page | `CommentThreadCard` → `ThreadComment` → `ReviewTargets` |
| Review activity on the plan page | `CommentThreadCard` → `ThreadComment` → `ReviewTargets` |
| Issue detail activity | `IssueDetailCommentList` → `ReviewResultActivityRow` → `ReviewTargets` |

## Principle

> Make the safe path the easy path. A reviewer who cannot see that a finding touches production
> treats it as if it does not. The row answers "where does this land?" before anything else.

## Decisions

**D1 · Two forms, one line.** A finding with four databases or fewer lists them all. One with
five or more shows a summary instead: a count per environment and the total, such as
`Prod 5 · Staging 5 · Test 6 · 16 databases`. The row's height no longer grows with the number of
databases, and no environment is hidden behind a count. Rejected: the first few database names
and a count, which BYT-10512 proposed. Which names come first is arbitrary, and the count can
still hide production. Rejected: expanding inline, which floods the comment.

**D2 · Each environment's badge once.** The short form puts the environment's badge before its
databases, and the summary and the popover's group headers carry one badge per environment. A
database chip or row carries none. The badge is `EnvironmentBadge`, as everywhere else
(invariant 9 of the plan comment card design). Rejected: a badge on every chip, which repeats the
same accent-colored badge many times and competes with the priority badge.

**D3 · Production first, then the latest rollout stage first.** Environments the workspace marks
as production come first, then the rest in reverse workspace order. The workspace's environment
order is the rollout order, so this puts the stage closest to production next. Then come
environments the workspace no longer lists, then unassigned databases, then unknown ones. Within
an environment, databases are ordered by name, then by full resource name.
A database's environment is its effective environment, falling back to its instance's
(`databaseEnvironmentName`). The title bar's production badges use the same function and keep the
workspace order.
The frontend orders; the server's order stays as it is, because a rule finding's fingerprint
hashes its sorted databases (`findingKey` in `backend/runner/reviewrun/rule_review.go`). Ordering
by environment there would change every fingerprint now and again whenever an environment is
reordered or a database moves, and threads would lose their state.
Rejected: production first, then the workspace order. Marking an environment as production needs
the environment-tier feature, so on other plans nothing is production, and the default order
(Test, then Prod) would put Test first. Rejected: ordering on the server.

**D4 · One button opens a popover.** The summary is one button, the shared `PopoverTrigger`. It
opens on click, Enter or Space. Focus moves into the popover; Escape or a click outside closes it
and returns focus to the button. Rejected: opening on hover, which does not work on touch and
cannot hold a search box. Rejected: a `Dialog`, whose scrim hides the plan. Rejected: a `Sheet`,
which the UX contract gives to creating or editing a resource.

**D5 · Every database, grouped.** The popover lists every database, with no cap and no "load
more", and mounts only the rows in view (`@tanstack/react-virtual`). Rows sit under environment
headers that stay pinned while the list scrolls. Each row is the engine, instance and database,
drawn by `DatabaseTargetDisplay`, so the instance always tells apart two databases of the same
name. The popover is 360px wide and its list at most 320px tall.

**D6 · Search in the browser.** Above ten databases a search box filters by database or instance
name, ignoring case. The database's name highlights the match. A group with no match is hidden and
each header counts its matches. When nothing matches, the list says so and offers Clear search.
Every database is already in the browser, so the contract allows filtering there (frontend-ux.md,
Without Pagination).

**D7 · Database groups.** When the spec targets a database group:

- A finding on every database the group matches hides the row, as a hand-picked list already
  does.
- A finding on some of them names the group and counts against it: `Database Group all-services
  · 16 of 22 databases`. The popover's header names the group and links to it.
- The count drops "of M" when the project samples its reviews below the group's size
  (`ciSamplingSize`), since the review read only some databases. It also drops "of M" when one of
  the finding's databases has left the group.
- The row stays hidden while the group loads, so a row is never shown and then hidden.

**D8 · Loading and errors.** Environments load before the workspace renders. Databases load
lazily: `ReviewTargets` asks for its own through `useTargetDatabases`, which shares one request
with the title bar's badges and with every other card asking in the same tick. Until they load,
the short form keeps the given order with no badges and the summary shows the total alone. A
database that cannot be read is Unknown, listed last, never Unassigned, so a database that may be
in production is never shown as having no environment.

**D9 · No motion of its own.** Of the shared overlays, only the sheet and the toast animate. The
popover appears as every other popover does, and filtering never animates, since the keyboard
drives it.

**D10 · Accessibility.** The button's accessible name reads the whole summary, for example
"Applies to, 18 databases, Prod 4, Staging 5, Test 4, Dev 3, Unassigned 2". The rows hold nothing
focusable, so the list itself takes focus for keyboard scrolling. Screen readers skip the pinned
headers, so each row names its environment in visually hidden text.

## States

| Finding's databases | Row | Popover |
| --- | --- | --- |
| None | Nothing | None |
| Exactly the spec's databases, or every database of its group | Nothing | None |
| A database group still loading | Nothing | None |
| 1 to 4 | Every database, grouped under one badge per environment | None |
| 5 to 10 | A count per environment, at most three named and the rest as "+N", then the total | Every database under environment headers |
| More than 10 | The same | A search box, then every database under environment headers |
| Part of a database group | The group's name, the counts, and "N of M databases" | The group's name and a link to it, then the list |
| Databases still loading | The given order with no badges, or the total alone | One list in the given order |
| Search with matches | Unchanged | The matching databases, under their headers |
| Search with no match | Unchanged | "No databases match" and Clear search |

## Known limits

- **Growth of a group.** A review run does not record which databases it read. "M" counts the
  databases the group matches now, so databases that joined after the review count too, and a
  finding that covered the whole group at review time shows as "N of M" instead of being hidden.
  Counting exactly needs the run to record the databases it reviewed.
- **Sampling.** When the project samples, the row says "N databases", without saying which of the
  group's databases the review skipped.
- **Search delay.** The shared `SearchInput` debounces its changes by 300ms. Filtering itself is
  instant.
- **Highlight.** A match is highlighted in the database name, not the instance name.

## Scope

One PR. It holds this doc and the frontend change, and needs no backend change.

- `productionTargets.ts`: `databaseEnvironmentName`, `groupTargetsByEnvironment`,
  `useTargetDatabases`, and `requestMissingDatabases` waiting for names another card's request
  holds. `CommentThreadCard.tsx` uses the hook for the title bar.
- `ReviewResult.tsx`: the two forms, the summary button, the popover, and the database group rules.
  `ReviewTargetList.tsx`: the popover's list, search and empty state.
- Five locale files for `plan.review.thread.targets.*`.
- Unit tests: `productionTargets.test.ts` for the order and the shared request,
  `ReviewResult.test.tsx` for both forms, the popover, search, focus and Escape, virtualization, and
  each database group case.

## Not in this PR

- Grouping the spec's Targets section by environment. It comes in its own PR.
- Ordering the title bar's production badges by D3. They keep the workspace order.
- Recording the databases a review run read, which exact "N of M" needs.
- A shared component for this list and the approval step reviewer list
  (`docs/design/approval-reviewer-list.md`). They share the pattern, not code.
