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
  test("prefixes the theme id", () => {
    expect(monacoThemeName(fixture({ id: "monokai" }))).toBe("bb-monokai");
  });
});

describe("buildMonacoTheme", () => {
  test("carries the base and the word-highlight reset, no rules", () => {
    const data = buildMonacoTheme(fixture({ monacoBase: "vs-dark" }));
    expect(data.base).toBe("vs-dark");
    expect(data.inherit).toBe(true);
    expect(data.rules).toEqual([]);
    expect(data.colors["editor.wordHighlightBackground"]).toBe("#00000000");
  });
});

describe("resolveAdminTheme", () => {
  test("keeps a dark selected theme", () => {
    const dark = fixture({ id: "monokai", monacoBase: "vs-dark" });
    expect(resolveAdminTheme(dark).id).toBe("monokai");
  });
  test("falls back to the dark preset for a light selected theme", () => {
    const light = fixture({ id: "light", monacoBase: "vs" });
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
});
