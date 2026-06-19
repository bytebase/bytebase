import { describe, expect, test } from "vitest";
import { deriveThemeFromAnchors } from "./derive";
import { PRESET_BY_ID } from "./presets";
import { resolveWorkspaceTheme } from "./useWorkspaceSQLEditorTheme";

const customDef = {
  ...deriveThemeFromAnchors(
    {
      background: "#1e1e1e",
      text: "#f4f4f5",
      accent: "#6366f1",
      border: "#3f3f46",
    },
    "Brand"
  ),
  id: "u1",
};

describe("resolveWorkspaceTheme", () => {
  test("custom matching id → custom", () => {
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "u1",
        sqlEditorCustomTheme: customDef,
      })?.id
    ).toBe("u1");
  });
  test("known preset id → catalog", () => {
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "dark",
        sqlEditorCustomTheme: undefined,
      })
    ).toBe(PRESET_BY_ID.dark);
  });
  test("unknown / empty id → light", () => {
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "",
        sqlEditorCustomTheme: undefined,
      })
    ).toBe(PRESET_BY_ID.light);
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "nope",
        sqlEditorCustomTheme: undefined,
      })
    ).toBe(PRESET_BY_ID.light);
  });
  test("custom id mismatch → falls through to preset/light", () => {
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "dark",
        sqlEditorCustomTheme: customDef,
      })
    ).toBe(PRESET_BY_ID.dark);
  });
  test("malformed custom (incomplete tokens) → light", () => {
    expect(
      resolveWorkspaceTheme({
        sqlEditorThemeId: "u1",
        sqlEditorCustomTheme: {
          id: "u1",
          name: "x",
          monacoBase: "vs",
          tokens: {},
        },
      })
    ).toBe(PRESET_BY_ID.light);
  });
});
