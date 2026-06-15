# SQL Editor Theme Pattern Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the SQL Editor's colors into a data-driven theme model and add a runtime preset switcher (6 themes) scoped to the SQL Editor, compatible with admin/terminal mode.

**Architecture:** A theme is one plain object (`tokens` chrome layer + `editor`/`syntax` Monaco layer). Chrome is themed per-scope by writing the `tokens` as inline `--color-*` CSS vars on a `<SQLEditorThemeScope>` container (CSS cascade re-themes Tailwind classes). Monaco — whose theme is necessarily global — is driven by a single controller off the foreground panel. Admin mode is a nested scope whose theme is `selected.isDark ? selected : dark`. Selection persists in localStorage.

**Tech Stack:** React, Zustand (+ immer), Tailwind v4 CSS-variable utilities, Base UI `Select`, monaco-editor, Vitest + Testing Library.

**Spec:** `docs/superpowers/specs/2026-06-09-sql-editor-theme-pattern-design.md`

**Phasing / handoff value:**
- **Phase 1–3** deliver a working theme switch for the editor chrome + Monaco surface (the named-theme look lands, though some hardcoded islands still show until Phase 5).
- **Phase 4** wires scopes (root, admin, portals) so admin mode and portaled surfaces theme correctly.
- **Phase 5** removes hardcoded colors so every surface follows the theme.
- **Phase 6** adds the user-facing switcher. **Phase 7** verifies.

---

## File Structure

New (theme module under the SQL Editor React tree):
- `frontend/src/react/components/sql-editor/theme/types.ts` — `SQLEditorTheme` + supporting types.
- `frontend/src/react/components/sql-editor/theme/derive.ts` — `themeToCssVars`, `buildMonacoTheme`, `monacoThemeName`, `resolveAdminTheme`, `validateTheme`.
- `frontend/src/react/components/sql-editor/theme/derive.test.ts`
- `frontend/src/react/components/sql-editor/theme/presets/{light,dark,solarized-light,solarized-dark,monokai,nord}.ts`
- `frontend/src/react/components/sql-editor/theme/presets/index.ts` — `PRESETS`, `PRESET_BY_ID`, `DEFAULT_THEME_ID`.
- `frontend/src/react/components/sql-editor/theme/presets/presets.test.ts`
- `frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.tsx` — context + scope component + `useSQLEditorTheme`.
- `frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx`
- `frontend/src/react/components/sql-editor/theme/useMonacoThemeController.ts` — global Monaco theme controller.
- `frontend/src/react/components/sql-editor/ThemeSelect.tsx` — switcher.

Modified:
- `frontend/src/utils/storage-keys.ts` — add `storageKeySqlEditorTheme`.
- `frontend/src/react/stores/sqlEditor/editor.ts` (+ `editor.test.ts`) — `themeId` + `setThemeId`.
- `frontend/src/react/components/monaco/core.ts` — register the 6 generated themes.
- `frontend/src/react/components/monaco/themes/{bb,bb-dark}.ts` — deleted (superseded by generated themes).
- `frontend/src/react/components/sql-editor/SQLEditorLayout.tsx` — root scope + Monaco controller mount.
- `frontend/src/react/components/sql-editor/TerminalPanel/TerminalPanel.tsx` — nested admin scope, `bg-dark-bg`→token.
- `frontend/src/react/components/sql-editor/TerminalPanel/CompactSQLEditor.tsx` — drop hardcoded `vs-dark`.
- `frontend/src/react/components/sql-editor/ResultView/ResultView.tsx` — drop `dark` prop.
- Result-view + standard-mode components — hardcoded color migration (Phase 5, exact list).
- `frontend/src/react/components/sql-editor/SQLEditorHomePage.tsx` — portal scopes.
- `frontend/src/react/components/sql-editor/RequestDrawerHost.tsx` / `AccessGrantRequestDrawer.tsx` — drawer chrome scope.
- `frontend/src/react/components/sql-editor/EditorAction.tsx` — mount `ThemeSelect`.
- `frontend/src/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — i18n key.

**Commands** (run from repo root):
- Test one file: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/derive.test.ts`
- Test all: `pnpm --dir frontend test`
- Lint/format/imports: `pnpm --dir frontend fix`
- Type check: `pnpm --dir frontend type-check`

---

## Phase 1 — Theme model foundation

### Task 1: Theme types

**Files:**
- Create: `frontend/src/react/components/sql-editor/theme/types.ts`

- [ ] **Step 1: Write the types**

```ts
// frontend/src/react/components/sql-editor/theme/types.ts

// Chrome tokens are "r g b" triples (Tailwind utilities resolve to
// `rgb(var(--color-x) / <alpha>)`). Monaco values are "#rrggbb" hex.
export type RGB = string;
export type Hex = string;

// The SQL-Editor subset of the existing --color-* names. Every preset MUST
// fill all of these (enforced by validateTheme).
export type SQLEditorThemeToken =
  | "--color-control"
  | "--color-control-hover"
  | "--color-control-light"
  | "--color-control-light-hover"
  | "--color-control-bg"
  | "--color-control-bg-hover"
  | "--color-control-placeholder"
  | "--color-control-border"
  | "--color-accent"
  | "--color-accent-hover"
  | "--color-accent-disabled"
  | "--color-accent-text"
  | "--color-main"
  | "--color-main-hover"
  | "--color-main-text"
  | "--color-background"
  | "--color-block-border"
  | "--color-link-hover"
  | "--color-info"
  | "--color-info-hover"
  | "--color-warning"
  | "--color-warning-hover"
  | "--color-error"
  | "--color-error-hover"
  | "--color-success"
  | "--color-success-hover"
  | "--color-matrix-green"
  | "--color-matrix-green-hover"
  | "--color-dark-bg";

// Monaco editor-surface colors (canvas — unreachable by CSS cascade).
export interface EditorChromeColors {
  background: Hex;
  selectionBackground: Hex;
  cursor: Hex;
  lineHighlight: Hex;
  gutterBackground: Hex;
  lineNumber: Hex;
  activeLineNumber: Hex;
}

// Normalized, language-agnostic syntax palette. buildMonacoTheme maps these
// onto Monaco token rules (incl. SQL scopes).
export interface SyntaxPalette {
  comment: Hex;
  keyword: Hex;
  string: Hex;
  number: Hex;
  type: Hex;
  function: Hex;
  variable: Hex;
  operator: Hex;
  delimiter: Hex;
  predefined: Hex;
}

export interface SQLEditorTheme {
  id: string;
  // i18n key for the display label (built-in palette proper nouns may be literal).
  name: string;
  isDark: boolean;
  monacoBase: "vs" | "vs-dark";
  tokens: Record<SQLEditorThemeToken, RGB>;
  editor: EditorChromeColors;
  syntax: SyntaxPalette;
}

// All chrome token names, for iteration/validation.
export const SQL_EDITOR_THEME_TOKENS: SQLEditorThemeToken[] = [
  "--color-control",
  "--color-control-hover",
  "--color-control-light",
  "--color-control-light-hover",
  "--color-control-bg",
  "--color-control-bg-hover",
  "--color-control-placeholder",
  "--color-control-border",
  "--color-accent",
  "--color-accent-hover",
  "--color-accent-disabled",
  "--color-accent-text",
  "--color-main",
  "--color-main-hover",
  "--color-main-text",
  "--color-background",
  "--color-block-border",
  "--color-link-hover",
  "--color-info",
  "--color-info-hover",
  "--color-warning",
  "--color-warning-hover",
  "--color-error",
  "--color-error-hover",
  "--color-success",
  "--color-success-hover",
  "--color-matrix-green",
  "--color-matrix-green-hover",
  "--color-dark-bg",
];
```

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS (no references yet; file compiles).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/sql-editor/theme/types.ts
git commit -m "feat(sql-editor): add theme model types"
```

---

### Task 2: Derivation functions (TDD)

**Files:**
- Create: `frontend/src/react/components/sql-editor/theme/derive.ts`
- Test: `frontend/src/react/components/sql-editor/theme/derive.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// frontend/src/react/components/sql-editor/theme/derive.test.ts
import { describe, expect, test } from "vitest";
import {
  buildMonacoTheme,
  monacoThemeName,
  resolveAdminTheme,
  themeToCssVars,
  validateTheme,
} from "./derive";
import type { SQLEditorTheme } from "./types";
import { SQL_EDITOR_THEME_TOKENS } from "./types";

// Minimal complete fixture (values arbitrary but every key present).
const fixture = (over: Partial<SQLEditorTheme> = {}): SQLEditorTheme => {
  const tokens = Object.fromEntries(
    SQL_EDITOR_THEME_TOKENS.map((k) => [k, "1 2 3"])
  ) as SQLEditorTheme["tokens"];
  return {
    id: "fixture",
    name: "Fixture",
    isDark: false,
    monacoBase: "vs",
    tokens,
    editor: {
      background: "#ffffff",
      selectionBackground: "#cccccc",
      cursor: "#000000",
      lineHighlight: "#eeeeee",
      gutterBackground: "#ffffff",
      lineNumber: "#999999",
      activeLineNumber: "#111111",
    },
    syntax: {
      comment: "#777777",
      keyword: "#0000ff",
      string: "#008000",
      number: "#098658",
      type: "#267f99",
      function: "#795e26",
      variable: "#001080",
      operator: "#000000",
      delimiter: "#000000",
      predefined: "#0000ff",
    },
    ...over,
  };
};

describe("themeToCssVars", () => {
  test("maps every chrome token to a --color-* CSS property", () => {
    const vars = themeToCssVars(fixture().tokens);
    expect(vars["--color-control"]).toBe("1 2 3");
    expect(Object.keys(vars)).toHaveLength(SQL_EDITOR_THEME_TOKENS.length);
  });
});

describe("monacoThemeName", () => {
  test("prefixes the theme id", () => {
    expect(monacoThemeName(fixture({ id: "monokai" }))).toBe("bb-monokai");
  });
});

describe("buildMonacoTheme", () => {
  test("maps editor colors and syntax onto Monaco theme data", () => {
    const data = buildMonacoTheme(fixture({ monacoBase: "vs-dark" }));
    expect(data.base).toBe("vs-dark");
    expect(data.inherit).toBe(true);
    expect(data.colors["editor.background"]).toBe("#ffffff");
    expect(data.colors["editor.selectionBackground"]).toBe("#cccccc");
    expect(data.colors["editorCursor.foreground"]).toBe("#000000");
    // syntax → rules; keyword present without the leading "#".
    const keyword = data.rules.find((r) => r.token === "keyword");
    expect(keyword?.foreground).toBe("0000ff");
  });
});

describe("resolveAdminTheme", () => {
  test("keeps a dark selected theme", () => {
    const dark = fixture({ id: "monokai", isDark: true });
    expect(resolveAdminTheme(dark).id).toBe("monokai");
  });
  test("falls back to the dark preset for a light selected theme", () => {
    const light = fixture({ id: "light", isDark: false });
    expect(resolveAdminTheme(light).id).toBe("dark");
  });
});

describe("validateTheme", () => {
  test("passes a complete theme", () => {
    expect(() => validateTheme(fixture())).not.toThrow();
  });
  test("throws when a chrome token is missing", () => {
    const bad = fixture();
    // @ts-expect-error intentionally drop a token
    delete bad.tokens["--color-accent"];
    expect(() => validateTheme(bad)).toThrow(/--color-accent/);
  });
  test("throws when a syntax key is missing", () => {
    const bad = fixture();
    // @ts-expect-error intentionally drop a syntax key
    delete bad.syntax.keyword;
    expect(() => validateTheme(bad)).toThrow(/keyword/);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/derive.test.ts`
Expected: FAIL — `derive.ts` does not exist / functions undefined.

- [ ] **Step 3: Write the implementation**

```ts
// frontend/src/react/components/sql-editor/theme/derive.ts
import type { CSSProperties } from "react";
import type { editor as Editor } from "monaco-editor";
import { PRESET_BY_ID } from "./presets";
import type {
  EditorChromeColors,
  SQLEditorTheme,
  SyntaxPalette,
} from "./types";
import { SQL_EDITOR_THEME_TOKENS } from "./types";

// Monaco wants hex without the leading "#" in rule foregrounds.
const bare = (hex: string): string => hex.replace(/^#/, "");

/** Chrome: tokens → inline CSS custom properties for a container's `style`. */
export function themeToCssVars(
  tokens: SQLEditorTheme["tokens"]
): CSSProperties {
  const vars: Record<string, string> = {};
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    vars[token] = tokens[token];
  }
  return vars as CSSProperties;
}

/** Monaco theme name registered via defineTheme + used by setTheme. */
export function monacoThemeName(theme: SQLEditorTheme): string {
  return `bb-${theme.id}`;
}

// Map the normalized syntax palette onto Monaco token scopes. SQL is the
// primary language; the generic scopes cover other languages Monaco loads.
const syntaxRules = (s: SyntaxPalette): Editor.ITokenThemeRule[] => {
  const rule = (token: string, hex: string): Editor.ITokenThemeRule => ({
    token,
    foreground: bare(hex),
  });
  return [
    rule("comment", s.comment),
    rule("keyword", s.keyword),
    rule("keyword.sql", s.keyword),
    rule("string", s.string),
    rule("string.sql", s.string),
    rule("number", s.number),
    rule("number.sql", s.number),
    rule("type", s.type),
    rule("type.sql", s.type),
    rule("identifier", s.variable),
    rule("predefined", s.predefined),
    rule("predefined.sql", s.predefined),
    rule("operator", s.operator),
    rule("operator.sql", s.operator),
    rule("delimiter", s.delimiter),
  ];
};

const editorColors = (e: EditorChromeColors): Editor.IColors => ({
  "editor.background": e.background,
  "editor.selectionBackground": e.selectionBackground,
  "editorCursor.foreground": e.cursor,
  "editor.lineHighlightBackground": e.lineHighlight,
  "editorGutter.background": e.gutterBackground,
  "editorLineNumber.foreground": e.lineNumber,
  "editorLineNumber.activeForeground": e.activeLineNumber,
  // Keep the existing "no word highlight box" behavior from bb/bb-dark.
  "editor.wordHighlightBackground": "#00000000",
  "editor.wordHighlightStrongBackground": "#00000000",
});

/** Monaco: whole theme → registered standalone theme data. */
export function buildMonacoTheme(
  theme: SQLEditorTheme
): Editor.IStandaloneThemeData {
  return {
    base: theme.monacoBase,
    inherit: true,
    rules: syntaxRules(theme.syntax),
    colors: editorColors(theme.editor),
  };
}

/** Admin foreground resolution: keep dark themes, else fall back to dark. */
export function resolveAdminTheme(selected: SQLEditorTheme): SQLEditorTheme {
  return selected.isDark ? selected : PRESET_BY_ID.dark;
}

/** Throws if any chrome token / editor key / syntax key is missing. */
export function validateTheme(theme: SQLEditorTheme): void {
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    if (typeof theme.tokens[token] !== "string") {
      throw new Error(`theme "${theme.id}" missing chrome token ${token}`);
    }
  }
  const editorKeys: (keyof EditorChromeColors)[] = [
    "background",
    "selectionBackground",
    "cursor",
    "lineHighlight",
    "gutterBackground",
    "lineNumber",
    "activeLineNumber",
  ];
  for (const key of editorKeys) {
    if (typeof theme.editor[key] !== "string") {
      throw new Error(`theme "${theme.id}" missing editor.${key}`);
    }
  }
  const syntaxKeys: (keyof SyntaxPalette)[] = [
    "comment",
    "keyword",
    "string",
    "number",
    "type",
    "function",
    "variable",
    "operator",
    "delimiter",
    "predefined",
  ];
  for (const key of syntaxKeys) {
    if (typeof theme.syntax[key] !== "string") {
      throw new Error(`theme "${theme.id}" missing syntax.${key}`);
    }
  }
}
```

> Note: `derive.ts` imports `PRESET_BY_ID` from `./presets` (Task 3). The
> `derive.test.ts` above does not, so it passes before presets exist **only if**
> the import resolves. To keep Task 2 self-contained, create a temporary stub
> `presets/index.ts` now: `export const PRESET_BY_ID = {} as Record<string, never>;`
> — Task 3 replaces it. (Alternatively do Task 2 + Task 3 together.)

- [ ] **Step 4: Add the temporary presets stub**

```ts
// frontend/src/react/components/sql-editor/theme/presets/index.ts (temporary)
import type { SQLEditorTheme } from "../types";
export const PRESET_BY_ID: Record<string, SQLEditorTheme> = {};
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/derive.test.ts`
Expected: PASS (all `describe` blocks green). `resolveAdminTheme` light-fallback test exercises `PRESET_BY_ID.dark`, which is `undefined` in the stub — its `.id` read throws. **Therefore include a local mock in the test** by re-importing: instead, the light-fallback assertion must run after Task 3. Mark that one test `test.skip` here and un-skip in Task 3 Step 4.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/sql-editor/theme/derive.ts frontend/src/react/components/sql-editor/theme/derive.test.ts frontend/src/react/components/sql-editor/theme/presets/index.ts
git commit -m "feat(sql-editor): add theme derivation functions"
```

---

### Task 3: Default presets (light, dark) + catalog

**Files:**
- Create: `frontend/src/react/components/sql-editor/theme/presets/light.ts`, `dark.ts`
- Replace: `frontend/src/react/components/sql-editor/theme/presets/index.ts`
- Test: `frontend/src/react/components/sql-editor/theme/presets/presets.test.ts`

- [ ] **Step 1: Write the `light` preset (values verbatim from `:root` in `tailwind.css`)**

```ts
// frontend/src/react/components/sql-editor/theme/presets/light.ts
import type { SQLEditorTheme } from "../types";

// Chrome tokens copied verbatim from :root (frontend/src/assets/css/tailwind.css)
// so "Default Light" renders byte-for-byte identical to today.
export const light: SQLEditorTheme = {
  id: "light",
  name: "sql-editor.theme.light",
  isDark: false,
  monacoBase: "vs",
  tokens: {
    "--color-control": "82 82 91",
    "--color-control-hover": "24 24 27",
    "--color-control-light": "113 113 122",
    "--color-control-light-hover": "82 82 91",
    "--color-control-bg": "243 244 246",
    "--color-control-bg-hover": "229 231 235",
    "--color-control-placeholder": "161 161 170",
    "--color-control-border": "209 213 219",
    "--color-accent": "79 70 229",
    "--color-accent-hover": "55 48 163",
    "--color-accent-disabled": "165 180 252",
    "--color-accent-text": "255 255 255",
    "--color-main": "24 24 27",
    "--color-main-hover": "63 63 70",
    "--color-main-text": "255 255 255",
    "--color-background": "255 255 255",
    "--color-block-border": "229 231 235",
    "--color-link-hover": "229 231 235",
    "--color-info": "37 99 235",
    "--color-info-hover": "29 78 216",
    "--color-warning": "245 158 11",
    "--color-warning-hover": "180 83 9",
    "--color-error": "220 38 38",
    "--color-error-hover": "185 28 28",
    "--color-success": "22 163 74",
    "--color-success-hover": "21 128 61",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  // Reproduces the old `bb` Monaco theme (transparent bg, accent cursor).
  editor: {
    background: "#ffffff",
    selectionBackground: "#add6ff",
    cursor: "#4f46e5",
    lineHighlight: "#f3f4f6",
    gutterBackground: "#ffffff",
    lineNumber: "#a1a1aa",
    activeLineNumber: "#18181b",
  },
  // VS light defaults (inherited from base "vs"; listed for completeness).
  syntax: {
    comment: "#008000",
    keyword: "#0000ff",
    string: "#a31515",
    number: "#098658",
    type: "#267f99",
    function: "#795e26",
    variable: "#001080",
    operator: "#000000",
    delimiter: "#000000",
    predefined: "#0000ff",
  },
};
```

- [ ] **Step 2: Write the `dark` preset (Default Dark — spec §2 starting values)**

```ts
// frontend/src/react/components/sql-editor/theme/presets/dark.ts
import type { SQLEditorTheme } from "../types";

export const dark: SQLEditorTheme = {
  id: "dark",
  name: "sql-editor.theme.dark",
  isDark: true,
  monacoBase: "vs-dark",
  tokens: {
    "--color-control": "229 231 235",
    "--color-control-hover": "243 244 246",
    "--color-control-light": "209 213 219",
    "--color-control-light-hover": "229 231 235",
    "--color-control-bg": "55 65 81",
    "--color-control-bg-hover": "75 85 99",
    "--color-control-placeholder": "156 163 175",
    "--color-control-border": "113 113 122",
    "--color-accent": "129 140 248",
    "--color-accent-hover": "165 180 252",
    "--color-accent-disabled": "67 56 202",
    "--color-accent-text": "255 255 255",
    "--color-main": "244 244 245",
    "--color-main-hover": "212 212 216",
    "--color-main-text": "24 24 27",
    "--color-background": "30 30 30",
    "--color-block-border": "82 82 91",
    "--color-link-hover": "63 63 70",
    "--color-info": "96 165 250",
    "--color-info-hover": "147 197 253",
    "--color-warning": "251 191 36",
    "--color-warning-hover": "253 224 71",
    "--color-error": "248 113 113",
    "--color-error-hover": "252 165 165",
    "--color-success": "74 222 128",
    "--color-success-hover": "134 239 172",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  editor: {
    background: "#1e1e1e",
    selectionBackground: "#264f78",
    cursor: "#818cf8",
    lineHighlight: "#2a2a2a",
    gutterBackground: "#1e1e1e",
    lineNumber: "#858585",
    activeLineNumber: "#c6c6c6",
  },
  // VS dark defaults.
  syntax: {
    comment: "#6a9955",
    keyword: "#569cd6",
    string: "#ce9178",
    number: "#b5cea8",
    type: "#4ec9b0",
    function: "#dcdcaa",
    variable: "#9cdcfe",
    operator: "#d4d4d4",
    delimiter: "#d4d4d4",
    predefined: "#569cd6",
  },
};
```

- [ ] **Step 3: Replace the catalog index**

```ts
// frontend/src/react/components/sql-editor/theme/presets/index.ts
import type { SQLEditorTheme } from "../types";
import { dark } from "./dark";
import { light } from "./light";
import { monokai } from "./monokai";
import { nord } from "./nord";
import { solarizedDark } from "./solarized-dark";
import { solarizedLight } from "./solarized-light";

// Display order in the switcher.
export const PRESETS: SQLEditorTheme[] = [
  light,
  dark,
  solarizedLight,
  solarizedDark,
  monokai,
  nord,
];

export const PRESET_BY_ID: Record<string, SQLEditorTheme> = Object.fromEntries(
  PRESETS.map((p) => [p.id, p])
);

export const DEFAULT_THEME_ID = "light";

export const resolveThemeId = (id: string | undefined): SQLEditorTheme =>
  (id && PRESET_BY_ID[id]) || PRESET_BY_ID[DEFAULT_THEME_ID];
```

> This imports the four named presets (Task 4). Create Task 4 files **before**
> running tests, or temporarily comment the four imports + array entries and
> restore them in Task 4. Recommended: do Task 3 + Task 4 in one sitting.

- [ ] **Step 4: Write the catalog test + un-skip the light-fallback test from Task 2**

```ts
// frontend/src/react/components/sql-editor/theme/presets/presets.test.ts
import { describe, expect, test } from "vitest";
import { validateTheme } from "../derive";
import { DEFAULT_THEME_ID, PRESETS, PRESET_BY_ID } from "./index";

describe("theme presets catalog", () => {
  test("every preset is structurally complete", () => {
    for (const preset of PRESETS) {
      expect(() => validateTheme(preset)).not.toThrow();
    }
  });
  test("ids are unique", () => {
    const ids = PRESETS.map((p) => p.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
  test("contains the six v1 themes in order", () => {
    expect(PRESETS.map((p) => p.id)).toEqual([
      "light",
      "dark",
      "solarized-light",
      "solarized-dark",
      "monokai",
      "nord",
    ]);
  });
  test("default theme exists and is light", () => {
    expect(DEFAULT_THEME_ID).toBe("light");
    expect(PRESET_BY_ID[DEFAULT_THEME_ID]).toBeDefined();
  });
});
```

Then in `derive.test.ts` remove the `.skip` added in Task 2 Step 5 on the
`resolveAdminTheme` light-fallback test.

- [ ] **Step 5: Run tests to verify they pass**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/`
Expected: PASS (catalog complete; `validateTheme` green across all six — requires Task 4 done).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/sql-editor/theme/presets/
git commit -m "feat(sql-editor): add light + dark theme presets and catalog"
```

---

### Task 4: Named presets (Solarized Light/Dark, Monokai, Nord)

Author each from its canonical palette. Each object has the exact same shape as
`light` (Task 3). Use the chrome-token mapping convention below so every named
theme is filled deterministically.

**Chrome-token mapping convention** (same as light/dark): `background` = base bg;
`control-bg` = nearest surface; `control` / `control-light` / `control-placeholder`
= foreground ramp (bright→dim); `control-border` / `block-border` = subtle border;
`main` = brightest fg; `accent` = primary accent; status = theme blue/yellow/red/green;
`matrix-green*` = `0 204 0` / `136 255 136` (constant); `dark-bg` = `30 30 30` (constant).

**Files:** create `solarized-light.ts`, `solarized-dark.ts`, `monokai.ts`, `nord.ts`.

- [ ] **Step 1: Monokai** (`bg #272822`, `fg #f8f8f2`, comment `#75715e`, pink `#f92672`, orange `#fd971f`, yellow `#e6db74`, green `#a6e22e`, cyan `#66d9ef`, purple `#ae81ff`, selection `#49483e`, line `#3e3d32`)

```ts
// frontend/src/react/components/sql-editor/theme/presets/monokai.ts
import type { SQLEditorTheme } from "../types";

export const monokai: SQLEditorTheme = {
  id: "monokai",
  name: "Monokai",
  isDark: true,
  monacoBase: "vs-dark",
  tokens: {
    "--color-control": "248 248 242",
    "--color-control-hover": "255 255 255",
    "--color-control-light": "204 204 196",
    "--color-control-light-hover": "248 248 242",
    "--color-control-bg": "62 61 50",
    "--color-control-bg-hover": "73 72 62",
    "--color-control-placeholder": "117 113 94",
    "--color-control-border": "73 72 62",
    "--color-accent": "166 226 46",
    "--color-accent-hover": "182 234 80",
    "--color-accent-disabled": "117 113 94",
    "--color-accent-text": "39 40 34",
    "--color-main": "248 248 242",
    "--color-main-hover": "255 255 255",
    "--color-main-text": "39 40 34",
    "--color-background": "39 40 34",
    "--color-block-border": "73 72 62",
    "--color-link-hover": "73 72 62",
    "--color-info": "102 217 239",
    "--color-info-hover": "140 226 244",
    "--color-warning": "253 151 31",
    "--color-warning-hover": "253 175 80",
    "--color-error": "249 38 114",
    "--color-error-hover": "251 90 147",
    "--color-success": "166 226 46",
    "--color-success-hover": "182 234 80",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  editor: {
    background: "#272822",
    selectionBackground: "#49483e",
    cursor: "#f8f8f0",
    lineHighlight: "#3e3d32",
    gutterBackground: "#272822",
    lineNumber: "#90908a",
    activeLineNumber: "#f8f8f2",
  },
  syntax: {
    comment: "#75715e",
    keyword: "#f92672",
    string: "#e6db74",
    number: "#ae81ff",
    type: "#66d9ef",
    function: "#a6e22e",
    variable: "#f8f8f2",
    operator: "#f92672",
    delimiter: "#f8f8f2",
    predefined: "#66d9ef",
  },
};
```

- [ ] **Step 2: Nord** (Polar Night `#2e3440/#3b4252/#434c5e/#4c566a`, Snow Storm `#d8dee9/#e5e9f0/#eceff4`, Frost `#8fbcbb/#88c0d0/#81a1c1/#5e81ac`, Aurora `#bf616a/#d08770/#ebcb8b/#a3be8c/#b48ead`)

```ts
// frontend/src/react/components/sql-editor/theme/presets/nord.ts
import type { SQLEditorTheme } from "../types";

export const nord: SQLEditorTheme = {
  id: "nord",
  name: "Nord",
  isDark: true,
  monacoBase: "vs-dark",
  tokens: {
    "--color-control": "216 222 233",
    "--color-control-hover": "236 239 244",
    "--color-control-light": "180 192 211",
    "--color-control-light-hover": "216 222 233",
    "--color-control-bg": "59 66 82",
    "--color-control-bg-hover": "67 76 94",
    "--color-control-placeholder": "118 128 146",
    "--color-control-border": "76 86 106",
    "--color-accent": "136 192 208",
    "--color-accent-hover": "143 188 187",
    "--color-accent-disabled": "76 86 106",
    "--color-accent-text": "46 52 64",
    "--color-main": "236 239 244",
    "--color-main-hover": "255 255 255",
    "--color-main-text": "46 52 64",
    "--color-background": "46 52 64",
    "--color-block-border": "67 76 94",
    "--color-link-hover": "67 76 94",
    "--color-info": "129 161 193",
    "--color-info-hover": "136 192 208",
    "--color-warning": "235 203 139",
    "--color-warning-hover": "242 218 165",
    "--color-error": "191 97 106",
    "--color-error-hover": "208 135 112",
    "--color-success": "163 190 140",
    "--color-success-hover": "181 205 161",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  editor: {
    background: "#2e3440",
    selectionBackground: "#434c5e",
    cursor: "#d8dee9",
    lineHighlight: "#3b4252",
    gutterBackground: "#2e3440",
    lineNumber: "#4c566a",
    activeLineNumber: "#d8dee9",
  },
  syntax: {
    comment: "#616e88",
    keyword: "#81a1c1",
    string: "#a3be8c",
    number: "#b48ead",
    type: "#8fbcbb",
    function: "#88c0d0",
    variable: "#d8dee9",
    operator: "#81a1c1",
    delimiter: "#eceff4",
    predefined: "#8fbcbb",
  },
};
```

- [ ] **Step 3: Solarized Dark** (base03 `#002b36`, base02 `#073642`, base01 `#586e75`, base00 `#657b83`, base0 `#839496`, base1 `#93a1a1`, accents yellow `#b58900`/orange `#cb4b16`/red `#dc322f`/magenta `#d33682`/violet `#6c71c4`/blue `#268bd2`/cyan `#2aa198`/green `#859900`)

```ts
// frontend/src/react/components/sql-editor/theme/presets/solarized-dark.ts
import type { SQLEditorTheme } from "../types";

export const solarizedDark: SQLEditorTheme = {
  id: "solarized-dark",
  name: "Solarized Dark",
  isDark: true,
  monacoBase: "vs-dark",
  tokens: {
    "--color-control": "131 148 150",
    "--color-control-hover": "147 161 161",
    "--color-control-light": "101 123 131",
    "--color-control-light-hover": "131 148 150",
    "--color-control-bg": "7 54 66",
    "--color-control-bg-hover": "20 70 84",
    "--color-control-placeholder": "88 110 117",
    "--color-control-border": "88 110 117",
    "--color-accent": "38 139 210",
    "--color-accent-hover": "42 161 152",
    "--color-accent-disabled": "88 110 117",
    "--color-accent-text": "253 246 227",
    "--color-main": "147 161 161",
    "--color-main-hover": "238 232 213",
    "--color-main-text": "0 43 54",
    "--color-background": "0 43 54",
    "--color-block-border": "7 54 66",
    "--color-link-hover": "7 54 66",
    "--color-info": "38 139 210",
    "--color-info-hover": "42 161 152",
    "--color-warning": "181 137 0",
    "--color-warning-hover": "203 75 22",
    "--color-error": "220 50 47",
    "--color-error-hover": "211 54 130",
    "--color-success": "133 153 0",
    "--color-success-hover": "159 181 0",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  editor: {
    background: "#002b36",
    selectionBackground: "#073642",
    cursor: "#839496",
    lineHighlight: "#073642",
    gutterBackground: "#002b36",
    lineNumber: "#586e75",
    activeLineNumber: "#93a1a1",
  },
  syntax: {
    comment: "#586e75",
    keyword: "#859900",
    string: "#2aa198",
    number: "#d33682",
    type: "#b58900",
    function: "#268bd2",
    variable: "#839496",
    operator: "#859900",
    delimiter: "#93a1a1",
    predefined: "#cb4b16",
  },
};
```

- [ ] **Step 4: Solarized Light** (same accents; light base: base3 `#fdf6e3`, base2 `#eee8d5`, base1 `#93a1a1`, base00 `#657b83`, base01 `#586e75`, base03 `#002b36`)

```ts
// frontend/src/react/components/sql-editor/theme/presets/solarized-light.ts
import type { SQLEditorTheme } from "../types";

export const solarizedLight: SQLEditorTheme = {
  id: "solarized-light",
  name: "Solarized Light",
  isDark: false,
  monacoBase: "vs",
  tokens: {
    "--color-control": "101 123 131",
    "--color-control-hover": "88 110 117",
    "--color-control-light": "131 148 150",
    "--color-control-light-hover": "101 123 131",
    "--color-control-bg": "238 232 213",
    "--color-control-bg-hover": "224 218 199",
    "--color-control-placeholder": "147 161 161",
    "--color-control-border": "147 161 161",
    "--color-accent": "38 139 210",
    "--color-accent-hover": "42 161 152",
    "--color-accent-disabled": "147 161 161",
    "--color-accent-text": "253 246 227",
    "--color-main": "0 43 54",
    "--color-main-hover": "7 54 66",
    "--color-main-text": "253 246 227",
    "--color-background": "253 246 227",
    "--color-block-border": "238 232 213",
    "--color-link-hover": "238 232 213",
    "--color-info": "38 139 210",
    "--color-info-hover": "29 110 168",
    "--color-warning": "181 137 0",
    "--color-warning-hover": "203 75 22",
    "--color-error": "220 50 47",
    "--color-error-hover": "211 54 130",
    "--color-success": "133 153 0",
    "--color-success-hover": "107 123 0",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "30 30 30",
  },
  editor: {
    background: "#fdf6e3",
    selectionBackground: "#eee8d5",
    cursor: "#657b83",
    lineHighlight: "#eee8d5",
    gutterBackground: "#fdf6e3",
    lineNumber: "#93a1a1",
    activeLineNumber: "#586e75",
  },
  syntax: {
    comment: "#93a1a1",
    keyword: "#859900",
    string: "#2aa198",
    number: "#d33682",
    type: "#b58900",
    function: "#268bd2",
    variable: "#657b83",
    operator: "#859900",
    delimiter: "#586e75",
    predefined: "#cb4b16",
  },
};
```

- [ ] **Step 5: Run the catalog test (now all six resolve)**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/`
Expected: PASS — `validateTheme` green for all six; order test green.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/sql-editor/theme/presets/
git commit -m "feat(sql-editor): add solarized, monokai, nord presets"
```

---

## Phase 2 — Store & persistence

### Task 5: Add the storage key

**Files:**
- Modify: `frontend/src/utils/storage-keys.ts` (SQL Editor section)

- [ ] **Step 1: Add the builder** (alongside `storageKeySqlEditorLastProject`)

```ts
// Workspace-scoped like other SQL Editor prefs.
export const storageKeySqlEditorTheme = (scope: string) =>
  withScope("bb.sql-editor.theme", scope);
```

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/utils/storage-keys.ts
git commit -m "feat(sql-editor): add theme storage key"
```

---

### Task 6: Store `themeId` + persistence (TDD)

**Files:**
- Modify: `frontend/src/react/stores/sqlEditor/editor.ts`
- Test: `frontend/src/react/stores/sqlEditor/editor.test.ts`

- [ ] **Step 1: Write the failing test** (append to `editor.test.ts`; mirror existing localStorage mock there)

```ts
import { storageKeySqlEditorTheme } from "@/utils/storage-keys";
import { DEFAULT_THEME_ID } from "@/react/components/sql-editor/theme/presets";

describe("theme persistence", () => {
  test("setThemeId updates state and persists", () => {
    getSQLEditorEditorState().setThemeId("monokai");
    expect(useSQLEditorEditorStore.getState().themeId).toBe("monokai");
    expect(storage.get(storageKeySqlEditorTheme(""))).toBe(
      JSON.stringify("monokai")
    );
  });

  test("invalid persisted id falls back to the default", () => {
    storage.set(storageKeySqlEditorTheme(""), JSON.stringify("does-not-exist"));
    // readThemeId() runs at store init; assert the reader's fallback.
    const raw = storage.get(storageKeySqlEditorTheme(""));
    expect(raw).toBeDefined();
    // The store reset in beforeEach sets themeId to DEFAULT; setThemeId guards.
    getSQLEditorEditorState().setThemeId("does-not-exist");
    expect(useSQLEditorEditorStore.getState().themeId).toBe(DEFAULT_THEME_ID);
  });
});
```

Also extend the `beforeEach` store reset object in `editor.test.ts` to include
`themeId: "light",`.

- [ ] **Step 2: Run to verify it fails**

Run: `pnpm --dir frontend test -- run src/react/stores/sqlEditor/editor.test.ts`
Expected: FAIL — `setThemeId` / `themeId` undefined.

- [ ] **Step 3: Implement in `editor.ts`**

Add the import:

```ts
import { storageKeySqlEditorTheme } from "@/utils/storage-keys";
import {
  DEFAULT_THEME_ID,
  PRESET_BY_ID,
} from "@/react/components/sql-editor/theme/presets";
```

Add to the `SQLEditorEditorState` interface:

```ts
  themeId: string;
  setThemeId: (id: string) => void;
```

Add reader helpers (near `lastProjectKey` / `readProject`):

```ts
const themeKey = () => {
  const state = useAppStore.getState();
  const isSaaS =
    typeof state?.isSaaSMode === "function" ? state.isSaaSMode() : false;
  return storageKeySqlEditorTheme(
    workspaceCacheScope(isSaaS, state?.currentUser?.workspace ?? "")
  );
};

const readThemeId = () =>
  safeRead<string>(
    themeKey(),
    (v) => (typeof v === "string" && PRESET_BY_ID[v] ? v : undefined),
    DEFAULT_THEME_ID
  );
```

Add to the `create(...)` initial object:

```ts
    themeId: readThemeId(),
```

Add the action (next to `setResultRowsLimit`):

```ts
    setThemeId(id) {
      // Ignore ids not in the catalog so a stale/garbage value can't break theming.
      const next = PRESET_BY_ID[id] ? id : DEFAULT_THEME_ID;
      set((s) => {
        s.themeId = next;
      });
      safeWrite(themeKey(), next);
    },
```

- [ ] **Step 4: Run to verify it passes**

Run: `pnpm --dir frontend test -- run src/react/stores/sqlEditor/editor.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/stores/sqlEditor/editor.ts frontend/src/react/stores/sqlEditor/editor.test.ts
git commit -m "feat(sql-editor): persist selected theme id in editor store"
```

---

## Phase 3 — Monaco integration

### Task 7: Register generated Monaco themes

**Files:**
- Modify: `frontend/src/react/components/monaco/core.ts`
- Delete: `frontend/src/react/components/monaco/themes/bb.ts`, `bb-dark.ts`

- [ ] **Step 1: Rewrite `initializeTheme` to register every preset**

Replace the `getBBTheme`/`getBBDarkTheme` imports and `initializeTheme` body:

```ts
import { PRESETS } from "@/react/components/sql-editor/theme/presets";
import {
  buildMonacoTheme,
  monacoThemeName,
} from "@/react/components/sql-editor/theme/derive";

const initializeTheme = () => {
  if (state.themeInitialized) return;
  state.themeInitialized = true;
  if (!monacoModule) return;
  for (const preset of PRESETS) {
    const name = monacoThemeName(preset);
    try {
      monacoModule.editor.defineTheme(name, buildMonacoTheme(preset));
      state.registeredThemes.add(name);
    } catch {
      // The vscode theme-service override owns themes in some runtime modes;
      // an un-registered theme falls back to `vs` via getResolvedTheme.
    }
  }
};
```

- [ ] **Step 2: Update the default editor theme** — change `theme: "bb"` to `theme: "bb-light"` in both `defaultEditorOptions()` and `defaultDiffEditorOptions()`. Update `getResolvedTheme`'s no-arg fallback:

```ts
export const getResolvedTheme = (requested?: string): string => {
  const fallback = "vs";
  if (!requested) {
    return state.registeredThemes.has("bb-light") ? "bb-light" : fallback;
  }
  return state.registeredThemes.has(requested) ? requested : fallback;
};
```

- [ ] **Step 3: Delete the obsolete theme files**

```bash
git rm frontend/src/react/components/monaco/themes/bb.ts frontend/src/react/components/monaco/themes/bb-dark.ts
```

- [ ] **Step 4: Type-check + targeted test**

Run: `pnpm --dir frontend type-check`
Expected: PASS (no remaining `getBBTheme` references — grep to confirm: `grep -rn "getBBTheme\|getBBDarkTheme\|\"bb\"" frontend/src/react/components/monaco` returns nothing except updated lines).

- [ ] **Step 5: Commit**

```bash
git add -A frontend/src/react/components/monaco
git commit -m "feat(monaco): register generated themes from SQL editor presets"
```

---

### Task 8: Global Monaco theme controller

The controller sets the one global Monaco theme from the foreground panel. It is
the **only** caller of `setTheme` in the SQL Editor. `CompactSQLEditor` and the
default options no longer pin a theme.

**Files:**
- Create: `frontend/src/react/components/sql-editor/theme/useMonacoThemeController.ts`
- Modify: `frontend/src/react/components/sql-editor/TerminalPanel/CompactSQLEditor.tsx`

- [ ] **Step 1: Write the controller hook**

```ts
// frontend/src/react/components/sql-editor/theme/useMonacoThemeController.ts
import { useEffect } from "react";
import {
  getResolvedTheme,
  loadMonacoEditor,
} from "@/react/components/monaco/core";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
import { monacoThemeName, resolveAdminTheme } from "./derive";
import { resolveThemeId } from "./presets";

/**
 * Drives the single global Monaco theme from the foreground code panel:
 * worksheet → selected theme, admin → resolveAdminTheme(selected). Mount once
 * at the SQL Editor root. Monaco's theme is global, so no other component may
 * call setTheme.
 */
export function useMonacoThemeController() {
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  const mode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );

  useEffect(() => {
    const selected = resolveThemeId(themeId);
    const active =
      mode === "ADMIN" ? resolveAdminTheme(selected) : selected;
    let cancelled = false;
    void loadMonacoEditor().then((monaco) => {
      if (cancelled) return;
      monaco.editor.setTheme(getResolvedTheme(monacoThemeName(active)));
    });
    return () => {
      cancelled = true;
    };
  }, [themeId, mode]);
}
```

- [ ] **Step 2: Stop `CompactSQLEditor` pinning `vs-dark`** — in `CompactSQLEditor.tsx` `editorOptions` (~line 323), delete the `theme: "vs-dark",` line. The controller now owns the theme.

- [ ] **Step 3: Stop `MonacoEditor` re-pinning a per-instance theme** — in `MonacoEditor.tsx` (~line 346), the post-construction `monaco.editor.setTheme(getResolvedTheme(optionsRef.current?.theme))` would fight the controller when `options.theme` is unset. Guard it so it only applies when an explicit theme option was passed:

```ts
        if (optionsRef.current?.theme) {
          monaco.editor.setTheme(
            getResolvedTheme(optionsRef.current.theme)
          );
        }
```

(Default SQL Editor editors pass no `theme` now, so the controller's global theme stands.)

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Commit** (controller is mounted in Task 10)

```bash
git add frontend/src/react/components/sql-editor/theme/useMonacoThemeController.ts frontend/src/react/components/sql-editor/TerminalPanel/CompactSQLEditor.tsx frontend/src/react/components/monaco/MonacoEditor.tsx
git commit -m "feat(sql-editor): add global Monaco theme controller"
```

---

## Phase 4 — Theme scope & wiring

### Task 9: `SQLEditorThemeScope` context + component (TDD)

**Files:**
- Create: `frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.tsx`
- Test: `frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { PRESET_BY_ID } from "./presets";
import {
  SQLEditorThemeScope,
  useSQLEditorTheme,
} from "./SQLEditorThemeScope";

function Probe() {
  const theme = useSQLEditorTheme();
  return <span data-testid="id">{theme.id}</span>;
}

describe("SQLEditorThemeScope", () => {
  test("writes the theme tokens as inline CSS vars on its container", () => {
    const { container } = render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.monokai}>
        <div>child</div>
      </SQLEditorThemeScope>
    );
    const el = container.firstChild as HTMLElement;
    expect(el.style.getPropertyValue("--color-background")).toBe("39 40 34");
  });

  test("provides the theme via context", () => {
    render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.nord}>
        <Probe />
      </SQLEditorThemeScope>
    );
    expect(screen.getByTestId("id").textContent).toBe("nord");
  });

  test("nested scope overrides the parent for context consumers", () => {
    render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.light}>
        <SQLEditorThemeScope theme={PRESET_BY_ID.dark}>
          <Probe />
        </SQLEditorThemeScope>
      </SQLEditorThemeScope>
    );
    expect(screen.getByTestId("id").textContent).toBe("dark");
  });
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

```tsx
// frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.tsx
import {
  createContext,
  useContext,
  useMemo,
  type ReactNode,
} from "react";
import { cn } from "@/react/lib/utils";
import { themeToCssVars } from "./derive";
import { PRESET_BY_ID } from "./presets";
import type { SQLEditorTheme } from "./types";

const SQLEditorThemeContext = createContext<SQLEditorTheme>(PRESET_BY_ID.light);

export const useSQLEditorTheme = (): SQLEditorTheme =>
  useContext(SQLEditorThemeContext);

type Props = {
  theme: SQLEditorTheme;
  children: ReactNode;
  className?: string;
  // Render as a bare wrapper (default) or pass `display: contents` so the scope
  // adds no box (used where an extra div would break layout).
  asContents?: boolean;
};

/**
 * Provides `theme` via context AND writes its chrome tokens as inline CSS
 * custom properties so descendant Tailwind classes (text-control, bg-*, …)
 * re-theme via cascade. Nest it to override a subtree (admin terminal, portals).
 */
export function SQLEditorThemeScope({
  theme,
  children,
  className,
  asContents = false,
}: Props) {
  const style = useMemo(() => themeToCssVars(theme.tokens), [theme]);
  return (
    <SQLEditorThemeContext.Provider value={theme}>
      <div
        className={cn(asContents && "contents", className)}
        style={style}
      >
        {children}
      </div>
    </SQLEditorThemeContext.Provider>
  );
}
```

> Note: with `display: contents` the element generates no box, but inline CSS
> custom properties on it **still cascade** to children — so `asContents` scopes
> tokens without affecting layout. Use it for the root and portal wrappers where
> an extra flex/block box would disturb sizing.

- [ ] **Step 4: Run to verify it passes**

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.tsx frontend/src/react/components/sql-editor/theme/SQLEditorThemeScope.test.tsx
git commit -m "feat(sql-editor): add SQLEditorThemeScope context + container"
```

---

### Task 10: Mount the root scope + Monaco controller

**Files:**
- Modify: `frontend/src/react/components/sql-editor/SQLEditorLayout.tsx`

- [ ] **Step 1: Wrap the layout body in the root scope and mount the controller**

In `SQLEditorLayout.tsx`, add imports and a tiny inner component so the
controller (which reads stores) runs inside the scope:

```tsx
import { SQLEditorThemeScope } from "./theme/SQLEditorThemeScope";
import { useMonacoThemeController } from "./theme/useMonacoThemeController";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { resolveThemeId } from "./theme/presets";

function SQLEditorThemeRoot({ children }: { children: React.ReactNode }) {
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  useMonacoThemeController();
  return (
    <SQLEditorThemeScope theme={resolveThemeId(themeId)} asContents>
      {children}
    </SQLEditorThemeScope>
  );
}
```

Wrap the existing `RequestDrawerHost` + `SQLEditorRouteShell` subtree
(`SQLEditorLayout.tsx:62`) so the scope sits above both the route content and the
drawer host:

```tsx
        <SQLEditorThemeRoot>
          <RequestDrawerHost>
            <SQLEditorRouteShell />
          </RequestDrawerHost>
        </SQLEditorThemeRoot>
```

- [ ] **Step 2: Type-check + run dev to smoke-test**

Run: `pnpm --dir frontend type-check`
Then `pnpm --dir frontend dev`, open the SQL Editor: it still renders normally on the default `light` theme (no visual change yet — switcher not added).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/sql-editor/SQLEditorLayout.tsx
git commit -m "feat(sql-editor): mount root theme scope + Monaco controller"
```

---

### Task 11: Nested admin scope + drop `ResultView` `dark` prop

**Files:**
- Modify: `frontend/src/react/components/sql-editor/TerminalPanel/TerminalPanel.tsx`
- Modify: `frontend/src/react/components/sql-editor/ResultView/ResultView.tsx`

- [ ] **Step 1: Wrap `TerminalPanel`'s root in the nested admin scope**

In `TerminalPanel.tsx`, add imports:

```tsx
import { SQLEditorThemeScope } from "../theme/SQLEditorThemeScope";
import { resolveAdminTheme } from "../theme/derive";
import { resolveThemeId } from "../theme/presets";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
```

Inside the component, compute the admin theme and wrap the outermost `<div>`
(currently `<div className="… bg-dark-bg">` at line 206). Replace `bg-dark-bg`
on the outer and inner containers with the token-driven `bg-background`:

```tsx
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  const adminTheme = resolveAdminTheme(resolveThemeId(themeId));

  return (
    <SQLEditorThemeScope theme={adminTheme} className="h-full w-full">
      <div className="flex h-full w-full flex-col justify-start items-stretch overflow-hidden bg-background">
        {/* …existing children, inner bg-dark-bg → bg-background… */}
      </div>
    </SQLEditorThemeScope>
  );
```

- [ ] **Step 2: Remove the hardcoded `dark` prop from the `ResultView` call** in `TerminalPanel.tsx:236` (delete the `dark` prop entirely).

- [ ] **Step 3: Drop the `dark` prop from `ResultView`**

In `ResultView.tsx`: remove `dark?: boolean` from `ResultViewProps`, remove
`dark = false` from the destructure, and replace the wrapper className
(`ResultView.tsx:237`):

```tsx
import { useSQLEditorTheme } from "../theme/SQLEditorThemeScope";
// …
  const { isDark } = useSQLEditorTheme();
// …
    <div
      className={cn(
        "relative flex flex-col justify-start items-start pb-1 overflow-y-auto h-full w-full",
        isDark && "bg-background"
      )}
    >
```

(The `dark` Tailwind-variant class is removed; descendants re-theme via the
scope tokens. `isDark` remains available for any boolean-only need.)

- [ ] **Step 4: Type-check + grep for stragglers**

Run: `pnpm --dir frontend type-check`
Then `grep -rn "dark={" frontend/src/react/components/sql-editor/ResultView` → expect nothing. `grep -rn "dark=" frontend/src/react/components/sql-editor/TerminalPanel` → expect nothing.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/sql-editor/TerminalPanel/TerminalPanel.tsx frontend/src/react/components/sql-editor/ResultView/ResultView.tsx
git commit -m "feat(sql-editor): nest admin theme scope, drop ResultView dark prop"
```

---

### Task 12: Portal scopes (overlay + request drawer)

**Files:**
- Modify: `frontend/src/react/components/sql-editor/SQLEditorHomePage.tsx`
- Modify: `frontend/src/react/components/sql-editor/AccessGrantRequestDrawer.tsx`

- [ ] **Step 1: Wrap portaled overlay children** — in `SQLEditorHomePage.tsx`, each `createPortal(<…>, getLayerRoot("overlay"))` (lines ~160 and ~191) puts DOM outside the chrome subtree, so CSS vars don't cascade. Wrap each portal's element tree in a scope reading the current theme:

```tsx
import { SQLEditorThemeScope } from "./theme/SQLEditorThemeScope";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { resolveThemeId } from "./theme/presets";
// inside the component:
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  const theme = resolveThemeId(themeId);
// for each portal payload:
  createPortal(
    <SQLEditorThemeScope theme={theme} asContents>
      {/* existing FAB / sidebar overlay markup */}
    </SQLEditorThemeScope>,
    getLayerRoot("overlay")
  )
```

- [ ] **Step 2: Wrap the drawer's `Sheet` content** — in `AccessGrantRequestDrawer.tsx`, the `Sheet` portals to the overlay layer. Wrap its content body in a scope so the drawer chrome (form controls, labels) re-themes. Its embedded `<MonacoEditor>` (line 262) gets no `theme` prop — it shares the global Monaco theme set by the controller:

```tsx
import { SQLEditorThemeScope } from "./theme/SQLEditorThemeScope";
import { useSQLEditorTheme } from "./theme/SQLEditorThemeScope";
// wrap the sheet content's root element:
  <SQLEditorThemeScope theme={useSQLEditorTheme()} asContents>
    {/* existing drawer body */}
  </SQLEditorThemeScope>
```

(`useSQLEditorTheme()` works because the drawer is under the layout-level root
scope's React context even though its DOM is portaled.)

- [ ] **Step 3: Confirm the drawer Monaco passes no theme** — grep `AccessGrantRequestDrawer.tsx` around line 262; if the `<MonacoEditor>` `options`/props set `theme`, remove it.

- [ ] **Step 4: Type-check + visual smoke** (after the switcher exists you re-verify; for now just type-check)

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/components/sql-editor/SQLEditorHomePage.tsx frontend/src/react/components/sql-editor/AccessGrantRequestDrawer.tsx
git commit -m "feat(sql-editor): apply theme scope to portaled overlays + drawer"
```

---

## Phase 5 — Hardcoded color migration

Each task replaces raw palette classes / `dark:` variants / inline colors with
semantic tokens (per spec §5). Replacement convention:

| Hardcoded | Token class |
|---|---|
| `bg-gray-100/200`, `bg-gray-50` | `bg-control-bg` |
| `bg-white` (surface) | `bg-background` |
| `bg-white/NN` (overlay) | `bg-background/NN` |
| `bg-black/NN` (overlay) | `bg-overlay/NN` |
| `bg-gray-900` (tooltip) | `bg-main` |
| `text-gray-400` | `text-control-placeholder` |
| `text-gray-500/600` | `text-control-light` |
| `text-white` (on dark surface) | `text-main-text` or `text-control` per context |
| `border-gray-100/200/400`, `border-zinc-500/600` | `border-block-border` (cells) / `border-control-border` (controls) |

All `dark:`-prefixed variants are **deleted** (the scope tokens carry dark values).

### Task 13: Result-view color migration

**Files (exact lines from spec §5):**
- `ResultView/VirtualDataTable.tsx` — 271, 275, 311, 313, 397, 436, 496
- `ResultView/VirtualDataBlock.tsx` — 147, 153, 166, 237
- `ResultView/SelectionCopyTooltips.tsx` — 56, 62, 75, 153
- `ResultView/SingleResultView.tsx` — 602, 617
- `ResultPanel/DatabaseQueryContext.tsx` — 75, 93

- [ ] **Step 1: VirtualDataTable** — delete all `dark:` variants on the listed lines; keep the light token classes already present (`bg-control-bg`, `text-control-light`, `border-block-border`). Where a line had only `dark:` (e.g. `dark:border-zinc-500`), ensure a base token class exists (`border-block-border`). Example, line 271:

```tsx
// before: className="... border-block-border dark:border-zinc-500"
// after:  className="... border-block-border"
```

- [ ] **Step 2: VirtualDataBlock** — same treatment on lines 147/153/166/237 (`dark:bg-gray-700`→ drop; base stays `bg-control-bg`).

- [ ] **Step 3: SelectionCopyTooltips** — lines 56/62/75/153: drop `dark:*`; replace `dark:bg-dark-bg` with base `bg-background`, `dark:border-zinc-600` with base `border-control-border`.

- [ ] **Step 4: SingleResultView** — lines 602/617: drop the `dark:bg-gray-700 dark:text-gray-100 dark:border-zinc-600 dark:hover:bg-gray-600` cluster; base becomes `bg-control-bg text-control border-control-border hover:bg-control-bg-hover`.

- [ ] **Step 5: DatabaseQueryContext** — lines 75/93: `bg-white/80 dark:bg-black/80` → `bg-background/80`.

- [ ] **Step 6: Type-check + grep**

Run: `pnpm --dir frontend type-check`
Then `grep -rn "dark:\|-gray-\|-zinc-" frontend/src/react/components/sql-editor/ResultView frontend/src/react/components/sql-editor/ResultPanel/DatabaseQueryContext.tsx` → expect zero.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/components/sql-editor/ResultView frontend/src/react/components/sql-editor/ResultPanel/DatabaseQueryContext.tsx
git commit -m "refactor(sql-editor): route result-view colors through theme tokens"
```

---

### Task 14: Standard-mode color migration

**Files (spec §5):**
- `ConnectionPane/DatabaseHoverPanel/DatabaseHoverPanel.tsx` — 97, 114, 124, 133, 142, 149
- `ConnectionPane/ConnectionPane.tsx` — 680
- `SQLEditorHomePage.tsx` — 164, 194
- `SchemaPane/FlatTableList.tsx` — 169, 174, 206, 209, 218, 221, 227, 228, 237, 240
- `SchemaPane/TreeNode/icons.tsx` — 43, 54, 59, 71, 87, 91, 126
- `SchemaPane/HoverPanel/HoverPanel.tsx` — 142; `HoverPanel/InfoItem.tsx` — 22
- `SharePopoverBody.tsx` — 182, 197, 198, 236, 247
- `Panels/common/EllipsisCell.tsx` — 70

- [ ] **Step 1: Hover panels** — `DatabaseHoverPanel.tsx` & `SchemaPane/HoverPanel`: `border-gray-100`→`border-block-border`, `bg-white`→`bg-background`, `text-gray-500`→`text-control-light`, `bg-gray-200/75`→`bg-control-bg`.

- [ ] **Step 2: Loading overlays** — `ConnectionPane.tsx:680` `bg-white/75`→`bg-background/75`; `SQLEditorHomePage.tsx:164` `bg-white`→`bg-background`, `:194` `bg-black/40`→`bg-overlay/40`.

- [ ] **Step 3: Schema pane** — `FlatTableList.tsx` & `TreeNode/icons.tsx`: `text-gray-400`→`text-control-placeholder`, `text-gray-500/600`→`text-control-light`, `bg-gray-50`→`bg-control-bg`, `border-gray-200`→`border-block-border`.

- [ ] **Step 4: Share + tooltip** — `SharePopoverBody.tsx`: `border-gray-200`→`border-control-border`, `text-gray-400`→`text-control-placeholder`, `hover:bg-gray-200`/`bg-gray-200`→`hover:bg-control-bg-hover`/`bg-control-bg`, `bg-white`→`bg-background`. `EllipsisCell.tsx:70` `bg-gray-900 text-white`→`bg-main text-main-text`.

- [ ] **Step 5: Type-check + grep**

Run: `pnpm --dir frontend type-check`
Then grep the touched files for `-gray-`, `-zinc-`, `bg-white`, `bg-black` → expect only the documented dynamic cases (none in these files).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/sql-editor
git commit -m "refactor(sql-editor): route standard-mode colors through theme tokens"
```

---

### Task 15: Terminal residual colors

**Files:**
- `TerminalPanel/TerminalPanel.tsx` — 246, 251

- [ ] **Step 1: Overlays/text** — line 246 `bg-black/20`→`bg-overlay/20`; line 251 `text-gray-400`→`text-control-placeholder`. (The `bg-dark-bg`→`bg-background` swap was done in Task 11.)

- [ ] **Step 2: Type-check + grep**

Run: `pnpm --dir frontend type-check`
Then `grep -rn "bg-dark-bg\|-gray-\|bg-black" frontend/src/react/components/sql-editor/TerminalPanel` → expect zero.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/components/sql-editor/TerminalPanel
git commit -m "refactor(sql-editor): route terminal residual colors through tokens"
```

---

## Phase 6 — Switcher UI

### Task 16: i18n keys

**Files:**
- Modify: `frontend/src/locales/en-US.json`, `zh-CN.json`, `ja-JP.json`, `es-ES.json`, `vi-VN.json`

- [ ] **Step 1: Add keys under the `sql-editor` object** (light/dark labels + the section label). Named-palette labels are literal proper nouns (no key). en-US:

```json
"theme": {
  "self": "Theme",
  "light": "Default Light",
  "dark": "Default Dark"
}
```

Add the locale-appropriate translations for the other four files (translate
`self`, `light`, `dark`; keep the same JSON shape). Example zh-CN:

```json
"theme": { "self": "主题", "light": "默认浅色", "dark": "默认深色" }
```

(ja-JP: `"テーマ" / "デフォルト（ライト）" / "デフォルト（ダーク）"`; es-ES:
`"Tema" / "Claro predeterminado" / "Oscuro predeterminado"`; vi-VN: `"Giao diện" /
"Sáng mặc định" / "Tối mặc định"`.)

- [ ] **Step 2: Verify no empty objects + valid JSON**

Run: `pnpm --dir frontend type-check` (i18n types) and `node -e "require('./frontend/src/locales/en-US.json')"` for each file (valid JSON).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/locales
git commit -m "i18n(sql-editor): add theme switcher labels"
```

---

### Task 17: `ThemeSelect` + mount in `EditorAction`

**Files:**
- Create: `frontend/src/react/components/sql-editor/ThemeSelect.tsx`
- Modify: `frontend/src/react/components/sql-editor/EditorAction.tsx`

- [ ] **Step 1: Build `ThemeSelect`** using the shared Base UI `Select` (confirm the exported subcomponent names in `ui/select.tsx`: `Select`, `SelectTrigger`, `SelectValue`, `SelectContent`/`SelectPositioner`+`SelectPopup`, `SelectItem`). Drive options off `PRESETS`:

```tsx
// frontend/src/react/components/sql-editor/ThemeSelect.tsx
import { useTranslation } from "react-i18next";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import { PRESETS } from "./theme/presets";

// Named-palette ids whose label is a literal proper noun (not an i18n key).
const LITERAL_NAME = new Set([
  "solarized-light",
  "solarized-dark",
  "monokai",
  "nord",
]);

export function ThemeSelect() {
  const { t } = useTranslation();
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  const label = (id: string, name: string) =>
    LITERAL_NAME.has(id) ? name : t(name);

  return (
    <Select
      value={themeId}
      onValueChange={(id: string) =>
        getSQLEditorEditorState().setThemeId(id)
      }
    >
      <SelectTrigger size="sm" aria-label={t("sql-editor.theme.self")}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {PRESETS.map((p) => (
          <SelectItem key={p.id} value={p.id}>
            {label(p.id, p.name)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
```

> Match the exact Base UI `Select` API used elsewhere (e.g. `MaxRowCountSelect`
> or another `ui/select` consumer). If the wrapper uses
> `onValueChange`/`value` vs `value`/`onChange`, follow that file. Confirm
> with: `grep -rln "from \"@/react/components/ui/select\"" frontend/src` and read
> one consumer.

- [ ] **Step 2: Mount in `EditorAction.tsx`** `action-right` group (always visible in both modes), before `ChooserGroup`:

```tsx
import { ThemeSelect } from "./ThemeSelect";
// …
      <div className="action-right gap-x-2 flex …">
        <ThemeSelect />
        <ChooserGroup />
        <OpenAIButton … />
      </div>
```

- [ ] **Step 3: Type-check + test the component renders all six**

Add `frontend/src/react/components/sql-editor/ThemeSelect.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { ThemeSelect } from "./ThemeSelect";

describe("ThemeSelect", () => {
  test("renders the trigger with the current theme", () => {
    render(<ThemeSelect />);
    // Trigger shows the selected value; default store themeId is "light".
    expect(screen.getByLabelText(/theme/i)).toBeInTheDocument();
  });
});
```

Run: `pnpm --dir frontend test -- run src/react/components/sql-editor/ThemeSelect.test.tsx`
Expected: PASS. (If i18n isn't set up in the test env, wrap with the project's test i18n provider — copy from a sibling `*.test.tsx`.)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/components/sql-editor/ThemeSelect.tsx frontend/src/react/components/sql-editor/ThemeSelect.test.tsx frontend/src/react/components/sql-editor/EditorAction.tsx
git commit -m "feat(sql-editor): add theme switcher to editor toolbar"
```

---

## Phase 7 — Verification

### Task 18: Full verification sweep

- [ ] **Step 1: No hardcoded colors remain** (excluding documented dynamic/animation cases + admin terminal already migrated)

Run:
```bash
grep -rn "dark:\|-gray-\|-zinc-\|-slate-\|-neutral-\|bg-white\|bg-black\|text-white\|text-black" \
  frontend/src/react/components/sql-editor | \
  grep -v -E "Label.tsx|TabItem.tsx|BatchQuerySelect.tsx|AccessGrantItem.tsx"
```
Expected: empty (any hit is an un-migrated island — fix it).

- [ ] **Step 2: Only the controller calls `setTheme`**

Run: `grep -rn "setTheme(" frontend/src/react/components/sql-editor frontend/src/react/components/monaco`
Expected: occurrences only in `useMonacoThemeController.ts` and `MonacoEditor.tsx` (guarded) — no per-component `setTheme`.

- [ ] **Step 3: Gates**

Run, in order, until clean:
```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```
Expected: all PASS.

- [ ] **Step 4: Manual visual check** (`pnpm --dir frontend dev`)

For each theme in the switcher:
- Chrome (sidebars, tabs, toolbar, schema pane), Monaco surface, and result grid all match the theme — no light islands on dark themes.
- **Default Light** is visually identical to `main`.
- **Admin mode:** enter admin with a **light** theme selected → terminal subtree (editor + result) is dark, surrounding chrome stays light. With a **dark/named** theme selected → terminal matches it.
- **Portals:** mobile FAB + sidebar overlay themed; open the access-request drawer → its chrome themed; its Monaco shares the active global theme without flicker, including when opened over an admin terminal.
- Reload the page → the selected theme persists (localStorage).

- [ ] **Step 5: Final commit** (if `fix`/`check` reformatted anything)

```bash
git add -A frontend
git commit -m "chore(sql-editor): lint/format pass for theme feature"
```

---

## Self-Review notes

- **Spec coverage:** model (T1–2), 6 presets (T3–4), store+persistence (T5–6), Monaco generation (T7) + global controller resolving the drawer conflict (T8), scope component (T9), root/admin/portal scopes (T10–12), full color migration incl. `dark`-prop removal (T11, T13–15), switcher in always-visible `EditorAction` (T16–17), verification incl. portal + drawer-Monaco checks (T18). All spec sections map to a task.
- **Known authoring follow-up:** named-theme palettes (T4) and the `dark` chrome values (T3) are spec-flagged as visually tunable — T18 Step 4 is where they get the visual pass.
- **API confirmations the implementer must make before coding the UI:** exact Base UI `Select` subcomponent names/props (`ui/select.tsx`) in T17, and the test-env i18n provider pattern from a sibling `*.test.tsx`.
