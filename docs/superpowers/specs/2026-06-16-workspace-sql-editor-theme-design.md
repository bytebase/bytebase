# Workspace-Enforced SQL Editor Theme — Design

**Date:** 2026-06-16
**Status:** Draft (design)
**Scope:** SQL Editor only (phase 1). App-wide branding is explicit future work.
**Builds on:** [2026-06-09 SQL Editor Theme Pattern](./2026-06-09-sql-editor-theme-pattern-design.md)
(the data-driven token model, `SQLEditorThemeScope`, presets, and `derive` helpers — all reused).

## Goal

Turn the dev-only, per-browser SQL Editor theme into a **workspace-enforced** theme for
brand governance: a workspace **admin** picks a built-in preset or authors a **custom
theme**, it is stored at the **workspace** level in the backend, and **every user** in the
workspace sees it applied to the SQL Editor. No per-user choice (pure enforcement).

## Locked decisions

| Decision | Choice | Rationale |
| --- | --- | --- |
| Ownership | **Workspace-enforced**, admin-authored | Brand governance — one palette for everyone (the requested model). Differs from Linear/Slack/Notion, which are per-user. |
| Application scope | **SQL Editor only** (phase 1) | Reuses the entire existing token infra. App-wide branding is a bigger lift (full app `--color-*` set + `:root` override + app-wide color audit) → [Future work](#future-work). |
| Custom-theme editor | **Anchor colors → derive** | Linear-style: admin sets ~5 anchors, we generate the ~28 tokens. Hard to produce a broken/ugly theme; live preview. |
| Storage | New field on **`WorkspaceProfileSetting`** | Existing workspace-admin settings blob (same home as branding logo); admin-gated; already fetched app-wide. No new table, no SQL migration. |
| Persistence removed | Per-user `localStorage` theme | Replaced by the workspace value (kept only as an offline read cache, optional). |

## Non-goals (phase 1)

- App-wide (non-SQL-Editor) theming.
- Per-user override of the workspace theme.
- Import/export of theme strings (a follow-up; the data model leaves room for it).
- Un-gating multiple end-user-selectable themes (there is exactly one workspace theme).

---

## 1. Backend

### 1.1 Proto

Add to the existing workspace-admin settings message. **Bytebase has two proto layers for
this setting** — the public v1 API (`proto/v1/v1/setting_service.proto`, use fields
**23/24**; 16 is `reserved`) **and** the persisted store (`proto/store/store/setting.proto`,
`message WorkspaceProfileSetting` — use ITS next free numbers). The **same** fields +
`SQLEditorThemeSetting` message must be added to **both** (and the v1↔store converters
updated — see §1.2), or values are dropped at the store boundary. The enforced **selection**
(`id`) is separated from the custom theme **definition**, so the stored definition is always
a complete custom theme (never an empty-token placeholder) and built-ins stay frontend-only
references:

```proto
message WorkspaceProfileSetting {
  // ... existing fields 1..22 ...

  // The enforced SQL Editor theme id: an OPAQUE string — a built-in preset id
  // resolved entirely by the frontend (the backend neither defines nor ships the
  // preset catalog), OR a custom theme's stable, randomly-generated uuid.
  // Empty ⇒ no enforced theme → default light.
  string sql_editor_theme_id = 23;

  // The enforced CUSTOM theme's full definition — present ONLY when
  // `sql_editor_theme_id` is a custom uuid. The store never holds a built-in here
  // (built-ins live only in the frontend), so this is always a complete custom
  // theme. Built-in tokens are never stored, so preset improvements propagate to
  // every workspace with no data migration.
  SQLEditorThemeSetting sql_editor_custom_theme = 24;
}

// The uniform theme shape. The frontend's built-in PRESETS use it too (with
// tokens), so the catalog is homogeneous (§2.1). In the backend store it ONLY
// ever describes a custom theme → `tokens` is always fully populated.
message SQLEditorThemeSetting {
  // The custom theme's stable uuid (equals `sql_editor_theme_id` when enforced).
  string id = 1;
  // Admin-given display name (required).
  string name = 2;
  // Monaco base: "vs" (light) | "vs-dark" (dark). Derived from background luminance.
  string monaco_base = 3;
  // The ~28 chrome tokens (`--color-*` → "r g b"). REQUIRED and complete.
  map<string, string> tokens = 4;
}
```

Regenerate: `buf format -w proto && buf lint proto && (cd proto && buf generate)`.

### 1.2 Store

`WorkspaceProfileSetting` is persisted as one **protojson blob** in the `setting` table
(`name = WORKSPACE_PROFILE`). Adding a proto field is backward-compatible:

- **No SQL migration** — it's a JSONB value, not a column. `LATEST.sql` unchanged.
- **v1↔store converters** (`backend/api/v1/setting_service_converter.go`):
  `convertWorkspaceProfileSetting` (v1→store) and `convertToWorkspaceProfileSetting`
  (store→v1) must map both new fields (+ a `SQLEditorThemeSetting` helper). The value is
  persisted as the **store** proto, so without this it's silently dropped on write and never
  returned on read.
- The update path is **field-mask based** with an explicit `switch` over
  `value.workspace_profile.*` paths in `setting_service.go` (each case copies one field into
  the stored payload; an unlisted path hits `default` → `InvalidArgument`). Add cases for
  `value.workspace_profile.sql_editor_theme_id` and
  `value.workspace_profile.sql_editor_custom_theme` (the latter validated, §1.3). A nil
  custom theme is valid and clears the field (built-in selection).

### 1.3 Validation (server-side)

On `UpdateSetting(WORKSPACE_PROFILE)` with `sql_editor_theme_id` / `sql_editor_custom_theme`
in the mask:

- **`sql_editor_theme_id`** — an opaque string; accept any value (empty ⇒ default light).
  The server is **catalog-agnostic**: built-ins are a frontend concept, so there's no preset
  list mirrored/kept-in-sync in Go; the client resolves the id, falling back to `light` for
  an unknown/empty id (existing `resolveThemeId`).
- **`sql_editor_custom_theme`** (when present) — this is the **only** thing validated hard,
  because it's the only server-stored pixels. There is **no empty-token case**: `tokens` is
  **required and complete** — it must contain **all** `SQL_EDITOR_THEME_TOKENS` keys, each a
  valid `"r g b"` triple (0–255). Also `id` non-empty, `name` non-empty,
  `monaco_base ∈ {"vs","vs-dark"}`. (Reuse the frontend's token list — mirror it in a small
  Go slice or validate structurally.)

Reject with `InvalidArgument` otherwise — the server must not store a malformed custom theme
that would break every user's editor.

### 1.4 Permission

Writes are already gated by **`bb.settings.setWorkspaceProfile`** (the same permission
`SQLEditorSection` uses for `sqlResultSize` / `queryTimeout`). No new permission.

---

## 2. Frontend

### 2.1 Theme source: workspace setting, not localStorage

Today `SQLEditorThemeRoot` reads `themeId` from `useSQLEditorEditorState` (localStorage,
`stores/sqlEditor/editor.ts`). Replace the **source**, keep everything downstream:

- New selector/helper `useWorkspaceSQLEditorTheme(): SQLEditorTheme`. Reads
  `sqlEditorThemeId` + `sqlEditorCustomTheme` from `getWorkspaceProfile()`:
  - `sqlEditorCustomTheme` present and its `id === sqlEditorThemeId` → build a
    `SQLEditorTheme` from it (validate via existing `validateTheme`; on failure → `light`).
    *[custom]*
  - else `sqlEditorThemeId` non-empty → `resolveThemeId(sqlEditorThemeId)` (catalog; `light`
    fallback). *[built-in]*
  - `sqlEditorThemeId` empty → `light`.
- `SQLEditorThemeRoot` (`SQLEditorLayout.tsx`) uses this instead of the store `themeId`.
  `useActiveSQLEditorTheme` keeps resolving admin mode the same way.
- **Reactivity:** `getWorkspaceProfile()` is reactive via the app store, so an admin
  changing the theme propagates to all open clients on their next settings refresh — no
  bespoke polling.

**Built-in themes live only in the frontend — in the `SQLEditorThemeSetting` shape.** The
built-in presets (`PRESETS`: `light` / `dark` / `solarized-dark`) stay **hardcoded
client-side** as full `{ id, name, monacoBase, tokens }` objects — the *same* shape as a
stored custom theme. The backend never ships a built-in. So the frontend's **selectable
theme list** is composed at runtime, custom-first then built-ins appended:

```
allThemes = [ custom theme(s) read from `sqlEditorCustomTheme` in the profile setting ]
            ++ [ hardcoded built-in PRESETS ]
```

Because both sides share the `SQLEditorThemeSetting` shape (full tokens), the list is
**homogeneous** — the admin picker and the editor operate over one uniform type. In phase 1
the profile holds a single `sqlEditorCustomTheme`, so "custom theme(s) from settings" is
**0-or-1**; this generalizes cleanly to a workspace **catalog of multiple custom themes**
later (see [Future work](#future-work)). `sqlEditorThemeId` selects which entry is enforced.

### 2.2 Remove the per-user path

- Delete the `themeId` field, `setThemeId`, `readThemeId`, and `themeKey` from
  `stores/sqlEditor/editor.ts`; drop `storageKeySqlEditorTheme` (`utils/storage-keys.ts`)
  and the store persistence test.
- Remove the dev-only `<ThemeSelect>` from `EditorAction.tsx` (and `ThemeSelect.tsx` +
  test). End users no longer choose a theme — they get the workspace theme.
  - *(Optional, deferred)* a localStorage cache of the resolved workspace theme to avoid a
    first-paint flash before settings load. Not required for phase 1.

### 2.3 Admin editor — a subsection of `SQLEditorSection`

Add an **"Appearance / Theme"** block to
`react/pages/settings/general/SQLEditorSection.tsx` (it already owns workspace-level SQL
Editor settings, the `SectionHandle` dirty/revert/update contract, and the
`bb.settings.setWorkspaceProfile` gate). The block:

1. **Preset picker** — radio/segmented over the built-in catalog (`PRESETS`) + a "Custom"
   option. Selecting a preset saves `sqlEditorThemeId = <preset id>` and **clears**
   `sqlEditorCustomTheme` (built-in reference, nothing stored beyond the id).
2. **Custom editor (when "Custom" selected)** — anchor color inputs (§2.4) + a **live
   preview** (§2.5). On first switch to Custom, generate a stable id once with `uuid`'s `v4`
   (`import { v4 as uuidv4 } from "uuid"`; **not** `crypto.randomUUID()`, which the
   `pnpm check` gate bans) and keep it across edits/saves; seed anchors from the currently
   selected theme via `themeToAnchors`. Saves `sqlEditorThemeId = <uuid>` **and**
   `sqlEditorCustomTheme = { id:<uuid>, name, monacoBase, tokens }` (always full tokens).
   The editor UI reflects state from the stored value: a `sqlEditorCustomTheme` whose id
   matches ⇒ "Custom" selected; else the preset matching `sqlEditorThemeId`.
3. Gated read/write exactly like the existing fields:
   - read `useAppStore((s) => s.getWorkspaceProfile())` → `sqlEditorThemeId` + `sqlEditorCustomTheme`
   - write `updateWorkspaceProfile({ payload: { sqlEditorThemeId, sqlEditorCustomTheme }, updateMask: ["value.workspace_profile.sql_editor_theme_id", "value.workspace_profile.sql_editor_custom_theme"] })` (the method takes `payload`; mask paths use the full `value.workspace_profile.*` form — matching the existing `query_timeout` / `sql_result_size` saves)
   - `PermissionGuard permissions={["bb.settings.setWorkspaceProfile"]}`; non-admins see
     it read-only.
   - participate in the section's `isDirty` / `revert` / `update` (`useImperativeHandle`).

### 2.4 Anchor-derive (feature 1 core)

New pure helper in `theme/derive.ts`:

```ts
// The ~5 colors the admin picks (hex). Everything else is generated.
export interface ThemeAnchors {
  background: Hex; // page / editor background  → --color-background, --color-dark-bg
  surface: Hex;    // elevated controls/headers → --color-control-bg (+ hover)
  text: Hex;       // primary text              → --color-main, --color-control (+ light/hover)
  accent: Hex;     // brand / primary action    → --color-accent (+ hover/disabled/text)
  border: Hex;     // dividers / outlines        → --color-block-border, --color-control-border, --color-link-hover
}

// Generate the full SQLEditorTheme.tokens from anchors. Neutral + accent ramps
// are derived by lightening/darkening toward the background (dark themes lighten,
// light themes darken); status colors (info/warning/error/success) and
// matrix-green stay FIXED semantic values — brand palettes shouldn't recolor
// "error is red". monacoBase = background luminance < 0.5 ? "vs-dark" : "vs".
export function deriveThemeFromAnchors(
  anchors: ThemeAnchors,
  name: string
): SQLEditorTheme;
```

Derivation rules (deterministic, unit-tested):
- `--color-*-hover` / `--color-*-light` / `--color-*-light-hover`: shift the base toward or
  away from `background` by fixed deltas (lighten on dark themes, darken on light).
- `--color-accent-disabled`: mix accent toward background ~50%; `--color-accent-text`:
  black/white by accent luminance.
- `--color-control-placeholder`: text mixed toward background ~45%.
- `--color-*-bg-hover`: surface shifted one step.
- **Fixed** (not derived): `--color-info/-warning/-error/-success` (+hover),
  `--color-matrix-green` (+hover). These keep their semantic meaning across brands.
- Inverse helper `themeToAnchors(theme)` lets the editor seed anchors from a built-in
  preset ("start from Dark, then tweak").

The output goes through the existing `validateTheme` before save/preview.

### 2.5 Live preview

A small `<ThemePreview theme={draft} />` rendered inside a `SQLEditorThemeScope theme={draft}`
showing a representative slice — a toolbar button, an input, a result-grid header row, a
code line, an accent action — so the admin sees the derived palette before saving. No real
Monaco needed (a styled snippet is enough; the editor canvas is `vs`/`vs-dark` base anyway).

---

## 3. Data flow

```
Admin (Settings → SQL Editor → Theme)
  pick preset ──────────────► themeId = <preset>,  customTheme = (cleared)                    (built-in reference)
  OR edit anchors ──► deriveThemeFromAnchors ──► validateTheme ──► themeId = <uuid>,  customTheme = { id:<uuid>, name, monacoBase, tokens }   (custom; tokens required)
        │
        ▼  updateWorkspaceProfile({ payload:{ sqlEditorThemeId, sqlEditorCustomTheme }, updateMask:["value.workspace_profile.sql_editor_theme_id","value.workspace_profile.sql_editor_custom_theme"] })
   UpdateSetting(WORKSPACE_PROFILE)  ── server validates custom_theme (tokens required) ──► setting blob (protojson)
        │
        ▼  (app store refresh)
Every user: getWorkspaceProfile() → { sqlEditorThemeId, sqlEditorCustomTheme }
   ──► useWorkspaceSQLEditorTheme() ──► SQLEditorThemeRoot ──► (existing scope/Monaco/overlay pipeline unchanged)
```

## 4. Migration / rollout

- **No SQL migration.** Adding proto fields to the `WORKSPACE_PROFILE` JSON blob is
  backward-compatible; empty `sql_editor_theme_id` ⇒ built-in `light` (today's look).
- Drop the per-user localStorage theme + dev-only switcher (§2.2). No data to migrate —
  it was dev-only and client-local.
- Default is **empty `sql_editor_theme_id`** ⇒ frontend `light` ⇒ zero visual change for any
  workspace that doesn't opt in.

## 5. Testing

- **Backend:** `setting_service` test — `sql_editor_theme_id` accepts any value
  (catalog-agnostic); `sql_editor_custom_theme` accepts a complete theme and **rejects empty
  or partial `tokens`** (the required-tokens rule), plus bad triple, empty id/name, bad
  `monaco_base`; and a partial update mask doesn't clobber other workspace-profile fields.
- **Frontend (unit):**
  - `deriveThemeFromAnchors` — golden token maps for a known anchor set (light + dark);
    every `SQL_EDITOR_THEME_TOKENS` key present; `validateTheme` passes; `monacoBase`
    follows background luminance; status colors are the fixed values.
  - `themeToAnchors ∘ deriveThemeFromAnchors` round-trips the anchors.
  - `useWorkspaceSQLEditorTheme` — `customTheme` matching `themeId` → custom; `themeId` =
    known preset (no/!matching customTheme) → catalog; unknown/empty `themeId` → `light`;
    malformed `customTheme` → `light`.
- **Frontend (component):** `SQLEditorSection` theme block — preset switch + custom edit
  set `isDirty`, `revert` restores, `update` calls `updateWorkspaceProfile` with the right
  mask; read-only without `bb.settings.setWorkspaceProfile`.

## 6. Files touched

| Area | File | Change |
| --- | --- | --- |
| proto (v1) | `proto/v1/v1/setting_service.proto` | `+SQLEditorThemeSetting`, `+ sql_editor_theme_id (23)`, `+ sql_editor_custom_theme (24)` |
| proto (store) | `proto/store/store/setting.proto` | same additions (its own field numbers) — persisted shape |
| backend (convert) | `backend/api/v1/setting_service_converter.go` | map both fields in `convert{,To}WorkspaceProfileSetting` |
| backend | `backend/api/v1/setting_service.go` | pass-through + validation (custom-theme tokens required) |
| backend (test) | `backend/.../setting_service_test.go` (or `backend/tests`) | update/validate/mask tests |
| fe theme | `react/components/sql-editor/theme/derive.ts` | `+deriveThemeFromAnchors`, `+themeToAnchors`, `ThemeAnchors` |
| fe theme | `react/components/sql-editor/theme/useWorkspaceSQLEditorTheme.ts` | new — read workspace setting |
| fe layout | `react/components/sql-editor/SQLEditorLayout.tsx` | source theme from workspace, not store |
| fe store | `react/stores/sqlEditor/editor.ts`, `utils/storage-keys.ts` | remove per-user theme path |
| fe settings | `react/pages/settings/general/SQLEditorSection.tsx` | + theme block (preset + custom + preview) |
| fe settings | `react/components/.../ThemePreview.tsx` | new — live preview |
| fe cleanup | `react/components/sql-editor/ThemeSelect.tsx` (+test), `EditorAction.tsx` | remove dev switcher |
| i18n | `src/locales/*.json` (all 5) | settings labels (theme, preset, anchors, custom) |

## 7. Future work

- **App-wide branding (phase 2):** expand the token set to the full app `--color-*`, apply
  the workspace theme at `:root` (or an app-root scope), and audit the app for hardcoded
  colors. The model here (`SQLEditorThemeSetting` → a broader `AppearanceSetting`) extends
  without rework.
- **Workspace catalog of multiple custom themes:** let an admin author several custom
  themes and enforce one. Proto change: `repeated SQLEditorThemeSetting custom_themes` + an
  `enforced_theme_id` (a built-in id or a custom uuid). The frontend's catalog-composition
  (§2.1) already assumes "custom theme(s)" plural, so the picker/resolution generalize
  without rework.
- **Per-user override** of the workspace theme (opt-in), if governance later softens.
- **Import/export theme string** (Linear-style paste-to-share) — `tokens`/anchors already
  serialize cleanly.
- **More built-in presets** once authored.
