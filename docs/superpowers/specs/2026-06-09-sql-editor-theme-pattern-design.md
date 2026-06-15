# SQL Editor Theme Pattern — Design

**Date:** 2026-06-09
**Status:** Implemented — the sections below are the original design; the box
immediately below records what actually shipped and where it diverged.
**Scope:** SQL Editor only (UI chrome + Monaco editor surface + result views)

## Implementation notes — as shipped (supersedes design details below)

The data-driven model, the `SQLEditorThemeScope`, the workspace-scoped localStorage
preference, and the `ThemeSelect` toolbar switcher all landed as designed. The
following diverged during implementation (driven by how the codingame VSCode Monaco
runtime and the real DOM behave):

- **Catalog reduced to two themes: `light` + `dark`.** Solarized×2, Monokai, and Nord
  were dropped — two polished themes beat six half-finished ones. `syntax` is therefore
  unused in practice: both presets omit it and inherit `vs`/`vs-dark` verbatim (the
  `SyntaxPalette` machinery stays for the documented custom-theme future).
- **Monaco theming is per-editor + a *change-only* controller, not a mount-time global
  `setTheme`.** Calling `monaco.editor.setTheme` while an editor is still constructing
  races the codingame theme service and throws → the editor falls back to a read-only
  `<pre>`. So: each SQL-Editor editor passes `options.theme` (themes at construction),
  and `useMonacoThemeController` re-applies the global theme **only on a genuine theme
  change** (StrictMode-safe via a prev-value ref), never on mount.
- **`getResolvedTheme` falls back to each theme's *base* (`vs` / `vs-dark`).** Custom
  `bb-*` themes do not register in this runtime (the service silently swallows
  `defineTheme`), so a dark theme used to fall back to the light `vs` — leaving the
  editor light under Dark. `core.ts` records each preset's `monacoBase` and falls back
  to it.
- **Chrome theming was a *transparent-container* problem, not hardcoded colors.** The
  big panels already use semantic tokens; they're just transparent / un-classed, so on
  the white default they looked fine and under Dark showed white. Fixes: `bg-background`
  + `text-main` on the `sqleditor--wrapper` (themed backdrop + default text), and the
  global default border now uses `rgb(var(--color-block-border))` (identical to the old
  gray-200 in light, themed inside a scope). The active-statement decoration was
  tokenized (`bg-control-bg-hover`). Only a handful of genuinely-hardcoded colors needed
  migrating.
- **Portaled overlays are themed at the overlay *layer root*, not per component.**
  Dialogs (Save Sheet, …), the access drawer, and Base-UI Select/Popover/Tooltip popups
  all portal to the app-global overlay root, outside the scope's DOM — CSS vars can't
  cascade. `useSQLEditorOverlayTheme` sets the theme's vars + a default text color on
  `getLayerRoot("overlay")` while the SQL Editor is mounted (reverted on unmount), which
  themes **all** of them from one place. (In light the vars equal `:root`, so it's a
  no-op outside a dark theme.)
- **Admin (terminal) nested scope, drawer chrome, and the `dark`-prop removal** landed as
  designed.
- **`isDark` is fully deprecated — no light/dark branching in components.** The original
  design exposed `SQLEditorTheme.isDark` "for the few spots that need a boolean," but in
  practice components used `isDark ? colorA : colorB`, which bakes colors into the
  component and can't extend to a 3rd/custom theme. So: the property is **removed** from
  `SQLEditorTheme`; all `isDark ? colorA : colorB` became plain semantic-token classes
  (each theme defines the value); `resolveAdminTheme` keys off `monacoBase === "vs-dark"`;
  and the one genuine terminal-vs-worksheet *layout* difference uses an explicit `compact`
  prop (`TerminalPanel` → `ResultView` → `SingleResultView`), not the theme. Net: colors
  are 100% token-driven and a custom theme is just another token map.

Net new files vs. the design: `theme/useSQLEditorOverlayTheme.ts`. The `getResolvedTheme`
base-fallback is unit-tested in `monaco/core.test.ts`.

## Goal

Extract the colors the SQL Editor uses today into a **data-driven theme model** and
add a **runtime preset switcher** scoped to the SQL Editor. The current look becomes
the `light` preset (zero visual change as the default); a `dark` preset plus several
named pre-defined themes ship alongside it. The model is shaped so that user-defined
**custom themes** can be added later with no re-architecture — a custom theme is just
another theme object.

> **Shipped reality:** the catalog was reduced to **two** themes (`light` + `dark`)
> before merge, and the `isDark` property was removed entirely — see
> [Implementation notes — as shipped](#implementation-notes--as-shipped) above, which
> is authoritative wherever it conflicts with the original design below.

This phase ships a **catalog of pre-defined preset themes** (below), each with its
authentic Monaco syntax palette. Custom-theme editing UI, a "System" option, and
backend persistence are explicitly out of scope (see [Future work](#future-work)).

### Theme catalog

| id | name | monacoBase | notes |
| --- | --- | --- | --- |
| `light` | Default Light | `vs` | today's `:root`/`bb` look, verbatim — the default |
| `dark` | Default Dark | `vs-dark` | Bytebase dark, inherits `vs-dark` chrome + tokens |

## Locked decisions

| Decision | Choice |
| --- | --- |
| End goal | Build the foundation for runtime-switchable themes |
| Scope | SQL Editor only — chrome + Monaco + result views |
| Granularity | Preset themes only — `light` + `dark` shipped (catalog above) |
| Syntax fidelity | Each named theme ships its full, authentic Monaco syntax palette |
| Token scoping | Scoped override — re-declare `--color-*` on the SQL Editor container |
| Persistence | localStorage, client-only, workspace-scoped |
| Theme representation | Approach A — theme is a plain JS token map; both layers derive from it |
| Admin (terminal) mode | A **nested theme scope**; effective theme = `selected.monacoBase === "vs-dark" ? selected : dark` |

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
  id: string;            // "light" | "dark" | … (stable, persisted)
  name: string;          // i18n key for the display label
  // NOTE: `isDark` was REMOVED before merge (see Implementation notes). Components
  // never branch on a boolean; light/dark differences are encoded in `tokens`, and
  // the one place that needs the Monaco family reads `monacoBase` directly.
  monacoBase: "vs" | "vs-dark";
  tokens: Record<SQLEditorThemeToken, RGB>;   // chrome layer
  editor: Partial<EditorChromeColors>;        // Monaco surface (bg, gutter, selection, cursor, lineHighlight, …)
  syntax?: SyntaxPalette;                      // Monaco token colors; omitted → inherit monacoBase verbatim
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

**Supporting types** (the keys each layer must fill — `validateTheme` enforces them):

```ts
// chrome layer — the SQL-Editor subset of existing --color-* names
type SQLEditorThemeToken =
  | "--color-control" | "--color-control-hover"
  | "--color-control-light" | "--color-control-light-hover"
  | "--color-control-bg" | "--color-control-bg-hover"
  | "--color-control-placeholder" | "--color-control-border"
  | "--color-accent" | "--color-accent-hover" | "--color-accent-disabled" | "--color-accent-text"
  | "--color-main" | "--color-main-hover" | "--color-main-text"
  | "--color-background" | "--color-block-border" | "--color-link-hover"
  | "--color-info" | "--color-info-hover" | "--color-warning" | "--color-warning-hover"
  | "--color-error" | "--color-error-hover" | "--color-success" | "--color-success-hover"
  | "--color-matrix-green" | "--color-matrix-green-hover" | "--color-dark-bg";

// Monaco surface (canvas — not reachable by CSS vars)
interface EditorChromeColors {
  background: Hex; selectionBackground: Hex; cursor: Hex; lineHighlight: Hex;
  gutterBackground: Hex; lineNumber: Hex; activeLineNumber: Hex;
}

// Monaco token colors, normalized & language-agnostic
interface SyntaxPalette {
  comment: Hex; keyword: Hex; string: Hex; number: Hex; type: Hex;
  function: Hex; variable: Hex; operator: Hex; delimiter: Hex; predefined: Hex;
}
```

**Derivation functions** — the single `SQLEditorTheme` object is the source; both layers
are *computed* from it (this is the whole point of Approach A: presets and future custom
themes flow through the exact same functions):

```ts
// theme/derive.ts

// CHROME: tokens → inline CSS custom properties, spread onto a container's `style` prop.
//   { "--color-control": "82 82 91", … }  →  <div style={vars}>
function themeToCssVars(tokens: SQLEditorTheme["tokens"]): CSSProperties;

// MONACO: whole theme → registered standalone theme data.
//   maps `editor` → IStandaloneThemeData.colors, `syntax` → .rules (incl. SQL scopes).
function buildMonacoTheme(theme: SQLEditorTheme): monaco.editor.IStandaloneThemeData;

// MONACO name used by defineTheme(name, …) and the global setTheme controller.
function monacoThemeName(theme: SQLEditorTheme): string;          // `bb-${theme.id}`

// ADMIN: foreground-panel resolution for the nested terminal scope + Monaco controller.
function resolveAdminTheme(selected: SQLEditorTheme): SQLEditorTheme; // selected.monacoBase === "vs-dark" ? selected : presets.dark

// VALIDATION: throws if any chrome token / editor key / syntax key is missing.
//   Unit-tested across every entry in PRESETS so no theme can ship with holes.
function validateTheme(theme: SQLEditorTheme): void;
```

`<SQLEditorThemeScope>` (§4) is the only consumer of `themeToCssVars` (for the chrome) and
context; `monaco/core.ts` calls `buildMonacoTheme` once per preset at init, and the single
Monaco controller (§4 → Monaco) is the only caller of `setTheme(monacoThemeName(active))`.

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
1. provides the theme via context (so descendants can read `theme.monacoBase` / the
   Monaco theme name), and
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
**chrome** theme on its own container: wrap the portal's children in a nested
`SQLEditorThemeScope` whose theme comes from `useSQLEditorTheme()` (read through the
context that the portal preserves) so it re-writes the `--color-*` vars onto the portaled
DOM. For the request drawer this re-themes its `Sheet` chrome (form controls, labels).
Its embedded **Monaco**, however, cannot have its own theme — Monaco's theme is global
(see **Monaco** below); the drawer's editor shares the one active Monaco theme. No new
global *chrome* theming — only the specific SQL-Editor portal subtrees re-apply vars.

**Portal coverage for v1 (scope decision).** The portal inventory is larger than the
FAB/sidebar + drawer: other SQL-Editor surfaces also mount under the app-global overlay
root — the custom hover panels (`DatabaseHoverPanel`, `SchemaPane/HoverPanel`), the
`RequestRoleSheet` (same `RequestDrawerHost` path), and **Base-UI-internal** popups
(`ConnectChooser`'s `Select` dropdown, `EllipsisCell`'s tooltip). For the first ship we
wrap **our own** `createPortal`/`Sheet` surfaces (overlays, drawer, RequestRoleSheet,
hover panels) in a nested scope. The Base-UI-internal popups portal through shared `ui/*`
primitives and can't be re-scoped per-SQL-Editor without touching app-wide components, so
they are a **tracked follow-up** — under a dark/named theme they may briefly show the
default (light) tokens. Visual tests (§ Testing) enumerate covered vs deferred surfaces.
(If app-scope chrome lands later, the whole portal problem disappears: tokens applied at
app root cascade into the shared overlay root automatically.)

**Nested admin scope.** Wrap `<TerminalPanel>`'s root div (`TerminalPanel.tsx:206`) in a
nested `SQLEditorThemeScope` whose theme is the **resolved admin theme**:

```ts
resolveAdminTheme(selected) = selected.monacoBase === "vs-dark" ? selected : presets.dark;
```

Its inline CSS-var overrides win over the parent scope for the terminal subtree (CSS
cascade), so the terminal + its `ResultView`s render dark (or in the selected dark
theme), while the surrounding chrome keeps the selected theme. Worksheet mode renders
under the root scope only — no nesting.

This is the **replacement for the `dark` boolean prop**: `ResultView` reads nothing —
its styling comes entirely from the scope's tokens rather than `dark:` variants or a
boolean. (As shipped, `ResultView` keeps only a `compact` prop, which is a terminal-vs-
worksheet *layout* difference, not a theme one.) The `dark` prop and the
`dark && "dark bg-dark-bg"` class at `ResultView.tsx:237` are removed; `TerminalPanel`'s
hardcoded `bg-dark-bg` becomes the token-driven `bg-background` (the nested dark theme
sets `--color-background` to the dark value).

**Monaco.** In `monaco/core.ts`, register one Monaco theme **per preset** via
`buildMonacoTheme(preset)`, which maps the preset's `editor` colors → Monaco `colors{}`
and its `syntax` palette → Monaco token `rules[]` (including SQL scopes). This replaces
the current `callCssVariable()`/`:root` reads — required because Monaco is
canvas-rendered and cannot see container-scoped CSS overrides, and because named themes
carry their own syntax colors rather than tinting `vs`/`vs-dark`. All six themes are
defined at init (`monaco.editor.defineTheme(\`bb-${id}\`, …)`).

**The Monaco theme is a single global value — it cannot be per-scope.**
`monaco.editor.setTheme` applies to **all** standalone editors at once; Monaco has no
per-instance theme. So unlike the chrome (which is genuinely per-scope via CSS cascade),
every visible Monaco editor shares **one** theme. Concretely, more than one Monaco can be
on screen simultaneously: the access-request drawer hosts its own editor
(`AccessGrantRequestDrawer.tsx:262`) and can be open **over** an admin terminal — so a
"one code panel at a time" assumption does **not** hold.

The SQL Editor therefore maintains **one active Monaco theme**, derived from the
**foreground code surface**: worksheet → the **selected** theme; admin →
`resolveAdminTheme(selected)` (`useActiveSQLEditorTheme()`). It is applied two ways that
agree on the value:

- The SQL-Editor editors (`StandardPanel/SQLEditor`, `TerminalPanel/CompactSQLEditor`,
  and the drawer's editor) pass an explicit `theme` option = `monacoThemeName(active)`,
  so the shared `MonacoEditor` sets the right theme on construction (no `bb-light`
  flash). `CompactSQLEditor` drops its hardcoded `theme: "vs-dark"` for this.
- A single controller hook `useMonacoThemeController()` (mounted once at the SQL-Editor
  root) re-applies the active theme on `themeId`/`mode` change via
  `monaco.editor.setTheme(getResolvedTheme(monacoThemeName(active)))` — covering live
  switches while editors stay mounted.

`getResolvedTheme` is **mandatory** here: in runtime modes where the vscode theme-service
override prevents a custom theme from registering (`initializeTheme` only adds a name to
`registeredThemes` after `defineTheme` succeeds), calling `setTheme` with an unregistered
name is a silent no-op; `getResolvedTheme` falls back to the always-available
`bb-light`/`vs`, so a named theme can never leave Monaco stuck on a stale theme.

**Boundary & non-SQL Monaco (scope decision).** `setTheme` is global, but the shared
`MonacoEditor` is **left unchanged**: editors that pass no explicit theme — every non-SQL
Monaco surface in the app — keep resetting to the default `bb-light` on construction. So
while the SQL Editor is mounted the controller owns the global Monaco theme; elsewhere
each editor resets itself. **Containment guarantee:** `useMonacoThemeController` also
resets the global Monaco theme back to `bb-light` on **unmount** (a separate `[]`-effect
cleanup), so the SQL Editor's selected theme never lingers on app-scope Monaco after the
user leaves — covering even an already-mounted/kept-alive non-SQL editor that wouldn't
re-run its own construction reset. Because the SQL Editor is a full-page view, non-SQL
Monaco is never on screen alongside it, so this split is invisible in normal navigation —
no app-wide Monaco change is needed now. (Monaco's hard limit: two editors visible at the
same instant cannot have different themes — there is one global theme. This never occurs
here because the SQL Editor doesn't share the screen with another Monaco surface.) App-level Monaco (all editors follow the preference)
was considered and **rejected for v1**: it would require coupling the shared `MonacoEditor`
to the theme preference + an app-root controller + an app-level store — broad blast radius
for a benefit that is only theoretical today. See **Extensibility to app scope** (Future
work) for the small lift path.

When the access drawer is open, its Monaco **adopts the active theme** (passes the same
active Monaco name), so it never competes with the terminal for the global theme.
Consequence: opening the drawer over a light-selected admin tab shows a dark statement
editor (matching the visible terminal) — acceptable, since per-editor Monaco themes are
impossible.

### 5. Color migration (prerequisite)

Scoped overrides only reach components that use the tokens. The following bypass them
and must be migrated. Two rules:

1. Replace raw palette classes / inline colors with the semantic token equivalent.
2. **Delete `dark:`-prefixed variants** — dark theming now flows through token
   *values* supplied by the active (root or nested) theme scope, not a global `.dark`
   class. The `dark` boolean prop threaded through `ResultView` is removed; the rare
   consumer that needs the Monaco family reads `useSQLEditorTheme().monacoBase`, but no
   component branches colors on a boolean. In admin mode the result grid renders dark because it
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
- `CompactSQLEditor.tsx:323` — drop hardcoded `theme: "vs-dark"`; let the global Monaco
  controller (§4 → Monaco) drive the theme.
- The result-view `dark:` variants listed above are removed here too — in admin mode
  they now get dark token *values* from the nested scope. The admin terminal is no
  longer a special-cased aesthetic; it is just the resolved admin theme applied to a
  nested scope.

### 6. Switcher UI

Add a preset selector using the shared React `Select` (a dropdown — preferred over a
radio group now that the catalog has six entries). It is its own small component
`ThemeSelect` placed in **`EditorAction.tsx`**'s `action-right` group (next to
`ChooserGroup` / `OpenAIButton`).

> **Why not `QueryContextSettingPopover` (the original draft target):** that popover
> early-returns `null` in admin mode (`currentTabMode !== "ADMIN"`), so a switcher there
> would disappear in exactly the mode we are theming. `EditorAction` is rendered by
> **both** `StandardPanel` and `TerminalPanel`, and its `action-right` group is always
> visible — so the theme control is reachable in worksheet and admin alike.

Options are driven off the `PRESETS` array (so adding a theme needs no UI change). Each
label uses the preset's `name` i18n key; built-in palette names (Solarized, Monokai,
Nord) are proper nouns and stay untranslated as values. The current value and `onChange`
bind to `useSQLEditorEditorState((s) => s.themeId)` / `setThemeId`.

## Data flow

CHROME (per-scope, CSS cascade):
```
localStorage ──load──▶ useSQLEditorEditorStore.themeId ──set by──▶ QueryContextSettingPopover (Select)
                              │
                              ▼
              Root SQLEditorThemeScope at SQLEditorLayout (selected preset)
              · inline --color-* on the layout container → CSS cascade re-themes chrome
              · provides theme via context (reaches RequestDrawerHost + portals)
                              │
   ┌──────────────────┬───────┴───────────────────┬───────────────────────────┐
   ▼ (worksheet tab)  ▼ (admin tab)                ▼ (overlay portals + request drawer)
 root scope only      Nested SQLEditorThemeScope   Nested SQLEditorThemeScope per portal
 (selected tokens)    theme=resolveAdminTheme(sel) · context read via useSQLEditorTheme()
                      · --color-* on TerminalPanel   (portals preserve context, not CSS)
                        root (overrides parent)     · re-applies --color-* on portaled DOM
                      · ResultView is token-driven    (drawer Sheet chrome, FAB, sidebar)
                        (compact prop = layout only)
```

MONACO (single global, driven by foreground panel — cannot be per-scope):
```
{ tab mode, selected themeId } ──▶ active = mode==="ADMIN" ? resolveAdminTheme(sel) : sel
                                          │
                                          ▼
                         monaco.editor.setTheme(`bb-${active.id}`)   // one global theme
                                          │
                ┌─────────────────────────┼─────────────────────────┐
                ▼                         ▼                          ▼
        worksheet/terminal editor   drawer's Monaco (if open)   any other Monaco
        ── all share the one active theme ──
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
  access-request drawer (`AccessGrantRequestDrawer`) **chrome** — none should fall back to
  light. Drawer-Monaco conflict: open the drawer over an admin terminal and confirm
  neither editor flickers/ends on the wrong theme — both share the one active global
  Monaco theme (the documented trade-off), not a fight that leaves one wrong.
  Checked through `pnpm --dir frontend dev`.
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
- **Monaco's theme is global, not per-instance.** Multiple editors can be visible at once
  (terminal + access-drawer Monaco), but `setTheme` repaints all of them. The design
  resolves this by driving one global theme from the foreground panel and having the
  drawer adopt it (§4 → Monaco), rather than per-editor themes (impossible in Monaco).
  Risk if implemented naively as "each editor sets its own theme": last-writer-wins
  flicker / wrong theme. The single-controller approach must own every `setTheme` call;
  no component should call `setTheme` independently.
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
- The deferred Base-UI-internal portal popups (ConnectChooser dropdown, EllipsisCell
  tooltip) — re-scope their content under a dark/named theme.

### Extensibility to app scope

The architecture is deliberately **liftable to app-wide** with a small, well-understood
change — not a rewrite — should that ever be wanted (a maintainer raised it; deferred as
out of scope for v1 to keep complexity/time down):

- **Chrome.** `SQLEditorThemeScope` and the token model already override the *global*
  `--color-*` names. Lifting = mount the scope at the app root instead of the SQL-Editor
  container. As a bonus, the portal problem vanishes (tokens at app root cascade into the
  shared overlay root).
- **Monaco.** Move `useMonacoThemeController()` from the SQL-Editor root to the app root,
  and change the shared `MonacoEditor`'s construction-time reset to use the active
  preference instead of the hardcoded `bb-light` default — then every Monaco surface
  follows the preference.
- **Preference.** Move `themeId` from the SQL-Editor store to an app-level (workspace/
  user-scoped) store, and relocate the switcher from the editor toolbar to a global
  appearance/settings surface.

Each step is independent and the token/preset/scope code is unchanged.

## Open questions

1. Exact palette values for `dark` and the four named themes — the tables/references
   above are starting points; confirm chrome + syntax values visually during
   implementation.
2. Ordering of presets in the switcher dropdown (proposed: Default Light, Default Dark,
   Solarized Light, Solarized Dark, Monokai, Nord).
3. Admin scope when the selected theme is light: falls back to `dark` (decided). Confirm
   that picking a *dark* theme and entering admin simply keeps that theme (no separate
   "terminal" identity / matrix-green) — assumed yes.
