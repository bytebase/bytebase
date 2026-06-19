import { describe, expect, test } from "vitest";
import {
  deriveThemeFromAnchors,
  isDarkTheme,
  monacoThemeName,
  resolveAdminTheme,
  type ThemeAnchors,
  themeToAnchors,
  themeToCssVars,
  validateTheme,
} from "./derive";
import { PRESET_BY_ID } from "./presets";
import {
  DEFAULT_DARK_EDITOR_THEME,
  DEFAULT_LIGHT_EDITOR_THEME,
  SQL_EDITOR_THEME_TOKENS,
  type SQLEditorTheme,
} from "./types";

// Minimal complete fixture (token values arbitrary but every key present).
const fixture = (over: Partial<SQLEditorTheme> = {}): SQLEditorTheme => {
  const tokens = Object.fromEntries(
    SQL_EDITOR_THEME_TOKENS.map((k) => [k, "1 2 3"])
  ) as SQLEditorTheme["tokens"];
  return {
    id: "fixture",
    name: "Fixture",
    monacoBase: "vs",
    tokens,
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
  test("returns the theme's registered editor theme id (monacoBase)", () => {
    expect(monacoThemeName(fixture({ monacoBase: "Dark Modern" }))).toBe(
      "Dark Modern"
    );
  });
});

// Helpers that set a recognizably light/dark chrome background, since
// `isDarkTheme` (and thus resolveAdminTheme/themeColorScheme) keys off it.
const lightThemed = (over: Partial<SQLEditorTheme> = {}) =>
  fixture({
    ...over,
    tokens: {
      ...fixture().tokens,
      "--color-background": "255 255 255",
    } as SQLEditorTheme["tokens"],
  });
const darkThemed = (over: Partial<SQLEditorTheme> = {}) =>
  fixture({
    ...over,
    tokens: {
      ...fixture().tokens,
      "--color-background": "30 30 30",
    } as SQLEditorTheme["tokens"],
  });

describe("isDarkTheme", () => {
  test("keys off the chrome background luminance, not monacoBase", () => {
    // A light theme that nonetheless carries a dark editor theme id.
    expect(isDarkTheme(lightThemed({ monacoBase: "Dark Modern" }))).toBe(false);
    expect(isDarkTheme(darkThemed({ monacoBase: "vs" }))).toBe(true);
  });
});

describe("resolveAdminTheme", () => {
  test("keeps a dark selected theme", () => {
    expect(resolveAdminTheme(darkThemed({ id: "monokai" })).id).toBe("monokai");
  });
  test("falls back to the dark preset for a light selected theme", () => {
    expect(resolveAdminTheme(lightThemed({ id: "light" })).id).toBe("dark");
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
});

const lightAnchors: ThemeAnchors = {
  background: "#ffffff",
  text: "#18181b",
  accent: "#4f46e5",
  border: "#e5e7eb",
};
const darkAnchors: ThemeAnchors = {
  background: "#1e1e1e",
  text: "#f4f4f5",
  accent: "#6366f1",
  border: "#3f3f46",
};

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
    expect(t.tokens["--color-main"]).toBe("24 24 27");
    expect(t.tokens["--color-accent"]).toBe("79 70 229");
    expect(t.tokens["--color-block-border"]).toBe("229 231 235");
  });
  test("surface is derived from the background (not an anchor)", () => {
    // white bg nudged 6% toward near-black text → a light gray.
    expect(
      deriveThemeFromAnchors(lightAnchors, "B").tokens["--color-control-bg"]
    ).toBe("241 241 241");
  });
  test("monacoBase default by luminance", () => {
    expect(deriveThemeFromAnchors(lightAnchors, "L").monacoBase).toBe(
      DEFAULT_LIGHT_EDITOR_THEME
    );
    expect(deriveThemeFromAnchors(darkAnchors, "D").monacoBase).toBe(
      DEFAULT_DARK_EDITOR_THEME
    );
  });
  test("explicit monacoBase overrides luminance", () => {
    // Light anchors derive `vs`, but an explicit base wins (and vice versa).
    expect(
      deriveThemeFromAnchors(lightAnchors, "L", "vs-dark").monacoBase
    ).toBe("vs-dark");
    expect(deriveThemeFromAnchors(darkAnchors, "D", "vs").monacoBase).toBe(
      "vs"
    );
  });
  test("status + matrix fixed (anchor-independent)", () => {
    const a = deriveThemeFromAnchors(lightAnchors, "A").tokens;
    const b = deriveThemeFromAnchors(darkAnchors, "B").tokens;
    for (const k of [
      "--color-error",
      "--color-warning",
      "--color-success",
      "--color-matrix-green",
    ] as const) {
      expect(a[k]).toBe(b[k]);
      expect(a[k]).toBe(PRESET_BY_ID.light.tokens[k]);
    }
  });
  test("info mirrors the accent (not a fixed status color)", () => {
    const t = deriveThemeFromAnchors(lightAnchors, "A").tokens;
    expect(t["--color-info"]).toBe(t["--color-accent"]);
    expect(t["--color-info-hover"]).toBe(t["--color-accent-hover"]);
  });
});

describe("themeToAnchors", () => {
  test("round-trips", () => {
    expect(themeToAnchors(deriveThemeFromAnchors(darkAnchors, "D"))).toEqual(
      darkAnchors
    );
  });
});
