import { describe, expect, test } from "vitest";
import { validateTheme } from "../derive";
import {
  DEFAULT_THEME_ID,
  PRESET_BY_ID,
  PRESETS,
  resolveThemeId,
} from "./index";

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
  test("contains the presets in display order", () => {
    expect(PRESETS.map((p) => p.id)).toEqual([
      "light",
      "dark",
      "solarized-dark",
    ]);
  });
  test("default theme exists and is light", () => {
    expect(DEFAULT_THEME_ID).toBe("light");
    expect(PRESET_BY_ID[DEFAULT_THEME_ID]).toBeDefined();
  });
});

describe("resolveThemeId", () => {
  test("returns the matching preset for a known id", () => {
    expect(resolveThemeId("dark")).toBe(PRESET_BY_ID["dark"]);
  });
  test("falls back to the default for undefined", () => {
    expect(resolveThemeId(undefined)).toBe(PRESET_BY_ID[DEFAULT_THEME_ID]);
  });
  test("falls back to the default for an unknown id", () => {
    expect(resolveThemeId("nonexistent")).toBe(PRESET_BY_ID[DEFAULT_THEME_ID]);
  });
});
