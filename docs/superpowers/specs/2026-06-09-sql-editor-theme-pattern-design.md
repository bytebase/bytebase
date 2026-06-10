# SQL Editor Theme Pattern — Design

**Date:** 2026-06-09
**Status:** Approved design, pending spec review
**Scope:** SQL Editor only (UI chrome + Monaco editor surface + result views)

## Goal

Extract the colors the SQL Editor uses today into a **data-driven theme model** and
add a **runtime preset switcher** scoped to the SQL Editor. The current look becomes
the `light` preset (zero visual change as the default); a `dark` preset plus several
named pre-defined themes ship alongside it. The model is shaped so that user-defined
**custom themes** can be added later with no re-architecture — a custom theme is just
another theme object.

This phase ships a **catalog of six pre-defined preset themes** (below), each with its
authentic Monaco syntax palette. Custom-theme editing UI, a "System" option, and
backend persistence are explicitly out of scope (see [Future work](#future-work)).

### Theme catalog (v1)

| id | name | isDark | monacoBase | notes |
| --- | --- | --- | --- | --- |
| `light` | Default Light | false | `vs` | today's `:root`/`bb` look, verbatim — the default |
| `dark` | Default Dark | true | `vs-dark` | Bytebase dark, derived from existing `dark:`/`bb-dark` |
| `solarized-light` | Solarized Light | false | `vs` | Ethan Schoonover palette |
| `solarized-dark` | Solarized Dark | true | `vs-dark` | Ethan Schoonover palette |
| `monokai` | Monokai | true | `vs-dark` | vivid green/pink/orange on near-black |
| `nord` | Nord | true | `vs-dark` | arctic blue-gray |

## Locked decisions

| Decision | Choice |
| --- | --- |
| End goal | Build the foundation for runtime-switchable themes |
| Scope | SQL Editor only — chrome + Monaco + result views |
| Granularity | Preset themes only — 6 pre-defined themes in v1 (catalog above) |
| Syntax fidelity | Each named theme ships its full, authentic Monaco syntax palette |
| Token scoping | Scoped override — re-declare `--color-*` on the SQL Editor container |
| Persistence | localStorage, client-only, workspace-scoped |
| Theme representation | Approach A — theme is a plain JS token map; both layers derive from it |
| Admin (terminal) mode | A **nested theme scope**; effective theme = `selected.isDark ? selected : dark` |

### Why Approach A (data-driven) over a static CSS-class model

A theme is **data**: a `{ token: "r g b" }` object. The chrome consumes it by writing
the tokens as inline CSS custom properties on the container; Monaco consumes it via a
theme **generated from the same object**. Presets and (future) custom themes are the
same shape and the same code path — custom is just an object we did not ship. A static
model (a `.sqle-theme-dark` CSS class + a hand-authored Monaco theme file) cannot
represent values unknown at build time, so custom themes would force re-introducing
the data-driven mechanism anyway, on top of two diverging sources of truth.

## Current state (as found)

- **Tokens.** Semantic CSS variables live at `:root` in
  `frontend/src/assets/css/tailwind.css` (`--color-control*`, `--color-accent*`,
  `--color-main*`, status colors, `--color-block-border`, `--color-background`,
  `--color-dark-bg`, `--color-matrix-green*`). Tailwind utilities resolve to
  `rgb(var(--color-x) / <alpha-value>)` (`frontend/tailwind.config.js`), so overriding
  a variable on a parent element cascades to all descendants. **This is the mechanism
  the whole design relies on.**
- **Monaco.** Two themes (`bb` light, `bb-dark`) are defined once in
  `frontend/src/react/components/monaco/core.ts` via `initializeTheme()`; they read
  CSS vars from `:root` through `callCssVariable()`. `monaco.editor.setTheme()` is
  global and is currently called once per editor at construction
  (`MonacoEditor.tsx:346`). The admin terminal uses `vs-dark` intentionally.
- **Settings store.** `useSQLEditorEditorStore` (Zustand) in
  `frontend/src/react/stores/sqlEditor/editor.ts` already persists prefs to
  localStorage via `safeRead`/`safeWrite`, workspace-scoped through
  `frontend/src/utils/storage-keys.ts`.
- **Root container.** `<div className="sqleditor--wrapper …">` in
  `SQLEditorHomePage.tsx:187` wraps the editor body, but **not** everything: portaled
  overlays (FAB/sidebar → `getLayerRoot("overlay")`) and `RequestDrawerHost`
  (`SQLEditorLayout.tsx:62`, a sibling above the wrapper) render outside it. The scoped
  CSS-variable attach point is therefore the **layout** (`SQLEditorLayout`), with
  per-portal re-application (see Architecture §4 → Portals & sibling hosts).
- **Admin (terminal) mode is a nested dark subtree, not whole-editor dark.** A tab's
  `mode` (`"WORKSHEET"` | `"ADMIN"`, in the tab store) drives `Panels.tsx:162` to swap
  the code panel between `StandardPanel` and `TerminalPanel`. Only `<TerminalPanel>`
  (outermost `<div class="… bg-dark-bg">` at `TerminalPanel.tsx:206`) goes dark — its
  `EditorAction`, `CompactSQLEditor` (Monaco `vs-dark`, `CompactSQLEditor.tsx:323`), and
  `ResultView` (`dark` prop hardcoded `true`, `TerminalPanel.tsx:236`). The surrounding
  chrome (tabs, schema pane, connection pane) stays light. This subtree is the natural
  **nested scope boundary** for the theme system.
- **Hardcoded colors.** Many SQL Editor components bypass the tokens with raw
  palette classes (`bg-gray-200`, `border-zinc-500`), `dark:`-prefixed variants, and
  inline `rgb()`/hex. These must be migrated for scoped theming to work end-to-end
  (full inventory in [Color migration](#5-color-migration-prerequisite)).

## Architecture

### 1. Theme token model

A theme has **two layers**: the **chrome** layer (CSS-variable token map that re-themes
the SQL Editor UI) and the **Monaco** layer (editor-surface colors + syntax palette
that the chrome `--color-*` set does not cover). A named theme like Monokai is defined
largely by its Monaco layer.

```ts
// frontend/src/react/components/sql-editor/theme/types.ts
type RGB = string;   // "r g b" for chrome tokens, e.g. "82 82 91"
type Hex = string;   // "#rrggbb" for Monaco (Monaco consumes hex)

interface SQLEditorTheme {
  id: string;            // "light" | "monokai" | … (stable, persisted)
  name: string;          // i18n key for the display label
  isDark: boolean;       // for the few spots that need a boolean (Monaco base, 3rd-party widgets)
  monacoBase: "vs" | "vs-dark";
  tokens: Record<SQLEditorThemeToken, RGB>;   // chrome layer
  editor: EditorChromeColors;                 // Monaco surface (bg, gutter, selection, cursor, lineHighlight, …)
  syntax: SyntaxPalette;                       // Monaco token colors (keyword, string, comment, number, type, …)
}
```

- **`tokens`** — the SQL-Editor subset of the existing `--color-*` names (control\*,
  accent\*, main\*, background, block-border, link-hover, status colors,
  matrix-green\*, dark-bg). Names are **unchanged**; we only override values per theme,
  so existing `text-control` / `bg-control-bg` classes keep working.
- **`editor`** — Monaco `colors{}` entries: `editor.background`, `editorGutter.*`,
  `editor.selectionBackground`, `editorCursor.foreground`,
  `editorLineNumber.foreground`/`activeForeground`, line-highlight, etc.
- **`syntax`** — a normalized palette (`comment`, `keyword`, `string`, `number`,
  `type`, `function`, `variable`, `operator`, `delimiter`, `predefined`) that
  `buildMonacoTheme` maps onto Monaco token rules (incl. SQL scopes like
  `predefined.sql`, `operator.sql`, `string.sql`). Defining `syntax` once per theme,
  language-agnostic, keeps each preset to a single small palette rather than per-grammar
  rule lists.

### 2. Presets

One file per theme under `frontend/src/react/components/sql-editor/theme/presets/`
(`light.ts`, `dark.ts`, `solarized-light.ts`, `solarized-dark.ts`, `monokai.ts`,
`nord.ts`), re-exported from an `index.ts` as an ordered `PRESETS` array. One file per
theme keeps each palette small and reviewable, and makes adding a theme a single new
file.

- **`light`** — `tokens` copied **verbatim** from today's `:root`; `editor`/`syntax`
  reproduce the current `bb` look so the default render is byte-for-byte identical to
  current. `monacoBase: "vs"`.
- **`dark`** — derived from existing `dark:` choices (`gray-700/800`, `zinc-500/600`,
  `--color-dark-bg`) and `bb-dark`. Starting chrome palette below; **tunable**, final
  values confirmed visually during implementation.

  | Token | Light (current) | Dark (proposed start) |
  | --- | --- | --- |
  | `--color-background` | `255 255 255` | `30 30 30` (dark-bg) |
  | `--color-control-bg` | `243 244 246` | `55 65 81` (gray-700) |
  | `--color-control-bg-hover` | `229 231 235` | `75 85 99` (gray-600) |
  | `--color-control` | `82 82 91` | `229 231 235` (gray-200) |
  | `--color-control-light` | `113 113 122` | `209 213 219` (gray-300) |
  | `--color-control-placeholder` | `161 161 170` | `156 163 175` (gray-400) |
  | `--color-control-border` | `209 213 219` | `113 113 122` (zinc-500) |
  | `--color-block-border` | `229 231 235` | `82 82 91` (zinc-600) |
  | `--color-main` | `24 24 27` | `244 244 245` |
  | `--color-accent` | `79 70 229` | `129 140 248` (indigo-400) |
  | status colors | unchanged hue, may lighten slightly for contrast on dark bg | |

- **`solarized-light` / `solarized-dark` / `monokai` / `nord`** — authored from each
  theme's published palette. Each derives its **chrome** `tokens` from its base/bg and
  accent colors (so the SQL Editor UI matches), plus its authentic **`editor`** and
  **`syntax`** values. Reference palettes:
  - Solarized: base03–base3 + accent set (yellow/orange/red/magenta/violet/blue/cyan/green)
  - Monokai: bg `#272822`, fg `#f8f8f2`, green `#a6e22e`, pink `#f92672`, orange `#fd971f`, blue `#66d9ef`, comment `#75715e`
  - Nord: Polar Night `#2e3440…#4c566a`, Snow Storm `#d8dee9…#eceff4`, Frost `#8fbcbb/#88c0d0/#81a1c1/#5e81ac`, Aurora accents

  Each theme must define **every** required chrome token (no holes) so it renders
  coherently; a `validateTheme()` helper + a unit test enforce completeness across the
  whole catalog.

### 3. Store & persistence

Extend `useSQLEditorEditorStore`:

- State: `themeId: string` (default `"light"`).
- Action: `setThemeId(id)`.
- Persistence: read on init / write on change via the existing `safeRead`/`safeWrite`
  + a new workspace-scoped key in `storage-keys.ts`, e.g.
  `storageKeySqlEditorTheme(scope)` → `bb.sql-editor.theme[.${workspaceId}]`,
  matching the current scoping scheme. Unknown/invalid persisted id falls back to
  `"light"`.

No backend or proto changes.

### 4. Applying the theme — context + nested scopes

Theming flows through a small **React context** so a subtree can override the active
theme (this is what makes admin mode compose cleanly):

```ts
const SQLEditorThemeContext = createContext<SQLEditorTheme>(/* light */);
const useSQLEditorTheme = () => useContext(SQLEditorThemeContext);
```

A `<SQLEditorThemeScope theme={…}>` component does two things for whatever it wraps:
1. provides the theme via context (so descendants can read `theme.isDark` / the Monaco
   theme name), and
2. applies the theme's `tokens` as inline CSS custom properties on its container
   element. Because every SQL Editor component uses the semantic Tailwind classes,
   the subtree re-themes via CSS cascade.

**Root scope.** Wrap the SQL Editor at the **layout level** — `SQLEditorLayout`,
around both the route content (`SQLEditorRouteShell` → `SQLEditorHomePage`) **and**
`RequestDrawerHost` (`SQLEditorLayout.tsx:62`) — in a `SQLEditorThemeScope` whose theme
is the selected preset (from `useSQLEditorEditorStore.themeId`). The rest of Bytebase
keeps the global `:root` values; only the SQL Editor subtree is re-themed. (An earlier
draft scoped only `.sqleditor--wrapper`; that misses SQL-Editor UI rendered outside that
subtree — see **Portals & sibling hosts** below.)

**Portals & sibling hosts.** Scoped CSS variables re-theme via **DOM cascade**, but
React portals mount their DOM elsewhere: **React context still propagates through a
portal, CSS custom properties do not.** Several SQL-Editor surfaces render outside the
chrome DOM subtree and would otherwise keep the global (light) `:root` tokens under a
dark/named theme:

- `SQLEditorHomePage` portals the mobile FAB and the sidebar overlay to
  `getLayerRoot("overlay")` (`SQLEditorHomePage.tsx:160-182, 191-208`).
- `AccessGrantRequestDrawer` (rendered by `RequestDrawerHost`, `RequestDrawerHost.tsx:74`)
  is a `Sheet` that portals to the overlay layer **and** hosts its own `MonacoEditor`
  (`AccessGrantRequestDrawer.tsx:262`).

The layer roots are **app-global** (shared with the rest of Bytebase), so we cannot wrap
them in a SQL-Editor-only scope. Instead, each portaled SQL-Editor surface re-applies the
theme on its own container: wrap the portal's children in a nested `SQLEditorThemeScope`
whose theme comes from `useSQLEditorTheme()` (read through the context that the portal
preserves) so it re-writes the `--color-*` vars onto the portaled DOM. The request
drawer, now under the layout-level root scope, reads the theme from context: its chrome
re-applies vars via a scope on the `Sheet` content, and its Monaco reads the scope theme
(`monacoThemeName(useSQLEditorTheme())`) like every other editor. No new global theming —
only the specific SQL-Editor portal subtrees re-apply vars.

**Nested admin scope.** Wrap `<TerminalPanel>`'s root div (`TerminalPanel.tsx:206`) in a
nested `SQLEditorThemeScope` whose theme is the **resolved admin theme**:

```ts
resolveAdminTheme(selected) = selected.isDark ? selected : presets.dark;
```

Its inline CSS-var overrides win over the parent scope for the terminal subtree (CSS
cascade), so the terminal + its `ResultView`s render dark (or in the selected dark
theme), while the surrounding chrome keeps the selected theme. Worksheet mode renders
under the root scope only — no nesting.

This is the **replacement for the `dark` boolean prop**: `ResultView` (and anything
else that needs a light/dark signal) reads `useSQLEditorTheme().isDark` instead of a
prop, and its styling comes from the scope's tokens rather than `dark:` variants. The
`dark` prop and the `dark && "dark bg-dark-bg"` class at `ResultView.tsx:237` are
removed; `TerminalPanel`'s hardcoded `bg-dark-bg` becomes the token-driven
`bg-background` (the nested dark theme sets `--color-background` to the dark value).

**Monaco.** In `monaco/core.ts`, register one Monaco theme **per preset** via
`buildMonacoTheme(preset)`, which maps the preset's `editor` colors → Monaco `colors{}`
and its `syntax` palette → Monaco token `rules[]` (including SQL scopes). This replaces
the current `callCssVariable()`/`:root` reads — required because Monaco is
canvas-rendered and cannot see container-scoped CSS overrides, and because named themes
carry their own syntax colors rather than tinting `vs`/`vs-dark`. All six themes are
defined at init (`monaco.editor.defineTheme(\`bb-${id}\`, …)`).

Each Monaco editor picks its theme from the **nearest theme scope**: it reads
`useSQLEditorTheme()` and applies `bb-${theme.id}`. The normal editor resolves to the
selected theme; the terminal's `CompactSQLEditor` (currently hardcoded `vs-dark` at
`CompactSQLEditor.tsx:323`) resolves to the admin theme. A `useEffect` calls
`monaco.editor.setTheme` whenever the scope's effective theme changes (today set once at
`MonacoEditor.tsx:346`).

> **Monaco constraint:** `monaco.editor.setTheme` is **global** — Monaco cannot paint
> two different themes on screen at the same time. This is acceptable because
> `Panels.tsx:162` renders only one code panel per active tab — worksheet **or**
> terminal, never both — and the chrome contains no standalone Monaco editor. So at any
> moment a single global theme (the active scope's) is correct. The terminal panel's
> several `CompactSQLEditor` instances all share that one admin theme. If a future
> layout ever shows worksheet and terminal Monaco simultaneously, this assumption must
> be revisited.

### 5. Color migration (prerequisite)

Scoped overrides only reach components that use the tokens. The following bypass them
and must be migrated. Two rules:

1. Replace raw palette classes / inline colors with the semantic token equivalent.
2. **Delete `dark:`-prefixed variants** — dark theming now flows through token
   *values* supplied by the active (root or nested) theme scope, not a global `.dark`
   class. The `dark` boolean prop threaded through `ResultView` is removed; consumers
   read `useSQLEditorTheme().isDark` where a boolean is genuinely needed (e.g. Monaco
   theme, third-party widgets). In admin mode the result grid renders dark because it
   sits inside the nested admin scope, not because of `dark:` classes.

**Result views (currently `dark:`-driven):**
- `ResultView/VirtualDataTable.tsx` — lines 271, 275, 311, 313, 397, 436, 496
  (grid borders, header bg/text/hover, data cells, resize handle)
- `ResultView/VirtualDataBlock.tsx` — lines 147, 153, 166, 237
  (block title/content/card bg, delete button)
- `ResultView/SelectionCopyTooltips.tsx` — lines 56, 62, 75, 153
  (selection toolbar, info icon, copy button, menu item)
- `ResultView/SingleResultView.tsx` — lines 602, 617 (copy-cell buttons)
- `ResultPanel/DatabaseQueryContext.tsx` — lines 75, 93 (`bg-white/80` +
  `dark:bg-black/80` overlays → token-based overlay)

**Standard-mode raw palette colors:**
- `ConnectionPane/DatabaseHoverPanel/DatabaseHoverPanel.tsx` — 97, 114, 124, 133, 142, 149
- `ConnectionPane/ConnectionPane.tsx` — 680 (`bg-white/75` loading overlay)
- `SQLEditorHomePage.tsx` — 164 (`bg-white` FAB), 194 (`bg-black/40` backdrop)
- `SQLEditorLayout.tsx` — 58 (debug panel; low priority, dev-only)
- `SchemaPane/FlatTableList.tsx` — 169, 174, 206, 209, 218, 221, 227, 228, 237, 240
- `SchemaPane/TreeNode/icons.tsx` — 43, 54, 59, 71, 87, 91, 126
- `SchemaPane/HoverPanel/HoverPanel.tsx` — 142; `HoverPanel/InfoItem.tsx` — 22
- `SharePopoverBody.tsx` — 182, 197, 198, 236, 247
- `Panels/common/EllipsisCell.tsx` — 70 (`bg-gray-900 text-white` tooltip)

**Inline rgb/hex — keep as-is (dynamic, not theme tokens):**
- `ConnectionPane/TreeNode/Label.tsx` 90–91, `TabItem/TabItem.tsx` 72/77/78,
  `ResultPanel/BatchQuerySelect.tsx` 451–455 — these derive from per-environment
  colors and the indigo fallback; out of scope for theming.
- `AccessGrantItem.tsx` 233–234 — animation keyframe; leave unless it clashes on dark.

**Admin / terminal — now themed via the nested scope (not pinned):**
- `TerminalPanel/TerminalPanel.tsx` — outermost `bg-dark-bg` (206) and inner
  `bg-dark-bg` (208) become `bg-background`; wrap the root div in the nested
  `SQLEditorThemeScope` (resolved admin theme). Overlays at 246 (`bg-black/20`), 251
  (`text-gray-400`) route through tokens.
- `CompactSQLEditor.tsx:323` — drop hardcoded `theme: "vs-dark"`; read the scope's
  Monaco theme.
- The result-view `dark:` variants listed above are removed here too — in admin mode
  they now get dark token *values* from the nested scope. The admin terminal is no
  longer a special-cased aesthetic; it is just the resolved admin theme applied to a
  nested scope.

### 6. Switcher UI

Add a preset selector using the shared React `Select` (a dropdown — preferred over a
radio group now that the catalog has six entries) inside
**`QueryContextSettingPopover.tsx`**, as a new `border-t pt-1` section below "Max row
count" — this popover already hosts editor/query preferences (data source, Redis, row
limit). Options are driven off the `PRESETS` array (so adding a theme needs no UI
change). Each label uses the preset's `name` i18n key; add a section label
(`sql-editor.editor-theme`) translated in all five locale files. Built-in palette names
(Solarized, Monokai, Nord) are proper nouns and stay untranslated as values.

> Alternative considered: a dedicated editor-settings menu. Rejected for now — reusing
> the existing settings popover avoids new toolbar surface. Revisit if a richer
> theme/custom-theme UI lands.

## Data flow

```
localStorage ──load──▶ useSQLEditorEditorStore.themeId ──set by──▶ QueryContextSettingPopover (Select)
                              │
                              ▼
              Root SQLEditorThemeScope at SQLEditorLayout (selected preset)
              · inline --color-* on the layout container → CSS cascade re-themes chrome
              · provides theme via context (reaches RequestDrawerHost + portals)
                              │
        ┌──────────────┬──────┴───────────────┬───────────────┐
        ▼ (worksheet)  ▼ (admin tab)           ▼ (overlay portals / request drawer)
                       │                       Nested SQLEditorThemeScope per portal
                       │                       · context read via useSQLEditorTheme()
                       │                       · re-applies --color-* on portaled DOM
                       │                         (CSS vars don't cross portals)
                       │                       · drawer Monaco → setTheme(bb-<selected>)
        ┌──────────────┴───────────────────────────────────┐
        ▼ (worksheet tab)                                  ▼ (admin tab)
 StandardPanel Monaco                          Nested SQLEditorThemeScope
 reads context → setTheme(bb-<selected>)       theme = resolveAdminTheme(selected)
                                               · inline --color-* on TerminalPanel root
                                                 (overrides parent for the subtree)
                                               · CompactSQLEditor + ResultView read
                                                 context → dark tokens + Monaco
                                                 setTheme(bb-<adminTheme>)
```

## Testing

- **Unit:** `validateTheme()` passes for **all six** presets (every required chrome
  token, every `editor` key, every `syntax` key present — no holes);
  `themeToCssVars(theme)` (the helper `SQLEditorThemeScope` uses) returns the correct
  `CSSProperties` for a given theme;
  `buildMonacoTheme` maps `editor`/`syntax` onto Monaco `colors`/`rules` correctly;
  store persists/restores `themeId` and falls back to `light` on invalid input.
- **Component:** rendering `sqleditor--wrapper` with each preset applies the expected
  CSS vars; switching `themeId` updates them. `resolveAdminTheme` returns the selected
  theme when dark and `dark` when light. A nested `SQLEditorThemeScope` overrides the
  parent's CSS vars for its subtree.
- **Manual/visual:** Light preset is visually identical to current `main`; each of the
  other five renders chrome + result grid + Monaco coherently (no un-themed islands,
  syntax colors match the named theme's identity). Admin mode: with a light theme
  selected, the terminal subtree is dark while chrome stays light; with a dark theme
  selected, the terminal matches it. With a non-light theme selected, also confirm the
  **portaled** surfaces theme correctly: the mobile FAB + sidebar overlay, and the
  access-request drawer (`AccessGrantRequestDrawer`) chrome **and** its Monaco editor —
  none should fall back to light. Checked through `pnpm --dir frontend dev`.
- Gates: `pnpm --dir frontend fix`, `check`, `type-check`, `test`.

## Risks

- **Migration breadth (Section 5) is the bulk of the work and the main risk.** Missing
  an offender shows up as an un-themed element on dark. Mitigate by grepping for
  `dark:`, `-gray-`, `-zinc-`, `-slate-`, `bg-white`, `bg-black` under the SQL Editor
  tree after migration and confirming zero remain (excluding the admin terminal and
  the documented dynamic/animation cases).
- **Portals break CSS-variable cascade.** Surfaces portaled to the app-global layer
  roots (mobile FAB / sidebar overlay) and the `RequestDrawerHost` sibling (incl. its
  Monaco) do not inherit the chrome scope's `--color-*` by DOM cascade. Mitigate by
  raising the root scope to `SQLEditorLayout` (context reaches the drawer) and wrapping
  each portal's children in a nested `SQLEditorThemeScope` that re-applies the vars (it
  reads the theme via the context the portal preserves). Verify each portaled surface
  under a dark/named theme; a missed portal renders light-on-dark.
- **Removing the `dark` prop / nested admin scope.** `ResultView` is shared between
  worksheet and terminal; after dropping the prop it must read the theme from context.
  Verify: (a) worksheet result grid follows the selected theme, (b) terminal result grid
  is dark via the nested scope, (c) entering/leaving admin mid-session re-themes both the
  chrome boundary and Monaco, (d) the nested scope's CSS-var override actually wins over
  the root scope for the whole terminal subtree (no leaked light tokens).
- **Monaco theme generation** replaces `callCssVariable` reads; confirm the generated
  `bb`-equivalent matches the current editor look pixel-for-pixel on light.
- **Named-theme authoring fidelity.** Six palettes (incl. four named themes with full
  syntax sets) must be derived from published references and checked against contrast
  needs for both chrome text and result-grid readability. This is real per-theme design
  effort and the second-largest cost after the migration. Author each as its own file
  so they can be reviewed/tuned independently; the `validateTheme()` test guards
  structural completeness but not aesthetic quality — that needs a visual pass.

## Future work (out of scope)

- Custom theme editor (color pickers / JSON), validation, and storage of arbitrary
  palettes. The model already supports it — only UI + persistence of custom objects
  is missing.
- "System" preset following `prefers-color-scheme`.
- Backend/cross-device persistence of the theme choice.
- Extending the same pattern app-wide beyond the SQL Editor.

## Open questions

1. Exact palette values for `dark` and the four named themes — the tables/references
   above are starting points; confirm chrome + syntax values visually during
   implementation.
2. Ordering of presets in the switcher dropdown (proposed: Default Light, Default Dark,
   Solarized Light, Solarized Dark, Monokai, Nord).
3. Admin scope when the selected theme is light: falls back to `dark` (decided). Confirm
   that picking a *dark* theme and entering admin simply keeps that theme (no separate
   "terminal" identity / matrix-green) — assumed yes.
