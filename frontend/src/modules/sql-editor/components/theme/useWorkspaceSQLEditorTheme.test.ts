import { describe, expect, test } from "vitest";
import { hexToColor } from "@/utils";
import { deriveThemeFromAnchors } from "./derive";
import { PRESET_BY_ID } from "./presets";
import type { SQLEditorTheme } from "./types";
import { resolveWorkspaceTheme } from "./useWorkspaceSQLEditorTheme";

const toStoredTheme = (theme: SQLEditorTheme) => ({
  ...theme,
  tokens: Object.fromEntries(
    Object.entries(theme.tokens).map(([key, value]) => [key, hexToColor(value)])
  ),
});

const customDef = {
  ...toStoredTheme(
    deriveThemeFromAnchors(
      {
        background: "#1e1e1e",
        text: "#f4f4f5",
        accent: "#6366f1",
        border: "#3f3f46",
      },
      "Brand"
    )
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
