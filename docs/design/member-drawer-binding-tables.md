# Member drawer binding tables

Status: implemented in this PR · 2026-11-01 · [BYT-10364](https://linear.app/bytebase/issue/BYT-10364)

On a project's Members page, Edit on a member opens a drawer with one card per role binding, and
each card lists the binding's databases, schemas and tables in a table. A long name widens its
column until the table runs out of the card, and every card lines its columns up differently.
**Each card's table keeps a fixed column layout fitted to the card, so every card opens with the
same columns; a name too long for its column ends in an ellipsis and shows in full on hover, and
the user can widen a column, in that card only.** Expiration moves from a column to the card's
header, because it belongs to the binding, not to a row.

## Problem

Verified on `main` at `cbaf903ae6`, in `EditMemberRoleDrawer` (`frontend/src/routes/workspace/MembersPage.tsx`):

- **Columns size to their content.** The table has no fixed layout, so the browser sizes each
  column from its longest value. A long schema or table name pushes Table and Expiration past the
  card's edge, and the whole drawer body scrolls sideways.
- **Every card has its own geometry.** Each card sizes its columns from its own rows, so the
  Database, Schema, Table and Expiration headers sit at a different x in every card.
- **Truncation does nothing.** `MemberDatabaseResourceName` truncates the instance and database
  names, but an auto-layout cell grows to fit them, so nothing is ever cut.
- **Expiration repeats on every row.** `getSingleBindingRows` copies the binding's one
  `expiredTime` onto each resource row, so a binding with ten tables shows the same expiration ten
  times, in the column that overflows first.
- **The drawer is narrower than the contract's tier.** It uses sheet width `medium` (640px); the UX
  contract gives `wide` (832px) to a sheet with nested tables.

## Principle

> A role card reads like the other tables in Bytebase, the plan list and Access grants: fixed
> columns that fit the space, long identifiers cut with the full value on hover, and columns the
> reader can widen. Nothing a card shows changes another card.

## Decisions

**D1 · Fixed columns, fitted to the card.** Each card's table uses `table-fixed` with Database,
Schema and Table columns. On open, the columns are spread across the card's width with
`distributeColumnWidths` and `fillableWidth` from `useColumnWidths`, as Access grants does, so
every card opens with the same columns and no card scrolls. The table sits in an `overflow-x-auto`
box inside the card, so a width the user drags past the card scrolls that card only. Rejected:
wrapping long names in fixed columns (#21699); it shows every name in full but gives rows uneven
heights and is not how Bytebase's other identifier tables read.

**D2 · Columns resize per card, and the widths are not kept.** Every column has the shared resize
handle. Dragging it changes that card only; widths reset when the drawer opens again. Rejected: one
set of widths shared by every card; a reader reviews one binding at a time, and sharing would tie
every card to state held by the drawer for no task the drawer serves.

**D3 · A cut name shows in full on hover.** Schema and Table cells use `EllipsisText`, which adds
the tooltip only when the value is cut. The Database cell keeps its engine icon, instance title and
database name, each an `EllipsisText` of its own, as `DatabaseTargetDisplay` does, so the database
name stays partly visible beside a long instance title and hovering either shows it in full. `*`
(every database, schema or table) never needs a tooltip.

**D4 · Expiration goes to the card header.** The header row reads: role title, the Expired badge
when expired (as today), then on the right "Expiration" and the value the column showed, the
`operational` timestamp or "Never", before the delete button. The table loses its Expiration
column. The strings already exist (`common.expiration`, `project.members.never-expires`).

**D5 · The drawer moves to sheet width `wide`.** The edit-member sheet goes from `medium` (640px) to
`wide` (832px), the contract's tier for a sheet with nested tables. The grant sheet the drawer
opens on top, and the workspace member sheet, keep their widths.

## States

| Binding scope | Name length | Card shows |
|---|---|---|
| Every database | — | One row: `*`, `*`, `*`. |
| A database | Fits | Engine icon, instance › database; Schema and Table `*`. |
| A database, schema and tables | Fits | One row per table, database and schema repeated, nothing cut. |
| Any | A name longer than its column | The name ends in `…`; hovering it shows the full value. |
| Any | A column dragged wider than the card | That card's table scrolls sideways; other cards unchanged. |

| Expiration | Header right side |
|---|---|
| None | "Expiration Never" |
| A future time | "Expiration" and the time, `operational` form |
| A past time | Expired badge after the role title, struck-through title, dimmed card (as today), and the time |

At phone width the sheet is the screen's width; columns shrink to fit, no column below its minimum
width, and a card whose minimum widths don't fit scrolls sideways.

## Scope

| File | Change |
|---|---|
| `frontend/src/routes/workspace/MemberBindingResourceTable.tsx` | New: one card's table (D1–D3). |
| `frontend/src/routes/workspace/MemberDatabaseResourceName.tsx` | A tooltip on the instance title or database name when it is cut (D3). |
| `frontend/src/routes/workspace/MembersPage.tsx` | Uses the new table, moves expiration to the header (D4), sheet `wide` (D5). |

Tests:

- `MemberBindingResourceTable.test.tsx`: Database, Schema and Table headers and no Expiration; a
  row per resource and `*` for an unscoped binding; dragging a column in one table leaves another
  table's widths unchanged.
- `MemberDatabaseResourceName.test.tsx`: the full instance title or database name on hover when it
  is cut, no tooltip when it fits.
- `MembersPage.test.tsx`: each card's header shows its expiration once, "Never" or the time.
- `tests/e2e/project-members/project-members.spec.ts`: with a long schema and table name, every
  card's columns line up and stay inside the card on open, hovering a cut table name shows it in
  full, and dragging a column in one card leaves the other cards' columns where they were.

## Not in this PR

- Grouping a binding's tables under one database and schema row (the issue's second open point).
- Reading a cut name on a touch screen, where there is no hover.
- The workspace member sheet and the grant sheet.
