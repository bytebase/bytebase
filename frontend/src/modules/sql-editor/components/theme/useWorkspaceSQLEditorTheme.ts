import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { Color } from "@/types/proto-es/google/type/color_pb";
import { colorToHex } from "@/utils";
import { validateTheme } from "./derive";
import { PRESET_BY_ID, resolveThemeId } from "./presets";
import { type SQLEditorTheme, type SQLEditorThemeToken } from "./types";

// Mirrors WorkspaceProfileSetting's two theme fields (protojson camelCase).
export interface WorkspaceThemeInput {
  sqlEditorThemeId?: string;
  sqlEditorCustomTheme?: {
    id: string;
    name: string;
    monacoBase: string;
    tokens: Record<string, Color>;
  };
}

/** Pure resolution: custom (matching id) → preset (by id) → light. */
export function resolveWorkspaceTheme(
  input: WorkspaceThemeInput
): SQLEditorTheme {
  const id = input.sqlEditorThemeId ?? "";
  const custom = input.sqlEditorCustomTheme;
  // `&& custom` keeps the type narrowing the optional chain alone doesn't give.
  if (custom?.id === id && custom) {
    const theme: SQLEditorTheme = {
      id: custom.id,
      name: custom.name,
      monacoBase: custom.monacoBase,
      tokens: Object.fromEntries(
        Object.entries(custom.tokens).map(([key, color]) => [
          key,
          colorToHex(color),
        ])
      ) as Record<SQLEditorThemeToken, string>,
    };
    try {
      validateTheme(theme);
      return theme;
    } catch {
      return PRESET_BY_ID.light;
    }
  }
  return resolveThemeId(id);
}

export function useWorkspaceSQLEditorTheme(): SQLEditorTheme {
  const profile = useAppStore((s) => s.getWorkspaceProfile());
  // Memoize on the (stable) profile reference so the resolved theme keeps a
  // STABLE identity across renders. `useMonacoThemeController` decides whether
  // to call the global `monaco.editor.setTheme` by comparing the theme BY
  // REFERENCE (`prev === theme`); a fresh object every render — which
  // `resolveWorkspaceTheme` returns for a custom theme — would make it fire
  // mid-construction, racing the codingame theme service and dropping the
  // editor to its read-only <pre> fallback.
  return useMemo(
    () => resolveWorkspaceTheme(profile as WorkspaceThemeInput),
    [profile]
  );
}
