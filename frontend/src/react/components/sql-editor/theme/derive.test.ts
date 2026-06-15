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
    const keyword = data.rules.find((r) => r.token === "keyword");
    expect(keyword?.foreground).toBe("0000ff");
  });
  test("emits empty rules and only present colors when syntax is absent", () => {
    const t = fixture({
      syntax: undefined,
      editor: { background: "#fffffe00", cursor: "#4f46e5" },
    });
    const data = buildMonacoTheme(t);
    expect(data.rules).toEqual([]);
    expect(data.colors["editor.background"]).toBe("#fffffe00");
    expect(data.colors["editorCursor.foreground"]).toBe("#4f46e5");
    expect(data.colors["editor.selectionBackground"]).toBeUndefined();
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
  test("passes a theme with no syntax and a partial editor", () => {
    const minimal = fixture({
      syntax: undefined,
      editor: { background: "#000" },
    });
    expect(() => validateTheme(minimal)).not.toThrow();
  });
  test("throws when a syntax key is missing", () => {
    const bad = fixture();
    // @ts-expect-error intentionally drop a syntax key
    delete bad.syntax.keyword;
    expect(() => validateTheme(bad)).toThrow(/keyword/);
  });
});
