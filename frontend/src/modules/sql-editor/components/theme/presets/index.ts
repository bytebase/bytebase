import type { SQLEditorTheme } from "../types";
import { dark } from "./dark";
import { light } from "./light";
import { solarizedDark } from "./solarized-dark";

// Display order in the switcher.
export const PRESETS: SQLEditorTheme[] = [light, dark, solarizedDark];

export const PRESET_BY_ID: Record<string, SQLEditorTheme> = Object.fromEntries(
  PRESETS.map((p) => [p.id, p])
);

export const DEFAULT_THEME_ID = "light";

export const resolveThemeId = (id: string | undefined): SQLEditorTheme =>
  (id && PRESET_BY_ID[id]) || PRESET_BY_ID[DEFAULT_THEME_ID];
