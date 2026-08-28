# Timestamp display — work queues vs history views

Relative timestamps ("x days ago") hide exactly the precision an audit reading needs
([BYT-10140](https://linear.app/bytebase/issue/BYT-10140)). The rule this doc lands on: **a surface
is either a work queue or a history view, and that classification decides its time display** — work
queues get GitHub-style relative time with a 30-day cap, history views get absolute date-time
always. A third kind of time is future-pointing — scheduled rollouts and expirations
([BYT-10023](https://linear.app/bytebase/issue/BYT-10023)) — and renders absolute with an explicit
timezone. Frontend display only; no backend, no API, no stored preference. Time *input* (the
schedule and expiration pickers) is explicitly out of scope for now.

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

The queue/history split covers *record* timestamps — times of things that already happened. A
**future-pointing time** is a third kind:

- **Operational time** — a scheduled rollout or an expiration, read in order to *act*: "exactly
  when will this fire or lapse". Display: absolute date-time **with an explicit timezone**, never
  relative-only. Relative wording ("in 7 hours") hides that the timezone question even exists;
  see the incident evidence under Operational times below.

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

**D5 (proposed) — Operational times: absolute with explicit timezone, minute precision.**
Scheduled rollout times and expirations render the absolute date-time with the short timezone name
in the visible string, not just the tooltip: "Sep 15, 2026, 9:00 AM GMT+8". Precision caveat:
minute-granular input covers only the `datetime-local` pickers. Day/second-count presets write
`now() + offset` with real sub-minute tails (`computeExpirationTimestamp` in `MembersPage.tsx`,
the SQL-editor access-grant drawer), so a minute display floors the enforced cutoff by up to 59
seconds — in the safe direction: the value never expires earlier than displayed, and the full
value stays in the tooltip (D6). Accepting D5 means accepting that bounded under-report;
alternatives are retaining seconds for expiration values, or normalizing preset writes to the
minute (a write-path change, out of scope for this display-only design). Driven by BYT-10023 —
see Operational times below. Proposed, pending confirmation; the time *pickers* that write these
values are explicitly deferred.

**D6 — Full date-time tooltip on every reduced display.**
Any timestamp that does not show the full form — relative, date-only after the switch, or the
compact history tier — carries the full absolute date-time with seconds and timezone in its
tooltip. This is the universal escape hatch that keeps every reduced cell recoverable.

**D7 (proposed) — History views carry two precision tiers.**
*A history row orients; a history record testifies.* Full precision (seconds + timezone,
`formatAbsoluteDateTime`) where the time itself is the evidence: the audit log — exportable, and
the export must match the screen — and single-record detail views. Compact precision (date +
hh:mm, no seconds, no timezone) where rows are scanned to locate a record inside one resource's
history: the embedded lists, where width is contended and D6 keeps full precision one hover away.
Objective divider: **exportable-as-evidence or a detail view → full; scannable embedded list →
compact.**

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
| Deploy current status (latest-run times) | `routes/project/plan-detail/components/deploy/DeployTaskHeader.tsx`, `DeployLatestTaskRunInfo.tsx` | Freshness | 30d switch |
| **Scheduled rollout pill** | `DeployTaskHeader.tsx` (task pinned to a run time) | **Operational** | **Absolute + timezone** |
| Schema sync status | `modules/sql-editor/components/SchemaPane/SyncSchemaButton.tsx`, `routes/project/ProjectSyncSchemaPage.tsx`, `components/database/DatabaseOverviewInfo.tsx` | Freshness | 30d switch |
| Agent chat | `modules/agent/components/AgentWindow.tsx` | Feed | 30d switch |
| **Database changelog** | `routes/project/database-detail/changelog/DatabaseChangelogTable.tsx` | **History view** | **Absolute always — compact tier (D7)** |
| **Database revisions** | `routes/project/database-detail/revision/DatabaseRevisionTable.tsx` | **History view** | **Absolute always — compact tier (D7)** |
| **Task-run history** | `routes/project/plan-detail/components/deploy/DeployTaskRunHistorySheet.tsx`, `routes/project/issue-detail/components/IssueDetailTaskRunTable.tsx` | **History view** | **Absolute always — compact tier (D7)** |
| Audit log | `components/AuditLogTable.tsx` | History view | Already absolute — unchanged |

## Operational times (BYT-10023)

The incident that motivates the third class: a customer scheduling a production rollout for that
night asked which timezone the schedule uses. Nobody on their side could tell from the product;
their two guesses — UTC+0 and the server's timezone — were both wrong (it is the operator's
*browser* timezone), and acting on the UTC+0 guess would have fired the rollout seven hours early,
mid-business-day. The product behavior is correct; the defect is that no visible surface says which
timezone applies. The docs side was fixed in
[bytebase.com#125](https://github.com/bytebase/bytebase.com/pull/125); this doc covers the display
side. The picker itself (input side) is deliberately not designed here.

Where operational times are displayed today:

| Surface | File | Today | Gap |
|---|---|---|---|
| Scheduled rollout pill | `routes/project/plan-detail/components/deploy/DeployTaskHeader.tsx` (task pinned to a run time) | `HumanizeTs` — relative ("in 7 hours"), tz only in tooltip | **The BYT-10023 display gap**: the one surface showing when a rollout will fire hides the timezone question entirely |
| Task-run waiting message | `frontend/src/lib/taskRun.ts` ("enqueued, will run at …") | `formatAbsoluteDateTime` interpolated into a plain i18n string | None — a plain-string context cannot host a tooltip, so it keeps the full-precision form (invariant corollary below); seconds do **not** drop here |
| SQL-editor access grant item, <24h remaining | `modules/sql-editor/components/AccessGrantItem.tsx` | Duration only ("expires in 3h20m"); the absolute value is discarded, no tooltip | Relative-only operational display — needs the D6 tooltip carrying the full absolute + timezone |
| Masking exemption expiration | `routes/project/ProjectMaskingExemptionPage.tsx` | dayjs `YYYY-MM-DD HH:mm` | No timezone, no seconds, not locale-aware |
| Role-grant expiration detail | `routes/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx` | dayjs `LLL` | No timezone |
| Member expiration preview | `routes/workspace/MembersPage.tsx` (`formatExpirationDate`) | `toLocaleDateString` + hour/minute | No timezone, no seconds |
| Member expiration table, access grants, IAM remind dialog, sample expiration, subscription expiry | `MembersPage.tsx`, `ProjectAccessGrantsPage.tsx`, `utils/accessGrant.ts`, `IAMRemindDialog.tsx`, `SampleExpirationAlert.tsx`, `stores/app/workspace.ts` | `formatAbsoluteDateTime` | None — already absolute + tz (seconds drop under D5's minute precision) |

Fixes under D5: the scheduled pill and the three bare-format expirations converge on the
operational format — absolute date-time + timezone at minute precision ("Sep 15, 2026, 9:00 AM
GMT+8"); relative age can move to the tooltip.

## Current absolute-time display inventory

For reference, absolute timestamps already appear in the product in three inconsistent families:

1. **`formatAbsoluteDateTime`** — locale-aware, seconds + short timezone ("GMT+8"). The dominant
   family: audit log, changelog detail page, revision detail panel, access grants, members
   expiration table, IAM remind dialog, sample-expiration alert, subscription expiry, task-run
   scheduled-time messages, SQL editor result panel, Monaco heartbeat, agent chat tooltip.
2. **Fixed dayjs strings** — time but no timezone, not locale-aware: masking exemption
   (`YYYY-MM-DD HH:mm`), role-grant details (`LLL`), SQL editor tab titles (`YYYY-MM-DD HH:mm:ss`).
3. **Ad-hoc `toLocaleDateString`** with hour/minute — no seconds, no timezone: the members-page
   expiration preview.

Two half-built versions of this design already exist as dead code: the unused `humanizeTs()`
30-day switch in `utils/util.ts`, and the never-passed `format="absolute"` branch of the date cell
in `IssueDetailTaskRunTable.tsx`. Both are evidence the need was felt before; both get subsumed.

Notably, the product's current pattern is *relative in the list, absolute in the detail* — the
changelog detail page and revision detail panel already render `formatAbsoluteDateTime` while
their list views render relative. D4 makes each list agree with its own detail view.

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

**History views**: absolute always; precision comes in two tiers under D7 (assignment proposed):

| History-view occurrence | Space | Tier |
|---|---|---|
| Audit log (workspace + project pages) | Dedicated page, exportable | **Full** — keep as is |
| Changelog detail page | Detail view | **Full** — keep as is |
| Revision detail panel | Detail view | **Full** — keep as is |
| Database changelog list | Full-width table | Compact |
| Database revision list | Full-width table | Compact |
| Task-run history sheet | 704px sheet — the space-constrained case | Compact |
| Issue-detail task-run table | Embedded table | Compact |

- **Full** = `formatAbsoluteDateTime`: "Aug 26, 2026, 2:03:22 PM GMT+8" / zh
  "2026年8月26日 14:03:22 GMT+8" (~30 characters).
- **Compact** = date + hh:mm, locale-aware: "Aug 26, 2026, 2:03 PM" / zh "2026年8月26日 14:03"
  (~21 characters). Seconds and timezone stay one hover away per D6.

**Operational times** (scheduled rollouts, expirations): a new `formatOperationalDateTime` helper
in `datetime.ts` — date + hh:mm + short timezone, locale-aware: "Sep 15, 2026, 9:00 AM GMT+8" / zh
"2026年9月15日 09:00 GMT+8". The timezone lives in the visible string — not only in the tooltip,
because the reader is about to act on the value and must not have to discover that a timezone
question exists. No seconds per D5 (`formatAbsoluteDateTime` is the alternative if D5's
minute-precision refinement is rejected). Relative age ("in 7 hours") may accompany it in the
tooltip.

## Implementation shape

- `HumanizeTs` adopts the switching behavior. The logic already exists as dead code: `humanizeTs()`
  in `frontend/src/utils/util.ts` (30-day switch) + `RELATIVE_THRESHOLD_MS` and
  `formatAbsoluteDate` in `frontend/src/utils/datetime.ts`. Consolidate into `datetime.ts`; delete
  the dead `humanizeTs`/`humanizeDate` pair in `util.ts` / `utils/v1/common.ts`.
- History views use a `mode` prop on `HumanizeTs` so the tooltip/i18n-resubscribe behavior stays
  shared: `mode="datetime"` (full, existing `formatAbsoluteDateTime`) and `mode="compact"` (a new
  `formatCompactDateTime` helper in `datetime.ts` — date + hh:mm, locale-aware). One canonical
  component, modes matching the principle one-to-one; the audit log can keep its direct call.
- Operational times render through the shared component, never as bare strings: `HumanizeTs` gains
  `mode="operational"` (`formatOperationalDateTime` in the cell, D6 tooltip carrying the full
  enforced value with seconds). The scheduled pill in `DeployTaskHeader.tsx` switches mode rather
  than dropping the component — its tooltip survives; the three bare-format expiration call sites
  and the six already-absolute ones adopt the same mode (open item 4), which is what makes the
  sub-minute preset tails recoverable. **Invariant: a reduced timestamp without a full-precision
  tooltip is a bug** — bare formatter calls are reserved for full-precision strings and exports.
  Corollary: a plain-string context that cannot host a tooltip — the i18n-interpolated task-run
  waiting message, exports, document titles — must embed the full-precision string, never the
  reduced one. The SQL-editor access grant item's <24h branch ("expires in 3h20m") is a relative
  operational display and adopts the same tooltip treatment as the pill.
- Tests: update `HumanizeTs.test.tsx` for the switch; sweep the `*.test.tsx` files that assert
  relative strings (`PlanDetailMeta.test.tsx`, `SchemaPane.test.tsx`,
  `DeployTaskRunHistorySheet.test.tsx`, `IssueCommentActivity.test.tsx`,
  `ProjectPlanDashboardPage.test.tsx`, `DatabaseChangelogTable.test.tsx`, …). New behavior gets new
  tests: threshold boundary (29d/31d), same-year vs cross-year, zh locale, history-view mode.

## What does not change

- The audit log (already correct).
- Duration display (`humanizeDurationV1` — elapsed times like "4.2s" on task runs and query
  results). Durations are elapsed quantities, not timestamps: they carry no calendar, timezone, or
  precision question, so this design leaves them untouched.
- Relative wording under 30 days.
- No user or workspace preference (revisit only if a customer asks for the opposite default —
  GitLab's model is the known shape for that).
- No backend or proto change.
- The time *pickers* (schedule rollout, expiration inputs — `datetime-local` fields writing browser
  local time): BYT-10023's input side. Deferred to its own design; the docs-side fix already landed
  in bytebase.com#125.

## Customer outcome check

- Historical tickets (> 30 days): issue list shows the real date — complaint resolved for genuinely
  old history.
- Changelog / revisions / task-run history: absolute date-time to the minute by default (compact
  tier, D7); seconds and timezone one hover away (D6). Full seconds remain the default on the
  evidence surfaces — the audit log and the changelog/revision detail views — so the
  year-month-day-hour-minute-second ask is met to the minute on record lists and fully on record
  details. If the customer pushes back specifically on visible seconds, the escalation is flipping
  those lists to the full tier — a one-line D7 assignment change, no model change.
- Queue rows 1–29 days old: still "x days ago" (D1/D3 trade-off, accepted because these are queue
  readings). **Risk**: if the reported "ticket history" reading includes *recent* issue-list rows,
  part of the complaint survives. Mitigation: tooltip; escalation path if it recurs is lowering the
  threshold (D3) or a GitLab-style preference (D1 rejected-for-now).

## Open items awaiting ruling

Confirmed since the first draft: release list = work queue; comment/activity timelines = feed;
D6 (full date-time tooltip on every reduced display). Still open:

1. **D7 tier assignment/format** — compact (date + hh:mm) vs full for the embedded history lists.
   Mockups of both options exist for the changelog table and the 704px task-run sheet;
   recommendation is compact per the orient/testify principle.
2. **D5 precision refinement** — operational times at minute precision (no seconds). Pickers
   write minutes, but day/second-count presets write `now() + offset` with sub-minute tails, so
   minute display floors the enforced cutoff by ≤59s (safe direction; full value in the tooltip).
   Alternatives: full seconds for expiration values, or normalizing preset writes to the minute
   (write-path change, out of scope here).
3. Scheduled rollout pill shows the operational format inline ("Sep 15, 2026, 9:00 AM GMT+8");
   relative age moves to the tooltip.
4. The three bare-format expiration displays (masking exemption, role-grant details, member
   preview) are normalized to the operational format in this effort. Alternative: file as
   follow-up cleanup.
5. Tooltip on *full* cells may show the relative age (inverse tooltip). Alternative: repeat the
   full string (GitHub parity).
