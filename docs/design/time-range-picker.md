# Time range picker

Status: implemented, not merged · 2026-11-24 · [BYT-10563](https://linear.app/bytebase/issue/BYT-10563)

The list pages filter by time with one shared picker. Its calendar opened in a second popover on
top of its own To field and Confirm button, and a picked To day meant 00:00, which left that day
out of the result. **The picker becomes one popover: quick ranges that apply on click, beside one
range calendar with an Apply button. Below 640px the same parts sit in a tighter layout.** The
stored value is unchanged, so no URL, parser or filter changes.

The filter not applying at all on the project Issues page was a separate bug, fixed in
[#22206](https://github.com/bytebase/bytebase/pull/22206).

Mockup of the options: [Time Range Picker Options](https://claude.ai/artifact/8LzeQ7wzPRGrrBkDD84B35).

## Problem

Verified on `main` at `abfa69f897`, in `TimeRangePicker` (`frontend/src/components/TimeRangePicker.tsx`)
and `DateTimePicker` (`frontend/src/components/ui/date-time-picker.tsx`). The picker is on project
Issues, My Issues, Plans, Releases, Access Grants and Audit Log.

- **A popover on a popover.** From and To were each a `DateTimePicker`, whose calendar opens in a
  second popover under its field. Opened from From, it covers To and part of Confirm, which the
  report read as a missing confirmation.
- **The last day is left out.** A picked day gets the time 00:00 and the filter is
  `create_time <= To`. A range of Nov 15 to Nov 20 returns nothing created on Nov 20.
- **A range takes 7 to 8 clicks**, up to two of them only to close a calendar.
- **The trigger does not say what it filters.** It reads "Select" until a range is set.
- **No common ranges.** Audit Log opens on a 30-day default that the picker cannot name, or
  restore once changed.
- **Minutes are a 60-item list.**

## Principle

> Everything needed to choose a range is in view at once. A common range is one click, a custom
> range is two clicks and Apply, and the control that commits is never covered.

## Decisions

**D1 · One popover, one range calendar.** The shared `Calendar` runs in range mode. From and To
show the draft, and the outlined one is the end the next click sets: the first click sets From,
the second sets To, and a click before From starts over. The calendar always draws six weeks, so
the popover keeps its height while paging months.

**D2 · Quick ranges apply on click.** Today, Last 7 days, Last 30 days, Last 3 months, Last 6
months, Last 12 months, All time. They are one family of whole-day ranges that end tonight and
count today: on Nov 24, Last 7 days starts Nov 18 and Last 3 months starts Aug 25. Rejected: This
month and Last month, which mix calendar-aligned ranges into a list of rolling ones.

**D3 · All time means no filter.** It is marked when no range is set, and picking it removes the
range. The popover therefore has no Clear button. An x on the trigger clears without opening.

**D4 · A quick range is stored as two timestamps.** The trigger shows a quick range's name while
the stored range equals what that name means today, and the dates otherwise. A link opened a week
later shows the dates it was made for. Rejected for now: a rolling value such as `created:7d`. It
adds a second form of the `created` scope, read by `buildSearchParamsBySearchText` and by every
consumer of the range: the issue filter, the plan filter, Audit Log and Access Grants. It can be
added later without changing the UI.

**D5 · To defaults to the end of the day.** A picked To day gets 23:59 and is sent as the last
millisecond of that day. Any other typed time is exact to the minute. The filter formats bounds to
the second, so the final fraction of a second of the day stays outside the range.

**D6 · Time of day is typed and optional.** Each end has a 24-hour text field: `9`, `930` and
`9:30` all read as 09:30. An unreadable time is reported when the field loses focus and blocks
Apply until fixed. Rejected: the hour and minute lists.

**D7 · The trigger names its filter.** A `label` prop: "Created" on Issues, Plans, Releases and
Access Grants, "Time" on Audit Log, matching its column. After it comes the quick range's name or
the dates. Whole-day ranges show dates only.

**D8 · Below 640px the popover is compact.** Quick ranges become one row of chips that scrolls
sideways, From and To share one line, and the time fields appear after Add time. Day cells grow
from 32px to 40px. The popover is 306px wide and about 490px tall, so it fits a 375px screen.

**D9 · Controls keep the product's sizes.** Every control in the popover is `md` (36px, 14px
text); the chips are `sm`. On a phone, buttons and chips take taps across 44px through an
invisible hit area, without looking bigger. Input text stays 14px like the shared `Input`. The
zoom iOS Safari applies to a focused field under 16px is a matter for `Input`, not this picker.

**D10 · One line, between the two halves.** A vertical separator divides the quick ranges, which
act on click, from the form, which waits for Apply. The quick range rail is lightly tinted and
marks its selection with a check in a reserved leading slot, so labels stay aligned. There are no
captions and no rules inside the form; 12px gaps group it.

**D11 · Audit Log defaults to the Last 30 days quick range.** It was "end of today minus 30 days",
which no quick range equals, so the trigger could not name it.

**D12 · Calendar fixes that reach the single-date picker.** `Calendar` gains range styling and a
40px day size. Two fixes also apply to `DateTimePicker`. The month arrows are placed against the
calendar, on the month name's row, where before they were placed against the popover's edge. And
a selected today takes the selection's colors; before, today's grey background stayed under the
selection's white text.

## Approaches considered

The whole picker, compared in the mockup:

| Approach | What it is | Why not |
|---|---|---|
| Keep the two nested pickers | What shipped | The overlap, the 00:00 default and the click count are its structure. |
| One range calendar | D1 alone, no quick ranges | The last 7 days still takes 4 clicks. |
| **Quick ranges and a range calendar** | **Chosen** | |
| Two typed date-time fields | Type the dates, calendar as a helper | Needs a parser and error states; on a phone the keyboard covers the calendar. |

The phone layout of the chosen approach, in a 390 by 720 frame:

| Layout | Height | Why not |
|---|---|---|
| Stacked: the desktop parts in one column | 593px | Apply is cut off at the fold. |
| Two steps: a menu of quick ranges, then the calendar | 291px, then 556px | A second view that only the phone has; a custom range costs one more tap. |
| **Compact** | **491px** | **Chosen** |
| Bottom sheet over a dimmed page | 620px | Needs a bottom variant of `Sheet`; covers 86% of the screen; a modal on a phone and a popover on desktop. |

## States

| Stored range | Trigger shows |
|---|---|
| None | The label |
| Equals a quick range today | The label, the quick range's name, an x |
| Whole days | The label, `2026-11-15 → 2026-11-20`, an x |
| One whole day | The label, `2026-11-20`, an x |
| Has a time of day | The label, `2026-11-15 09:30 → 2026-11-20 23:59`, an x |
| Saved by the old picker, To at 00:00 | As "has a time of day" |

| In the popover | Result |
|---|---|
| No range set | All time is marked; From is outlined. |
| From picked | To is outlined; Apply is disabled. |
| From and To picked | The days between show as a band; Apply is enabled. |
| An unreadable time | The field is marked, a message says to use 24-hour time, Apply is disabled. |
| To earlier than From on the same day | "To is before From."; Apply is disabled. |
| Closed without Apply | The draft is dropped; it reopens from the stored range. |

## Scope

| File | Change |
|---|---|
| `frontend/src/components/TimeRangePicker.tsx` | Rewritten (D1–D10). New required `label` prop. |
| `frontend/src/components/time-range.ts` | New: quick ranges, time parsing, range bounds, the scope value. |
| `frontend/src/components/ui/calendar.tsx` | Range styling, `size`, month arrows, selected today (D12). |
| `frontend/src/components/IssueTable.tsx` | Pass `label` (D7). |
| `frontend/src/routes/project/ProjectPlanDashboardPage.tsx`, `ProjectReleaseDashboardPage.tsx`, `ProjectAccessGrantsPage.tsx` | Pass `label` (D7). |
| `frontend/src/components/AuditLogTable.tsx` | Pass `label`; default range (D11). |
| `frontend/src/locales/*.json` | `time-range.*` and `common.apply`, five locales. |

Tests:

- `time-range.test.ts`: each quick range's bounds, a range named only while it still equals a quick
  range, accepted and rejected times, the end-of-day bound.
- `TimeRangePicker.test.tsx`: a quick range applies on click and names the trigger; two picked days
  apply as whole days; the outlined end follows each pick; a typed time narrows the range; an
  unreadable time blocks Apply; month arrows page without losing the draft; All time and the x
  remove only the range; the compact layout hides time fields until asked.
- `ProjectIssueDashboardPage.test.tsx`: a picked range reaches the URL and the issue filter with
  the end-of-day bound.

A test cannot see one element covering another or one color overriding another, so the month
arrows and a selected today were also checked on the Issues page, at desktop and 375px widths.

## Not in this PR

- Quick ranges that roll forward (D4).
- A preview of the range while hovering toward the second day.
- Month and weekday names in the calendar follow the reader's language. They are English today, as
  in the single-date picker.
- `DateTimePicker`, beyond D12. The expiration picker and the rollout action panel still use it.
- A full-width trigger in phone toolbars. The trigger keeps its content width, so the compact
  popover is sized to the calendar.
- 16px input text on phones (D9).
