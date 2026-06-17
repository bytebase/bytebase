# Workspace-Enforced SQL Editor Theme — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A workspace admin enforces one SQL Editor theme (built-in preset or anchor-derived custom theme), stored in the backend `WorkspaceProfileSetting`, applied to every user's SQL Editor.

**Architecture:** Reuse the existing SQL-Editor token pipeline (`SQLEditorThemeScope`, `PRESETS`, `derive`, overlay, `color-scheme`) unchanged — only swap the *source* from per-user localStorage to the workspace setting. Two new proto fields: `sql_editor_theme_id` (opaque enforced-selection id) + `sql_editor_custom_theme` (custom definition, tokens always complete). Built-ins stay frontend-only references. Admin authoring lives in `SQLEditorSection`.

**Tech Stack:** Go (Connect/gRPC `SettingService`), protobuf/buf, React + Zustand + Base UI, Tailwind v4 tokens.

**Design doc:** [`docs/superpowers/specs/2026-06-16-workspace-sql-editor-theme-design.md`](../specs/2026-06-16-workspace-sql-editor-theme-design.md)

**Decisions locked:** workspace-enforced (no per-user pick); SQL-Editor scope only (app-wide = future); 5 anchors (background/surface/text/accent/border); status colors (info/warning/error/success) + matrix-green stay fixed semantic; custom `tokens` required (no empty-token store state).

**Conventions:** Go — `gofmt -w`, `golangci-lint run --allow-parallel-runners` (repeat to clean). Frontend — `pnpm --dir frontend fix`, `type-check`, `test`, `node scripts/check-react-layering.mjs`. Commit after each task (the implementer commits; never on `main`).

---

## File Structure

- `proto/v1/v1/setting_service.proto` **and** `proto/store/store/setting.proto` — `+message SQLEditorThemeSetting`, `+sql_editor_theme_id`, `+sql_editor_custom_theme` (BOTH layers).
- `backend/api/v1/setting_service_converter.go` — map both fields in `convert{,To}WorkspaceProfileSetting`.
- `backend/api/v1/setting_service.go` (+`_test.go`) — update-mask cases + validate custom theme.
- `frontend/src/react/components/sql-editor/theme/derive.ts` (+`derive.test.ts`) — anchors + derivation.
- `frontend/src/react/components/sql-editor/theme/useWorkspaceSQLEditorTheme.ts` (+`.test.ts`) — resolver.
- `frontend/src/react/components/sql-editor/SQLEditorLayout.tsx` — source theme from workspace.
- `frontend/src/react/stores/sqlEditor/editor.ts`, `frontend/src/utils/storage-keys.ts` — remove per-user path.
- `frontend/src/react/components/sql-editor/ThemeSelect.tsx` (+test), `EditorAction.tsx` — remove dev switcher.
- `frontend/src/react/pages/settings/general/SQLEditorSection.tsx` (+test) — admin theme block.
- `frontend/src/react/pages/settings/general/sql-editor-theme/{ThemeAnchorEditor,ThemePreview}.tsx` — editor + preview.
- `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` — labels.

Order: backend contract (1-2) → frontend core logic (3-4) → wiring (5-6) → UI (7-9) → i18n + review (10-11).

---

## Task 1: Proto — theme fields on BOTH proto layers

> **Critical (Codex P1):** `WorkspaceProfileSetting` exists in **two** protos — the public
> v1 API (`proto/v1/v1/setting_service.proto`) **and** the persisted store
> (`proto/store/store/setting.proto:37`). `UpdateSetting` converts v1→store via
> `convertWorkspaceProfileSetting` and reads back store→v1 via
> `convertToWorkspaceProfileSetting`. Both proto layers AND both converters (Task 2) must
> carry the new fields, or every value is dropped at the store boundary.

**Files:** `proto/v1/v1/setting_service.proto`, `proto/store/store/setting.proto`.

- [ ] **Step 1 (v1 proto):** In `message WorkspaceProfileSetting`, after `allow_email_code_signin = 22;`:

```proto
  // Enforced SQL Editor theme id: OPAQUE — a frontend-resolved built-in preset id
  // OR a custom theme's uuid. Empty ⇒ default light.
  string sql_editor_theme_id = 23;
  // The enforced CUSTOM theme's full definition — present ONLY when
  // sql_editor_theme_id is a custom uuid. tokens is always complete.
  SQLEditorThemeSetting sql_editor_custom_theme = 24;
```

Add at top level (after `WorkspaceProfileSetting`):

```proto
message SQLEditorThemeSetting {
  string id = 1;
  string name = 2;
  string monaco_base = 3;          // "vs" | "vs-dark"
  map<string, string> tokens = 4;  // ~29 "--color-*" → "r g b"; required & complete
}
```

- [ ] **Step 2 (store proto):** Mirror the SAME additions in `proto/store/store/setting.proto`'s `message WorkspaceProfileSetting` (use its next free field numbers — check the file; do NOT assume 23/24 match the v1 numbers) and add an identical `message SQLEditorThemeSetting`. The store proto is the persisted shape.
- [ ] **Step 3:** `buf format -w proto && buf lint proto && (cd proto && buf generate)`. Expected: no lint errors; both `v1pb` and `storepb` now expose `SQLEditorThemeSetting`, `SqlEditorThemeId`, `SqlEditorCustomTheme`; TS `frontend/src/types/proto-es/...` exposes `sqlEditorThemeId` / `sqlEditorCustomTheme`.
- [ ] **Step 4:** Commit `feat(proto): add workspace SQL Editor theme setting (v1 + store)` (include `proto`, `backend/generated-go`, `frontend/src/types`).

---

## Task 2: Backend — converters, update-mask cases, validation

**Files:** `backend/api/v1/setting_service_converter.go` (converters), `backend/api/v1/setting_service.go` (mask switch + validator), `setting_service_test.go`.

The validator runs on the **store** payload (the update switch builds `storepb.WorkspaceProfileSetting`), so it takes `*storepb.SQLEditorThemeSetting`.

- [ ] **Step 1: Failing test** — append to `setting_service_test.go`:

```go
func TestValidateSQLEditorCustomTheme(t *testing.T) {
	full := func() map[string]string {
		m := map[string]string{}
		for _, k := range sqlEditorThemeTokenKeys {
			m[k] = "1 2 3"
		}
		return m
	}
	drop := func(k string) *storepb.SQLEditorThemeSetting {
		m := full()
		delete(m, k)
		return &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: m}
	}
	bad := func(k, v string) *storepb.SQLEditorThemeSetting {
		m := full()
		m[k] = v
		return &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: m}
	}
	ok := &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: full()}
	cases := []struct {
		name    string
		theme   *storepb.SQLEditorThemeSetting
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"complete", ok, false},
		{"empty tokens", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "vs-dark", Tokens: map[string]string{}}, true},
		{"missing token", drop("--color-accent"), true},
		{"bad triple", bad("--color-accent", "300 0 0"), true},
		{"empty id", &storepb.SQLEditorThemeSetting{Id: "", Name: "Brand", MonacoBase: "vs-dark", Tokens: full()}, true},
		{"empty name", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "", MonacoBase: "vs-dark", Tokens: full()}, true},
		{"bad base", &storepb.SQLEditorThemeSetting{Id: "u1", Name: "Brand", MonacoBase: "x", Tokens: full()}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSQLEditorCustomTheme(tc.theme)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 2:** Run `go test ./backend/api/v1/ -run TestValidateSQLEditorCustomTheme` → FAIL (undefined symbols).

- [ ] **Step 3: Implement** in `setting_service.go` (`sqlEditorThemeTokenKeys` mirrors the 29 frontend `SQL_EDITOR_THEME_TOKENS` — copy the exact list from `frontend/src/react/components/sql-editor/theme/types.ts`):

```go
var sqlEditorThemeTokenKeys = []string{ /* the 29 "--color-*" keys, verbatim from types.ts */ }

func validateSQLEditorCustomTheme(t *storepb.SQLEditorThemeSetting) error {
	if t == nil {
		return nil // built-in reference or unset
	}
	if t.Id == "" {
		return status.Errorf(codes.InvalidArgument, "sql_editor_custom_theme.id is required")
	}
	if t.Name == "" {
		return status.Errorf(codes.InvalidArgument, "sql_editor_custom_theme.name is required")
	}
	if t.MonacoBase != "vs" && t.MonacoBase != "vs-dark" {
		return status.Errorf(codes.InvalidArgument, "sql_editor_custom_theme.monaco_base must be vs or vs-dark")
	}
	for _, k := range sqlEditorThemeTokenKeys {
		v, ok := t.Tokens[k]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "sql_editor_custom_theme missing token %s", k)
		}
		if !isRGBTriple(v) {
			return status.Errorf(codes.InvalidArgument, "sql_editor_custom_theme token %s invalid: %q", k, v)
		}
	}
	return nil
}

func isRGBTriple(s string) bool {
	parts := strings.Fields(s)
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Converters** (Codex P1) — in `setting_service_converter.go`, map the two new
  fields in **both** directions. Add a small `SQLEditorThemeSetting` v1↔store helper (fields
  are identical: Id, Name, MonacoBase, Tokens), then in `convertWorkspaceProfileSetting`
  (v1→store) set `SqlEditorThemeId` + `SqlEditorCustomTheme`, and in
  `convertToWorkspaceProfileSetting` (store→v1) set them back. Without this the values never
  reach/leave the DB.

- [ ] **Step 5: Update-mask cases** — in `setting_service.go`, the `WORKSPACE_PROFILE` update
  has a `switch path` over `value.workspace_profile.*` where each case copies one field from
  the incoming (converted-to-store) `payload` into `oldSetting`, and `default` rejects
  unknown paths with `InvalidArgument`. Add two cases (before `default`):
```go
case "value.workspace_profile.sql_editor_theme_id":
	oldSetting.SqlEditorThemeId = payload.SqlEditorThemeId
case "value.workspace_profile.sql_editor_custom_theme":
	if err := validateSQLEditorCustomTheme(payload.SqlEditorCustomTheme); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	oldSetting.SqlEditorCustomTheme = payload.SqlEditorCustomTheme // nil clears it (built-in)
```
`sql_editor_theme_id` is opaque (no validation); a nil `sql_editor_custom_theme` is valid and clears the field (built-in selection).

- [ ] **Step 6:** `go test ./backend/api/v1/ -run TestValidateSQLEditorCustomTheme` → PASS; `golangci-lint run --allow-parallel-runners` clean. (Optional: a service-level update test asserting a custom theme round-trips through store + that picking a built-in clears `SqlEditorCustomTheme`.)
- [ ] **Step 7:** Commit `feat(setting): persist + validate workspace SQL Editor theme`.

---

## Task 3: Frontend — anchor-derive helpers

**Files:** Modify `theme/derive.ts`; append `theme/derive.test.ts`.

**Derivation contract:** 5 anchors → 29 tokens. Anchors map directly to their primary token; the rest are derived by mixing toward `text` (elevate) or `background` (recede). `monacoBase = luminance(background) < 0.5 ? "vs-dark" : "vs"`. The 10 status/matrix tokens are FIXED (copied from `PRESET_BY_ID.light`). `themeToAnchors` reads the 5 primary tokens back. Exact mix deltas are tunable against the live preview; tests assert the **contract** (completeness, direct mappings, fixed status, base-by-luminance, round-trip), not brittle golden RGB.

- [ ] **Step 1: Failing tests** — append to `derive.test.ts`:

```ts
import { deriveThemeFromAnchors, themeToAnchors, type ThemeAnchors } from "./derive";

const lightAnchors: ThemeAnchors = { background: "#ffffff", surface: "#f3f4f6", text: "#18181b", accent: "#4f46e5", border: "#e5e7eb" };
const darkAnchors: ThemeAnchors = { background: "#1e1e1e", surface: "#374151", text: "#f4f4f5", accent: "#6366f1", border: "#3f3f46" };

describe("deriveThemeFromAnchors", () => {
  test("complete + valid", () => {
    const t = deriveThemeFromAnchors(lightAnchors, "Brand");
    expect(() => validateTheme(t)).not.toThrow();
    expect(Object.keys(t.tokens)).toHaveLength(SQL_EDITOR_THEME_TOKENS.length);
    expect(t.name).toBe("Brand");
  });
  test("anchors map to direct tokens", () => {
    const t = deriveThemeFromAnchors(lightAnchors, "B");
    expect(t.tokens["--color-background"]).toBe("255 255 255");
    expect(t.tokens["--color-control-bg"]).toBe("243 244 246");
    expect(t.tokens["--color-main"]).toBe("24 24 27");
    expect(t.tokens["--color-accent"]).toBe("79 70 229");
    expect(t.tokens["--color-block-border"]).toBe("229 231 235");
  });
  test("monacoBase by luminance", () => {
    expect(deriveThemeFromAnchors(lightAnchors, "L").monacoBase).toBe("vs");
    expect(deriveThemeFromAnchors(darkAnchors, "D").monacoBase).toBe("vs-dark");
  });
  test("status + matrix fixed (anchor-independent)", () => {
    const a = deriveThemeFromAnchors(lightAnchors, "A").tokens;
    const b = deriveThemeFromAnchors(darkAnchors, "B").tokens;
    for (const k of ["--color-error", "--color-warning", "--color-success", "--color-info", "--color-matrix-green"] as const) {
      expect(a[k]).toBe(b[k]);
      expect(a[k]).toBe(PRESET_BY_ID.light.tokens[k]);
    }
  });
});

describe("themeToAnchors", () => {
  test("round-trips", () => {
    expect(themeToAnchors(deriveThemeFromAnchors(darkAnchors, "D"))).toEqual(darkAnchors);
  });
});
```
(Add `import { PRESET_BY_ID } from "./presets";` if not already present in the test file.)

- [ ] **Step 2:** Run `npx vitest run src/react/components/sql-editor/theme/derive.test.ts` → FAIL (undefined).

- [ ] **Step 3: Implement** in `derive.ts`:

```ts
import { PRESET_BY_ID } from "./presets";
import { SQL_EDITOR_THEME_TOKENS } from "./types";

export type Hex = string;
export interface ThemeAnchors { background: Hex; surface: Hex; text: Hex; accent: Hex; border: Hex; }

type RGB3 = [number, number, number];
const hexToRgb = (h: Hex): RGB3 => { const x = h.replace("#", ""); return [0, 2, 4].map((i) => parseInt(x.slice(i, i + 2), 16)) as RGB3; };
const toStr = (c: RGB3): string => c.map((n) => Math.max(0, Math.min(255, Math.round(n)))).join(" ");
const fromStr = (s: string): RGB3 => s.split(" ").map(Number) as RGB3;
const toHex = (s: string): Hex => "#" + fromStr(s).map((n) => n.toString(16).padStart(2, "0")).join("");
const mix = (a: RGB3, b: RGB3, t: number): RGB3 => a.map((v, i) => v + (b[i] - v) * t) as RGB3;
const luminance = ([r, g, b]: RGB3): number => (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;

// Status + matrix tokens keep their semantic values across brands.
const FIXED_TOKENS = [
  "--color-info", "--color-info-hover", "--color-warning", "--color-warning-hover",
  "--color-error", "--color-error-hover", "--color-success", "--color-success-hover",
  "--color-matrix-green", "--color-matrix-green-hover",
] as const;

export function deriveThemeFromAnchors(anchors: ThemeAnchors, name: string): SQLEditorTheme {
  const bg = hexToRgb(anchors.background);
  const surface = hexToRgb(anchors.surface);
  const text = hexToRgb(anchors.text);
  const accent = hexToRgb(anchors.accent);
  const border = hexToRgb(anchors.border);
  const dark = luminance(bg) < 0.5;
  // "elevate" = toward text (more contrast); "recede" = toward background.
  const elevate = (c: RGB3, t: number) => mix(c, text, t);
  const recede = (c: RGB3, t: number) => mix(c, bg, t);

  const tokens: Record<string, string> = {
    "--color-background": toStr(bg),
    "--color-dark-bg": toStr(bg),
    "--color-control-bg": toStr(surface),
    "--color-control-bg-hover": toStr(elevate(surface, 0.06)),
    "--color-control": toStr(text),
    "--color-control-hover": toStr(elevate(text, dark ? -0 : 0.12)), // text is already max-contrast; hover nudges
    "--color-control-light": toStr(recede(text, 0.3)),
    "--color-control-light-hover": toStr(recede(text, 0.15)),
    "--color-control-placeholder": toStr(recede(text, 0.5)),
    "--color-control-border": toStr(border),
    "--color-block-border": toStr(border),
    "--color-link-hover": toStr(border),
    "--color-accent": toStr(accent),
    "--color-accent-hover": toStr(dark ? elevate(accent, 0.15) : recede(accent, 0.2)),
    "--color-accent-disabled": toStr(recede(accent, 0.5)),
    "--color-accent-text": luminance(accent) < 0.5 ? "255 255 255" : "24 24 27",
    "--color-main": toStr(text),
    "--color-main-hover": toStr(recede(text, 0.2)),
    "--color-main-text": luminance(text) < 0.5 ? "255 255 255" : "24 24 27",
  };
  for (const k of FIXED_TOKENS) tokens[k] = PRESET_BY_ID.light.tokens[k];

  const theme: SQLEditorTheme = {
    id: "", // set by caller (uuid)
    name,
    monacoBase: dark ? "vs-dark" : "vs",
    tokens: tokens as SQLEditorTheme["tokens"],
  };
  validateTheme(theme); // throws if a key was missed → caught by the completeness test
  return theme;
}

export function themeToAnchors(theme: SQLEditorTheme): ThemeAnchors {
  return {
    background: toHex(theme.tokens["--color-background"]),
    surface: toHex(theme.tokens["--color-control-bg"]),
    text: toHex(theme.tokens["--color-main"]),
    accent: toHex(theme.tokens["--color-accent"]),
    border: toHex(theme.tokens["--color-block-border"]),
  };
}
```

Note: the derived-token formulas (deltas) are a starting point — tune visually via the live preview (Task 9). Keep every `SQL_EDITOR_THEME_TOKENS` key assigned (the completeness test + `validateTheme` enforce this). The `id` is filled by the editor caller with a uuid.

- [ ] **Step 4:** Run the derive tests → PASS. `pnpm --dir frontend type-check`.
- [ ] **Step 5:** Commit `feat(sql-editor-theme): anchor-derive helpers`.

---

## Task 4: Frontend — workspace theme resolver

**Files:** Create `theme/useWorkspaceSQLEditorTheme.ts` (+`.test.ts`).

- [ ] **Step 1: Failing test** (`useWorkspaceSQLEditorTheme.test.ts`) — mock the app store's `getWorkspaceProfile`:

```ts
// resolveWorkspaceTheme(profile) is the pure core; test it directly.
import { resolveWorkspaceTheme } from "./useWorkspaceSQLEditorTheme";
import { PRESET_BY_ID } from "./presets";
import { deriveThemeFromAnchors } from "./derive";

const customDef = { ...deriveThemeFromAnchors({ background:"#1e1e1e", surface:"#374151", text:"#f4f4f5", accent:"#6366f1", border:"#3f3f46" }, "Brand"), id: "u1" };

test("custom matching id → custom", () => {
  expect(resolveWorkspaceTheme({ sqlEditorThemeId: "u1", sqlEditorCustomTheme: customDef })?.id).toBe("u1");
});
test("known preset id → catalog", () => {
  expect(resolveWorkspaceTheme({ sqlEditorThemeId: "dark", sqlEditorCustomTheme: undefined })).toBe(PRESET_BY_ID.dark);
});
test("unknown/empty id → light", () => {
  expect(resolveWorkspaceTheme({ sqlEditorThemeId: "", sqlEditorCustomTheme: undefined })).toBe(PRESET_BY_ID.light);
  expect(resolveWorkspaceTheme({ sqlEditorThemeId: "nope", sqlEditorCustomTheme: undefined })).toBe(PRESET_BY_ID.light);
});
test("malformed custom → light", () => {
  expect(resolveWorkspaceTheme({ sqlEditorThemeId: "u1", sqlEditorCustomTheme: { id:"u1", name:"x", monacoBase:"vs", tokens:{} } })).toBe(PRESET_BY_ID.light);
});
```

- [ ] **Step 2:** Run → FAIL.

- [ ] **Step 3: Implement** `useWorkspaceSQLEditorTheme.ts`:

```ts
import { useAppStore } from "@/react/stores/app";
import { resolveThemeId, PRESET_BY_ID } from "./presets";
import { validateTheme } from "./derive";
import type { SQLEditorTheme } from "./types";

// Shape mirrors WorkspaceProfileSetting's two theme fields (protojson camelCase).
interface WorkspaceThemeInput {
  sqlEditorThemeId?: string;
  sqlEditorCustomTheme?: { id: string; name: string; monacoBase: string; tokens: Record<string, string> };
}

export function resolveWorkspaceTheme(p: WorkspaceThemeInput): SQLEditorTheme {
  const id = p.sqlEditorThemeId ?? "";
  const custom = p.sqlEditorCustomTheme;
  if (custom && custom.id === id) {
    const theme = { id: custom.id, name: custom.name, monacoBase: custom.monacoBase as SQLEditorTheme["monacoBase"], tokens: custom.tokens as SQLEditorTheme["tokens"] };
    try { validateTheme(theme); return theme; } catch { return PRESET_BY_ID.light; }
  }
  return resolveThemeId(id); // catalog; light fallback for unknown/empty
}

export function useWorkspaceSQLEditorTheme(): SQLEditorTheme {
  const profile = useAppStore((s) => s.getWorkspaceProfile());
  return resolveWorkspaceTheme(profile as WorkspaceThemeInput);
}
```
(Confirm `resolveThemeId("")` and unknown ids fall back to `light` — they do, per the existing `presets/index.ts`.)

- [ ] **Step 4:** Run → PASS; `type-check`.
- [ ] **Step 5:** Commit `feat(sql-editor-theme): workspace theme resolver`.

---

## Task 5: Wire the resolver into the layout (source swap)

**Files:** Modify `SQLEditorLayout.tsx`.

- [ ] **Step 1:** In `SQLEditorThemeRoot`, replace the localStorage read with the resolver:
```tsx
// before: const themeId = useSQLEditorEditorState((s) => s.themeId);
//         <SQLEditorThemeScope theme={resolveThemeId(themeId)} asContents>
const theme = useWorkspaceSQLEditorTheme();
// ...
<SQLEditorThemeScope theme={theme} asContents>
```
Keep `useMonacoThemeController()` + `useSQLEditorOverlayTheme()` calls. `useActiveSQLEditorTheme` continues to resolve admin mode the same way (it already reads the active theme via the scope chain / store — verify it now reads from `useWorkspaceSQLEditorTheme` indirectly or update its source to match).
- [ ] **Step 2:** `pnpm --dir frontend type-check`; run the SQL-editor theme tests (`theme/`) → green. Manually verify the editor renders the default (`light`) when no workspace theme is set.
- [ ] **Step 3:** Commit `feat(sql-editor): source theme from workspace setting`.

---

## Task 6: Remove the per-user theme path

**Files:** `stores/sqlEditor/editor.ts`, `utils/storage-keys.ts`, `ThemeSelect.tsx` (+test), `EditorAction.tsx`, `stores/sqlEditor/editor.test.ts`.

- [ ] **Step 1:** Delete from `stores/sqlEditor/editor.ts`: `themeId` field, `setThemeId`, `readThemeId`, `themeKey`, and the `themeId: readThemeId()` initializer entry + the interface member. Remove the `storageKeySqlEditorTheme` import.
- [ ] **Step 2:** Delete `storageKeySqlEditorTheme` from `utils/storage-keys.ts`. Delete the "theme persistence" tests from `stores/sqlEditor/editor.test.ts`.
- [ ] **Step 3:** Delete `ThemeSelect.tsx` + `ThemeSelect.test.tsx`. In `EditorAction.tsx`, remove the `{isDev() && <ThemeSelect />}` line and the `ThemeSelect` import (and `isDev` import if now unused).
- [ ] **Step 4:** `pnpm --dir frontend fix && type-check && test` (run the editor-store + EditorAction tests). Grep `grep -rn "ThemeSelect\|setThemeId\|storageKeySqlEditorTheme" frontend/src` → no results.
- [ ] **Step 5:** Commit `refactor(sql-editor): remove per-user/dev theme switcher`.

---

## Task 7: Admin editor — preset picker (SQLEditorSection)

**Files:** Modify `react/pages/settings/general/SQLEditorSection.tsx`; test `SQLEditorSection.test.tsx`.

Follow the existing `SQLEditorSection` pattern exactly: read `useAppStore((s) => s.getWorkspaceProfile())`; gate writes with `hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile")` + `<PermissionGuard permissions={["bb.settings.setWorkspaceProfile"]}>`; participate in the section's `SectionHandle` (`isDirty` / `revert` / `update` via `useImperativeHandle`).

- [ ] **Step 1:** Add a "Theme" block. Local state: `selectedThemeId` + `customDraft: SQLEditorTheme | null`. Initialize from the profile (`sqlEditorThemeId`, `sqlEditorCustomTheme`). The picker = radio over `PRESETS` (built-ins) + a "Custom" option; the picker list is `[customDraft if present] ++ PRESETS` (§2.1 composition).
- [ ] **Step 2:** Selecting a **built-in** sets `selectedThemeId = preset.id`, `customDraft = null`. `isDirty` = differs from the stored profile values.
- [ ] **Step 3:** `update()` writes:
```tsx
// Codex P2: the method takes `payload` (NOT `setting`), and mask paths are the
// full `value.workspace_profile.*` form (see query_timeout / sql_result_size in
// the same file). Both fields go in one call.
await useAppStore.getState().updateWorkspaceProfile({
  payload: { sqlEditorThemeId: selectedThemeId, sqlEditorCustomTheme: customDraft ?? undefined },
  updateMask: create(FieldMaskSchema, {
    paths: [
      "value.workspace_profile.sql_editor_theme_id",
      "value.workspace_profile.sql_editor_custom_theme",
    ],
  }),
});
```
- [ ] **Step 4: Test** (`SQLEditorSection.test.tsx`) — selecting a different preset sets `isDirty`; `revert` restores; `update` calls `updateWorkspaceProfile` with `sqlEditorThemeId` = the picked id and `sqlEditorCustomTheme` cleared, and the right mask; without `bb.settings.setWorkspaceProfile` the controls are read-only.
- [ ] **Step 5:** `fix && type-check && test` + `node scripts/check-react-layering.mjs`.
- [ ] **Step 6:** Commit `feat(settings): workspace SQL Editor theme preset picker`.

---

## Task 8: Admin editor — anchor editor (custom)

**Files:** Create `react/pages/settings/general/sql-editor-theme/ThemeAnchorEditor.tsx`; wire into `SQLEditorSection`.

- [ ] **Step 1:** `ThemeAnchorEditor` props: `{ value: ThemeAnchors; name: string; onChange(anchors, name): void }`. Render 5 labeled color inputs (use the shared UI color input if one exists in `react/components/ui/`; else a native `<input type="color">` is acceptable here — note it in the PR) + a name text field. All labels via i18n (Task 10).
- [ ] **Step 2:** In `SQLEditorSection`, when "Custom" is selected: on first switch, generate the id with `import { v4 as uuidv4 } from "uuid"` → `customDraft = { ...deriveThemeFromAnchors(themeToAnchors(currentResolvedTheme), defaultName), id: uuidv4() }`; render `<ThemeAnchorEditor>`; on change → `customDraft = { ...deriveThemeFromAnchors(nextAnchors, name), id: keptUuid }` (preserve the uuid across edits). `selectedThemeId = customDraft.id`. **Do NOT use `crypto.randomUUID()`** — `pnpm check` runs `scripts/check-no-crypto-randomuuid.mjs` which bans it (use `uuid`'s `v4`).
- [ ] **Step 3: Test** — editing an anchor updates `customDraft.tokens` (derived) and keeps `isDirty`; the uuid is stable across edits; `update` writes `sqlEditorCustomTheme` with complete tokens + `sqlEditorThemeId === customDraft.id`.
- [ ] **Step 4:** `fix && type-check && test`.
- [ ] **Step 5:** Commit `feat(settings): custom SQL Editor theme anchor editor`.

---

## Task 9: Admin editor — live preview

**Files:** Create `react/pages/settings/general/sql-editor-theme/ThemePreview.tsx`; render in `SQLEditorSection` next to the editor.

- [ ] **Step 1:** `ThemePreview` props `{ theme: SQLEditorTheme }`. Wrap a representative slice in `<SQLEditorThemeScope theme={theme}>` (a non-`asContents` bordered box) showing: a primary (accent) button, a ghost button, a text input, a result-grid-style header row + two data rows, a code-ish line, and muted/secondary text — using semantic token classes only (`bg-background`, `text-main`, `text-control-light`, `border-block-border`, `bg-control-bg`, `bg-accent`, etc.). No Monaco. Set `color-scheme` via the scope (already handled by `SQLEditorThemeScope`).
- [ ] **Step 2:** Pass the live `customDraft` (custom) or the resolved preset (built-in) as `theme`, so the admin sees changes immediately. Use this preview to **tune the Task 3 derive deltas** until light/dark/a sample brand look right.
- [ ] **Step 3: Test** — `ThemePreview` writes the theme's `--color-background` as an inline CSS var on its container (mirrors the existing `SQLEditorThemeScope` test).
- [ ] **Step 4:** `fix && type-check && test` + layering scan.
- [ ] **Step 5:** Commit `feat(settings): SQL Editor theme live preview`.

---

## Task 10: i18n

**Files:** `frontend/src/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` (all 5).

- [ ] **Step 1:** Add keys under the settings SQL-Editor section, e.g. `settings.sql-editor.theme.{self, preset, custom, name, anchor-background, anchor-surface, anchor-text, anchor-accent, anchor-border, preview}`. Translate in **all 5** files (not just en-US). Follow existing sibling-key plural conventions; no empty objects.
- [ ] **Step 2:** `pnpm --dir frontend fix` (runs the locale sorter) + `type-check`. Grep the new components for any hardcoded user-facing string → none.
- [ ] **Step 3:** Commit `i18n: workspace SQL Editor theme settings`.

---

## Task 11: Final review + gates

- [ ] **Step 1:** Full gates — backend: `gofmt`, `golangci-lint run --allow-parallel-runners` (clean), `go test ./backend/api/v1/...`, `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`. Frontend: `pnpm --dir frontend check && type-check && test` + `node scripts/check-react-layering.mjs`.
- [ ] **Step 2:** Manual smoke (dev server): as admin, pick `dark` → all users' editors go dark (re-login/refresh another session); author a custom theme via anchors → preview matches → save → applied; the native datetime picker + scrollbars follow `color-scheme`; non-admin sees the section read-only; a workspace with no theme set renders identical to today's light.
- [ ] **Step 3:** Walk `docs/pre-pr-checklist.md` (breaking-change/proto, lint/test gates, SonarCloud props). Open the PR.

---

## Out of scope (tracked in the spec's Future work)

App-wide branding; multiple-custom-theme catalog (`repeated` + enforced id); per-user override; import/export theme string; more built-in presets.
