# Plan comment link

Status: implemented, not merged · 2026-11-25 · [BYT-10604](https://linear.app/bytebase/issue/BYT-10604)

On a plan, the Activity timeline holds every comment, reply, review finding and event. None of
them can be pointed at. The only link to share is the plan's, and whoever opens it has to find
the comment. **Every Activity entry stored as a comment gets one address: the plan's URL followed
by `#issueComments/{id}`. Its timestamp is that link, the comment's menu copies it, and opening
it shows the comment whatever is folded in the way.** The fragment is the end of the comment's
API name, so an agent handed the link can read exactly that comment.

## Problem

Verified at `048977e797`, under `frontend/src/routes/project/plan-detail/` and
`frontend/src/components/issue-activity/`.

- **A timestamp is text.** `IssueCommentHeader` and `ThreadComment` draw it with `HumanizeTs`, a
  `span` with a tooltip. No comment has a copy action.
- **The page does not read the fragment.** Plan Detail takes `routeHash` only so that
  `useCanonicalPlanDetailRoute` keeps it when it rewrites the URL.
- **The router's fragment scroll cannot land.** `scrollToHash` in `NavigationScrollRestoration.tsx`
  looks the element up once, when the location changes. Comments load after the page mounts.
- **Five things can hide a comment.** Only Deploy is expanded once the plan has a rollout
  (`getDefaultActivePhases`). A long timeline folds its middle (`foldTimeline`). A review run with
  no finding still current shows no cards. A resolved thread's card starts folded. A thread with
  more than four replies shows the last two.
- **The page can reveal a thread, past three of the five.** `focusInActivity` expands Review and
  calls `requestActivityFocus`, and the timeline then unfolds, opens a hidden run and scrolls. It
  takes the name of a thread's first comment or of a run. It does not unfold a card or its
  replies, and nothing marks the entry it lands on.
- **One name is the `id` of several elements.** A thread's first comment gives its name to its
  timeline row, its card and its comment, and the statement editor draws the same card with the
  same ids. `getElementById` returns the first. That is the Activity row today only because the
  callers ask for a run, or for a thread the editor does not show.
- **The API cannot read one comment.** `IssueService` has `ListIssueComments`,
  `CreateIssueComment` and `UpdateIssueComment`. A reply comes back only from a list filtered by
  its root, which a client holding the reply's name does not know.

## Principle

> A comment has one address. It is built from names the product already has, it works from any
> state of the page, and a person and a program can both follow it.

## What other tools do

- **GitHub**, read from a pull request page. Every timeline entry's timestamp is a link whose
  `href` is only a fragment: `#issuecomment-…` for a comment, `#discussion_r…` for a comment on
  a line, `#pullrequestreview-…` for a review, `#event-…` for an event. The comment's "…" menu
  has "Copy link". The entry the fragment names is outlined.
- **Linear.** Its [changelog](https://linear.app/changelog/2019-10-14-improved-comments) made
  "the timestamps on comments and other issue history entries" links, to share one comment.
- **Why it matters.** Ben Balter lists
  [deep links](https://ben.balter.com/2015/11/18/tools-to-empower-open-collaboration/#tools-in-practice)
  among the patterns of open collaboration: a link to one comment grounds a discussion better
  than a link to the whole document. A URL
  [does not imply access](https://ben.balter.com/2015/11/12/why-urls/). It is checked on open.

What this design takes from them: the address is the page's URL plus a fragment that names the
entry, the timestamp is the link, events have addresses too, and the linked entry is marked.

## The address

```text
https://bytebase.example.com/projects/shop/plans/674#issueComments/2f1c9a52-7a3e-4f0e-9c1b-0d6b8a4f3e21
```

| Part | Is |
| --- | --- |
| Path | The plan's page, `/projects/{project}/plans/{plan}` |
| Fragment | The last two segments of the comment's name, `projects/{project}/issues/{issue}/issueComments/{id}` |

The plan names its issue (`Plan.issue`), so the comment's API name is that issue's name, a
slash, and the fragment.

| Entry in Activity | Address | Opening it marks |
| --- | --- | --- |
| A person's comment | Its own | Its row |
| An event: an approval, a plan update, an issue update | Its own | Its row |
| An event shown inside "N similar activities" | Its own | The one row that stands for it |
| A thread's first comment, from a person or a review | Its own | That comment, in its card |
| A reply | Its own | That reply, in its thread's card |
| A review run's line | The run's completion comment | The line |
| A row with no comment behind it, such as "created this plan" | None | Its timestamp stays text |

## Decisions

**D1 · One address, in the fragment of the plan's URL.** The fragment is spelled as the API
spells it, so nothing new is named. Rejected: the comment's name as the path,
`/projects/shop/issues/701/issueComments/…`, redirected to the plan as an issue URL is today. The
address bar must still name the comment after the redirect, so there would be two forms.
Rejected: the whole name in the fragment. It repeats the project and is a second path glued to
the first. Rejected: `#issuecomment-{id}`, GitHub's spelling, which the API does not use.
Rejected: a query, `?comment=`. Plan Detail moved its selection out of the query and into paths,
and a place within a page is what a fragment is for.

**D2 · The address lands in Activity.** A thread also shows under its lines in the statement
editor, but only while those lines are in the current statement. Activity holds every comment,
and its card has the SQL excerpt and "View in Statement". Rejected: landing in the editor when
the lines are current. One link would then open in two ways, and which one would change as the
statement is edited.

**D3 · The fragment works on every Plan Detail path.** The path selects a change, a stage or a
task. The fragment selects a comment. Copy link writes the plan's root path, so a link does not
carry what its sender had selected. A timestamp's `href` is the fragment alone, as on GitHub, so
a click does not change the selected change.

**D4 · Opening the address reveals the comment.** The page expands Review, shows the whole
timeline, opens the run, unfolds the card and shows all replies, as far as each is in the way.
Each step is what the reader's own click does, and it stays that way afterwards. Then the page
puts the comment in the middle of the viewport. It jumps there without animation, because a
reader who opens a link never saw where the scroll began. A comment already in view does not
move, so a click on its timestamp leaves the page still. The card's SQL excerpt takes its height
from the comment's line range before its sheet loads, so the comment does not shift after the
jump. The page does this once per navigation, never on a refresh of its data. A comment the
page has not loaded yet, as when a link is pasted into a tab that already shows the plan, is
looked for by listing the comments once more.

**D5 · The linked comment is outlined while the address names it.** A 2px outline in `accent`,
on the row or, inside a card, on the one comment and not the card. In a card it sits 4px inside
the card's edge with a 4px corner. That follows the card's 8px corner and keeps it apart from
the focus ring, which is also 2px in `accent`. It is read from the URL, so it moves when another
timestamp is clicked and goes when the fragment does. The mark is not for the eye only: on
landing, keyboard focus moves to the comment, and the comment carries `aria-current`. Rejected:
a highlight
that fades. A reader who looks a second late sees nothing, and it needs a timer and a
reduced-motion case.

**D6 · The timestamp is the link.** It becomes an `a` in the same `text-control-light`,
underlined on hover and focus. Its tooltip stays, and also opens on keyboard focus. Open in a
new tab and the browser's own
copy both work. This holds for every row in the table above that has an address, events
included. Rejected: a click that copies. A timestamp that looks like a link and is not one
breaks both.

**D7 · A comment's actions are in a menu.** An icon-only "…" `Button size="xs"` at the right
of the comment's author row opens the shared `DropdownMenu`. The button is
`appearance="secondary"`, the borderless neutral style the pencil has, so a thread does not show
a column of boxes or of colored marks. It is named with the comment's author, such as "Actions
for Lee Tanaka's comment", so one can be told from the next. The menu holds Copy link and, for
a reader who may edit, Edit, which is a pencil button in that row today. Copy link copies the
absolute URL and confirms with the "Copied" notification. Every comment then has one control in
that place, whatever the reader may do, as on GitHub. The menu is needed beside D6 because
`ThreadComment` hides its timestamp in a comment narrower than 18rem, which is a phone, and
because copying is the job the link exists for. Event rows have no actions and get no menu. The
agent help button on a review finding stays beside the menu. Rejected: a Copy link icon button
beside the pencil. A comment then carries a row of icons, and each new action adds one.
Rejected: showing the control on hover only, which touch does not have.

**D8 · An address that names nothing opens the plan.** The page opens as it would without the
fragment: nothing scrolls, nothing is outlined and no error shows. GitHub does the same.

**D9 · `GetIssueComment` reads the comment an address names.** A standard Get on
`/v1/{name=projects/*/issues/*/issueComments/*}`, classed `READ` for MCP. It reads under
`bb.issueComments.list`, as `GetTaskRun` reads under `bb.taskRuns.list`, so no role changes.
Its API comment says how a comment's web address maps to its name, which is where an agent
that searches the API finds it. Rejected: a sentence in the MCP server's instructions. They are
held to 2,048 characters for older clients, and 13 were left. Rejected: leaving an agent to
`ListIssueComments`. It finds a reply only by listing every thread.

## States

| Opening the address when | Result |
| --- | --- |
| The comment is off screen | Jumped to without animation, outlined, focused |
| The comment is already in view, as on a click on its timestamp | Outlined. The page does not scroll |
| Review is collapsed | Review expands first |
| The comment is in the timeline's folded middle | The whole timeline shows |
| The comment is a finding of a hidden run | The run opens |
| The comment is in a folded card | The card unfolds. The thread stays resolved |
| The comment is a reply behind "N replies hidden" | All replies show |
| The thread is also unfolded in the statement editor | The Activity card is the target. The editor does not move |
| The comment is newer than what the page has loaded | The comments are listed once more, then it lands |
| The fragment names no comment of this plan's issue | The plan, by D8 |
| The plan has no issue, or the reader cannot list its comments | The plan, by D8 |
| The fragment is not `issueComments/…` | Left to the router, as today |
| The page refreshes its data | No second scroll. The outline stays |

## Scope

One PR, in three parts. The first two are frontend only. The third is backend, and the first
two do not call it.

1. **Landing (D1 to D5, D8).**
   - Files, under `frontend/src/routes/project/plan-detail/`: `utils/commentLink.ts`, for the
     fragment; `hooks/useCommentLinkLanding.ts`, which reads `routeHash` once the comments have
     loaded; `shell/CommentLinkContext.tsx`, which carries the address and the request into
     Activity; `hooks/useActivityLanding.ts`, by which the entry asked for scrolls and focuses
     itself; `ReviewActivityTimeline.tsx`; `consolidateTimeline.ts`, to keep the names a merged
     row stands for; `CommentThreadCard.tsx` and `ThreadComment.tsx`, for unfolding and the
     outline; `StatementAnchorContext.tsx`, for the excerpt's height before its sheet loads.
     Shared: `IssueCommentActivity.tsx`, for the row's outline. The entry takes the landing
     through its own ref, so it is one element whatever else shares its `id`.
   - Tests: beside each of those files. They include every row of States; a reply among hidden
     replies; the same comment outside Activity, which is neither outlined nor landed on; no
     second landing on a data refresh; focus on the comment after landing; the excerpt's box at
     its height before the sheet loads. E2E:
     `frontend/tests/e2e/plan-detail/plan-detail-review.spec.ts` opens the address of a reply
     folded away in a resolved thread, then again with Review collapsed.
2. **Timestamp link and the comment menu (D6, D7).**
   - Files: `HumanizeTs.tsx`, which can draw its label as a link; `IssueCommentActivity.tsx`,
     whose header takes the link from the page and draws text without one;
     `components/threads/CommentActionsMenu.tsx`; `ThreadComment.tsx` and
     `ReviewActivityTimeline.tsx`, whose Edit buttons move into the menu; the five locale files,
     for "Copy link" and the menu button's name.
   - Tests: beside each of those files. They include: the `href` is the fragment alone; a row
     with no comment behind it has no link; the menu has Edit only for a reader who may edit;
     the menu button has no border and is named with the author; Copy link copies the root path
     on a spec path; a plain click changes only the fragment and keeps the scroll, and a
     modified click is left to the browser.
3. **`GetIssueComment` (D9).**
   - Files: `proto/v1/v1/issue_service.proto` and its generated code;
     `backend/api/v1/issue_service.go`; `backend/api/v1/testdata/mcp_method_classification.md`.
     The request's `name` carries a resource reference, which is how the permission check finds
     the comment's project. The "Read the change workflow" row in
     [MCP access policy](mcp-capability-ladder.md#keeping-the-rows-true) already says "Issues and
     comments", so its wording does not change.
   - Tests: `backend/api/v1`. They include: an event, a first comment and a reply each come back
     as `ListIssueComments` returns them; a name whose project or issue is not the comment's is
     `NOT_FOUND`; the permission is checked on the comment's own name.

## Not in this PR

- Issue Detail, where export, grant and create-database issues stay. It draws the same rows,
  and its timestamps stay text until it gets its own landing.
- Notifications that link to a comment. Webhook messages keep linking to the issue.
- A link to a line of a statement, or to a task run's log.
