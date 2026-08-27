# Timestamp display — work queues vs history views

Relative timestamps ("x days ago") hide exactly the precision an audit reading needs
([BYT-10140](https://linear.app/bytebase/issue/BYT-10140)). The rule this doc lands on: **a surface
is either a work queue or a history view, and that classification decides its time display** — work
queues get GitHub-style relative time with a 30-day cap, history views get absolute date-time
always. Frontend display only; no backend, no API, no stored preference.

## Problem

Every timestamp outside the audit log renders through the shared `HumanizeTs` component
(`frontend/src/components/HumanizeTs.tsx`), which is **relative-forever**: "45 seconds ago" → "12
minutes ago" → "7 hours ago" → "N days ago" with no upper cap. A year-old issue reads "365 days
ago". The absolute time exists only in a hover tooltip, recoverable one row at a time. The audit
log is the sole exception — it renders absolute date-time always.

For someone reviewing historical tickets, relative wording past a certain age carries no usable
information: it cannot be correlated with an incident window, a changelog entry, or an external
audit request.

## Principle

> **A surface is either a work queue or a history view, and that classification decides its time
> display.**

- **Work queue** — read to decide *what needs attention now*. The primary time question is
  **freshness**. Display: GitHub-style relative time with a 30-day cap, then absolute date.
- **History view** — read as a *record of what already happened*. The primary time question is
  **exactly when**. Display: absolute date-time, always.

Surfaces that are neither literally a queue nor a record (activity feeds, "created N ago" meta,
sync freshness, agent chat) are freshness-first readings and follow the work-queue rendering.

## Research — how other products display timestamps

How to read this table: "Default display" is what a list row shows without interaction; "Switch" is
any built-in age-based change of format; "Escape hatch" is how a user recovers the other form. Rows
marked ✔ were verified against primary sources on 2026-08-26; unmarked rows are from product
knowledge and worth spot-checking before citing externally.

| Product / surface | Surface kind | Default display | Switch | Escape hatch | User setting |
|---|---|---|---|---|---|
| **GitHub** issues/PRs/commits ✔ | Work queue / feed | Relative ("3 days ago") | At **30 days** → absolute **date only** ("on Apr 1, 2014"; same-year drops the year) | Tooltip: full date-time | None |
| **GitLab** ✔ | Work queue / feed | Relative everywhere | None | Tooltip | Per-user "Use relative times" (uncheck → absolute everywhere) + 12/24-hour format |
| **Linear** issue list | Work queue | Compact relative ("3d") | None | Tooltip | None |
| **Jira** issue views | Work queue | Relative for recent | Absolute for older items | Tooltip | None |
| **Slack** messages | Feed | Contextual ("Today at 2:31 PM") | Older days show the date | Hover | 12/24-hour |
| **Sentry** issue stream | Queue/feed | Relative age | None in stream; detail view is absolute | Detail page | None |
| **Stripe Dashboard** payments | History / audit table | Absolute ("Jul 3, 2:35 PM") | — | — | None |
| **AWS CloudTrail** | Audit log | Absolute, uniform column | — | — | UTC/local |
| **Datadog** logs | Record/observability | Absolute (ms precision) | — | — | Timezone |

**Pattern**: mature products do not pick one global convention — they pick **per reading mode**.
Queues and feeds are relative (GitHub adds the 30-day age cap); records and audit tables are
absolute, uniform, and column-scannable. GitLab is the outlier that solves it with a user
preference — and its *default* is still relative, so an audit-reading user must discover the
toggle. Our principle matches the industry split; we get both halves right by default instead of
shipping a setting.

Note the original request (absolute *always*, with seconds) is stronger than GitHub's pattern. The
resolution below satisfies it on history views; on work queues it is intentionally traded for
queue scanability (D1).

## Decisions

**D1 — Mechanism: GitHub-style age switch on queues, not absolute-always, not a preference.**
The issue list is a work queue, and the plan list is becoming one. Freshness is the dominant
question there, so relative time under a cap is correct. Rejected: absolute-always on queues (kills
freshness reading); GitLab-style user preference (settings machinery, wrong default for whoever
hasn't toggled); doing nothing (the complaint is real — "365 days ago" is information-free).

**D2 — Absolute form after the switch: date only (GitHub parity).**
"Jul 12, 2026"; same-year rows drop the year ("Jul 12"). Time-of-day stays in the tooltip.
Prioritizes queue density; accepts that on queue surfaces the exact-to-the-second ask is met only
via hover (it is met fully on history views, where that reading actually lives).

**D3 — Threshold: 30 days (GitHub parity).**
Matches GitHub exactly and matches the existing `RELATIVE_THRESHOLD_MS = 30d` constant.
Consequence accepted: queue rows aged 1–29 days still read "x days ago". Rejected: 24-hour and
7-day thresholds (would kill the complained-about string sooner, but diverge from the familiar
GitHub behavior).

**D4 — Scope: switch everywhere + history-view carve-out.**
`HumanizeTs` itself gains the 30-day switch, so every freshness-first surface behaves GitHub-style.
The history views — database changelog, revision table, task-run history — flip to **absolute
date-time always**, consistent with the audit log. They are records of execution, not queues.

## Surface classification

Every current `HumanizeTs` call site, classified under the principle:

| Surface | File | Class | New display |
|---|---|---|---|
| Issue list | `components/IssueTable.tsx` | Work queue | 30d switch |
| Plan list | `routes/project/ProjectPlanDashboardPage.tsx` | Work queue | 30d switch |
| Release list / detail meta | `routes/project/ProjectReleaseDashboardPage.tsx`, `ProjectReleaseDetailPage.tsx` | Work queue / meta | 30d switch |
| Issue comments & activity | `components/issue-activity/IssueCommentActivity.tsx`, `routes/project/issue-detail/components/IssueDetailCommentList.tsx` | Feed | 30d switch |
| Review timeline / rejection banner | `routes/project/plan-detail/components/review/ReviewActivityTimeline.tsx`, `ReviewRejectionBanner.tsx` | Feed | 30d switch |
| Plan detail "created" | `routes/project/plan-detail/components/PlanDetailMeta.tsx` | Meta | 30d switch |
| Deploy current status | `routes/project/plan-detail/components/deploy/DeployTaskHeader.tsx`, `DeployLatestTaskRunInfo.tsx` | Freshness | 30d switch |
| Schema sync status | `modules/sql-editor/components/SchemaPane/SyncSchemaButton.tsx`, `routes/project/ProjectSyncSchemaPage.tsx`, `components/database/DatabaseOverviewInfo.tsx` | Freshness | 30d switch |
| Agent chat | `modules/agent/components/AgentWindow.tsx` | Feed | 30d switch |
| **Database changelog** | `routes/project/database-detail/changelog/DatabaseChangelogTable.tsx` | **History view** | **Absolute date-time always** |
| **Database revisions** | `routes/project/database-detail/revision/DatabaseRevisionTable.tsx` | **History view** | **Absolute date-time always** |
| **Task-run history** | `routes/project/plan-detail/components/deploy/DeployTaskRunHistorySheet.tsx`, `routes/project/issue-detail/components/IssueDetailTaskRunTable.tsx` | **History view** | **Absolute date-time always** |
| Audit log | `components/AuditLogTable.tsx` | History view | Already absolute — unchanged |

## Rendering specification

**Work-queue / freshness surfaces** (via `HumanizeTs`, which gains the switch):

| Row age | Rendered | Example (en) | Example (zh) |
|---|---|---|---|
| < 10 s | "now" | now | 现在 |
| < 1 min | relative seconds | 45 seconds ago | 45秒钟前 |
| < 1 h | relative minutes | 12 minutes ago | 12分钟前 |
| < 24 h | relative hours | 7 hours ago | 7小时前 |
| < 30 d | relative days | 6 days ago | 6天前 |
| ≥ 30 d, same year | absolute date, no year | Jul 12 | 7月12日 |
| ≥ 30 d, other year | absolute date with year | Jul 12, 2025 | 2025年7月12日 |

- Tooltip (both forms, unchanged contract): full absolute date-time with seconds and timezone —
  "Aug 26, 2026, 2:03:22 PM GMT+8" / zh "2026年8月26日 14:03:22 GMT+8".
- Future timestamps mirror on |age| (`Intl.RelativeTimeFormat` already signs correctly; the ≥30d
  branch shows the date).
- All strings locale-aware via `Intl` with the active i18n locale — no hardcoded formats, no new
  locale keys needed.

**History views** (changelog, revisions, task-run history): absolute date-time always, using the
audit log's exact format (`formatAbsoluteDateTime`): "Aug 26, 2026, 2:03:22 PM GMT+8" — seconds
included, which is where the exact-time ask lands fully. Tooltip retained (GitHub also tooltips its
absolute dates); it may additionally show the relative age — see defaults below.

## Implementation shape

- `HumanizeTs` adopts the switching behavior. The logic already exists as dead code: `humanizeTs()`
  in `frontend/src/utils/util.ts` (30-day switch) + `RELATIVE_THRESHOLD_MS` and
  `formatAbsoluteDate` in `frontend/src/utils/datetime.ts`. Consolidate into `datetime.ts`; delete
  the dead `humanizeTs`/`humanizeDate` pair in `util.ts` / `utils/v1/common.ts`.
- History views either call `formatAbsoluteDateTime` directly (as `AuditLogTable.tsx` does today)
  or `HumanizeTs` gains a `mode="absolute"` prop so the tooltip/i18n-resubscribe behavior stays
  shared. Prefer the prop — one canonical component, two declared modes, matching the principle
  one-to-one.
- Tests: update `HumanizeTs.test.tsx` for the switch; sweep the `*.test.tsx` files that assert
  relative strings (`PlanDetailMeta.test.tsx`, `SchemaPane.test.tsx`,
  `DeployTaskRunHistorySheet.test.tsx`, `IssueCommentActivity.test.tsx`,
  `ProjectPlanDashboardPage.test.tsx`, `DatabaseChangelogTable.test.tsx`, …). New behavior gets new
  tests: threshold boundary (29d/31d), same-year vs cross-year, zh locale, history-view mode.

## What does not change

- The audit log (already correct).
- Relative wording under 30 days.
- No user or workspace preference (revisit only if a customer asks for the opposite default —
  GitLab's model is the known shape for that).
- No backend or proto change.

## Customer outcome check

- Historical tickets (> 30 days): issue list shows the real date — complaint resolved for genuinely
  old history.
- Changelog / revisions / task-run history: full date-time with seconds by default — the
  year-month-day-hour-minute-second ask fully satisfied on the record surfaces.
- Queue rows 1–29 days old: still "x days ago" (D1/D3 trade-off, accepted because these are queue
  readings). **Risk**: if the reported "ticket history" reading includes *recent* issue-list rows,
  part of the complaint survives. Mitigation: tooltip; escalation path if it recurs is lowering the
  threshold (D3) or a GitLab-style preference (D1 rejected-for-now).

## Defaults awaiting veto

1. History-view cells keep the timezone suffix ("GMT+8"), matching the audit log. Alternative: move
   tz to tooltip for density.
2. Tooltip on absolute cells shows the full date-time (GitHub parity). Alternative: show the
   relative age instead (inverse tooltip).
3. Release list classified as work queue (switch), not history.
4. Comment/activity timelines classified as feed (switch) even on closed issues.
