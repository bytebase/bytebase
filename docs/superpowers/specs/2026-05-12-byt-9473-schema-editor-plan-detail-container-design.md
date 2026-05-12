# Schema Editor — Plan Detail Container Redesign + BYT-9473 Polish

**Date:** 2026-05-12
**Linear:** [BYT-9473](https://linear.app/bytebase/issue/BYT-9473/schema-editor-display-not-work-properly)
**Branch:** `steven/byt-9473-schema-editor-display-not-work-properly`

## Summary

Replace the fixed-width drawer that hosts the schema editor inside the plan
detail page with a drawer that can be maximized to ~95% viewport. Bundle the
four polish issues called out in BYT-9473 (column icon mismatch, broken "new
table" action, missing add-column highlight, general unfinished feel) into the
same change so the editor lands in a coherent state.

The editor's *internals* (tree + table editor + DDL preview layout) are
explicitly out of scope. This is a container change plus targeted polish, not
a redesign of the schema editor itself.

## Motivation

Today `SchemaEditorSheet` opens a right-side `Sheet` at the `xlarge` width tier
(70rem ≈ 1120px). At that width the tree, table editor, and DDL preview
compete for horizontal space and the experience feels cramped — which is half
of what BYT-9473's "overall display not so polished" actually reports. The
other half is four concrete defects flagged in the same ticket.

A wider drawer is the smallest change that resolves the cramped feeling
without (a) violating the AGENTS.md rule that resource editing uses Sheets,
(b) restructuring the plan detail page, or (c) rewriting the schema editor
internals.

## Goals

- Give the schema editor enough horizontal room to feel polished without
  losing the plan context behind it.
- Fix the four BYT-9473 defects in one coherent change.
- Stay inside the existing design system (Sheet for editing, shared UI
  components, semantic color tokens).

## Non-Goals

- No redesign of the schema editor's internal layout (tree / table editor /
  DDL preview). The "diff-first / ER diagram / DDL split / step builder"
  options were explored and ruled out.
- No changes to other call sites of `SchemaEditorLite`. Only
  `SchemaEditorSheet` in plan detail gets the new container behavior.
- No persistence of the maximize toggle across sessions. Always resets to
  the default width on each open.
- No new schema editor tests. The component has no integration tests today;
  this change is not the place to add them.

## Design

### 1. Container behavior

**Default open** — same as today: right-side `Sheet` at the `xlarge` width
tier (70rem). No regression for users with current habits.

**New `huge` width tier** — extend `sheetContentVariants` in
`frontend/src/react/components/ui/sheet.tsx`:

```ts
huge: "w-[95vw]",
```

This leaves a 5vw strip on the left as the visual anchor the user asked for.

**Maximize toggle** — a `⤢` icon button (`Maximize2` / `Minimize2` from
`lucide-react`) in the sheet header, immediately left of the close X. Click
toggles between `xlarge` and `huge`. Tooltips: "Maximize" / "Restore" (new
locale entries).

**State** — local `useState(false)` inside `SchemaEditorSheet`. The body is
already mounted via `{open && <SchemaEditorSheetBody />}`, so closing and
reopening the sheet remounts the body and resets `maximized` to `false`. No
localStorage, no Pinia.

**Strip click semantics** — keep Base UI's default: clicking the scrim /
strip closes the sheet (same as every other sheet in the app). The strip is
purely a visual anchor; it is not a de-maximize handle. Toggle is via the
button only. This avoids hijacking Base UI's outside-click behavior and
keeps a single mental model for sheet dismissal across the app.

**Keyboard** — `Esc` closes (existing behavior, unchanged). No new shortcut
for maximize, to avoid collision risk with the schema editor's own
keybindings.

### 2. Sheet header actions slot

The shared `SheetHeader` currently renders `children` (a flex column for
title + description) and a fixed `SheetClose`. There's no place to inject a
secondary action like the maximize toggle.

Add an optional `actions` slot rendered immediately before the close button:

```tsx
function SheetHeader({ className, children, actions, ...props }) {
  return (
    <div className={...}>
      <div className="flex flex-col gap-y-1 min-w-0 flex-1">{children}</div>
      {actions ? (
        <div className="flex items-center gap-x-1 shrink-0">{actions}</div>
      ) : null}
      <BaseDialog.Close ...>
        <X className="size-4" />
      </BaseDialog.Close>
    </div>
  );
}
```

This is a reusable extension. Any other sheet that needs a header-level
action (e.g. a settings gear, an external-link icon) can use the same slot.
No existing callers need to change.

### 3. BYT-9473 polish fixes

#### 3.1 Column "C" icon mismatch

Today `AsideTree.tsx:472-477`:

```tsx
case "column":
  return <div className="size-4 text-center text-xs font-bold leading-4">C</div>;
```

SQL Editor uses a proper SVG `ColumnIcon` in
`frontend/src/react/components/sql-editor/SchemaPane/TreeNode/icons.tsx:133-150`.

**Fix** — create a shared module
`frontend/src/react/components/schema/icons.tsx` and move `ColumnIcon`
(plus its siblings if convenient) there. Import from both editors so they
cannot drift again. During the move, swap the raw `text-gray-500` for the
semantic `text-control-light`, per AGENTS.md "no raw color values".

#### 3.2 "New table" action does not work

`TableNameDialog` is a Base UI `Dialog` rendered inside an open `Sheet`. Both
portal into the same `overlay` layer family. The symptom in BYT-9473 is
consistent with the inner Dialog losing focus / event propagation to the
parent Sheet's focus trap — a known Base UI nested-modal hazard.

**Fix** — replace the nested Dialog with an inline name-entry popover
anchored to the schema's "New table" menu item. Specifically:

- Trigger from the tree context menu opens a `Popover` (Base UI) anchored to
  the menu item, containing the same `Input` + Cancel/Create buttons.
- No portal nesting inside the overlay family, no focus trap conflict.
- Rename flow can keep the inline popover as well, or stay as a Dialog if
  it is invoked from a context where no Sheet is open. Default: convert
  both to the popover for consistency.

This is meaningfully lighter than repairing the nested focus trap and
matches recent React UI patterns elsewhere in the app.

#### 3.3 No highlight on "Add column"

Today `TableEditor.tsx:69-92` pushes the new `ColumnMetadata` and marks it
`created`. The row's status text turns green but the row blends in; no
animation, no scroll, no focus shift.

**Fix** — three small additions in `handleAddColumn`:

1. Queue a scroll-to via the existing
   `scrollStatus.queuePendingScrollToColumn(...)` (already used on click).
2. Set `data-just-added="true"` on the new row. A Tailwind keyframe
   (`animate-row-flash`) fades a `bg-success/10` background over 1.2 s and
   removes the attribute on animation end.
3. Focus the Name input on the new row so the user can immediately type.

Define `animate-row-flash` in `frontend/src/assets/css/tailwind.css` via
`@keyframes` + `@utility`. Uses the semantic `--color-success` token.

#### 3.4 General polish

Tightly scoped — only changes that read as "unfinished" today:

- Replace `<div>C</div>`-style bare text status indicators with shared
  `Badge` for create / update / drop markers in the tree. Status text-color
  alone (`text-success` / `text-warning` / `text-error`) is too quiet.
- Toolbar density pass on the editor body: current `gap-y-3` is uneven
  against the database combobox. Tighten to `gap-y-2` and align baselines.
- Empty-state visual when the selected database has no tables yet (today
  the panel is silent and looks broken).

### 4. Files touched

```
frontend/src/react/components/ui/sheet.tsx
  + add "huge" width tier (w-[95vw])
  + add optional `actions` slot to SheetHeader

frontend/src/react/pages/project/plan-detail/components/SchemaEditorSheet.tsx
  + maximized state + ⤢ Maximize2/Minimize2 toggle wired into SheetHeader actions
  + width = maximized ? "huge" : "xlarge"
  + new locale keys for "Maximize" / "Restore" tooltips

frontend/src/react/components/schema/icons.tsx                          (new)
  + house ColumnIcon (+ siblings if appropriate); text-control-light not text-gray-500

frontend/src/react/components/SchemaEditorLite/Aside/AsideTree.tsx
  + replace <div>C</div> with the shared ColumnIcon
  + wrap status text with Badge for create / update / drop markers

frontend/src/react/components/SchemaEditorLite/Modals/TableNameDialog.tsx
  → convert to an inline Popover-based component (rename file accordingly,
    e.g. TableNamePopover.tsx); update call sites

frontend/src/react/components/SchemaEditorLite/Panels/TableEditor.tsx
  + handleAddColumn: queue scroll, set data-just-added, focus Name input

frontend/src/assets/css/tailwind.css
  + @keyframes + @utility for animate-row-flash (1.2s bg-success/10 fade)

frontend/src/locales/*.json
  + Maximize, Restore tooltip strings
```

### 5. Testing

**Manual** — open a plan, click "Schema editor":

- Default opens at `xlarge` width (no regression).
- ⤢ toggles to ~95vw; ⤢ again restores; closing and reopening always lands
  back at `xlarge`.
- 5vw strip click closes the sheet (same as today's scrim behavior).
- Tree column rows show the new `ColumnIcon`, visually identical to SQL
  Editor.
- "New table" creates a table, opens a tab, scrolls to it; works from
  every entry point.
- "Add column" — new row scrolls into view, flashes briefly, Name input is
  focused.
- `Esc` closes the sheet from both `xlarge` and `huge` modes.

**Automated** — none required for this scope. Existing checks cover:

- `pnpm --dir frontend type-check`
- `pnpm --dir frontend check` (lint, biome, layering scanner)
- `pnpm --dir frontend test` (existing tests must still pass)

**i18n** — confirm no hardcoded display strings; the two new tooltip
strings are added to `frontend/src/locales/`.

## Trade-offs Considered

- **Full-page route** instead of a maximizable drawer — explicitly rejected.
  Loses plan context; routing overhead is wrong for a short-lived
  compose-then-insert flow.
- **Centered modal Dialog** — explicitly rejected. Violates AGENTS.md
  ("Sheet for editing, Dialog for confirmations").
- **Inline expansion** inside the plan detail statement section — explicitly
  rejected. Crowds vertical space and competes with the SQL textarea.
- **Split pane** in plan detail — explicitly rejected. Restructures the plan
  detail page for one feature; cost > benefit.
- **Persisting maximize across sessions** — rejected. Reset on each open
  keeps state model simple; users who always want maximized can ask later.

## Open Questions

None at design time. Implementation may surface a focus-trap edge case in
section 3.2 (inline popover inside a sheet) — the fallback is to leave
rename as a Dialog and convert only the create path.
