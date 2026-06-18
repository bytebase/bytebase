import { initializeMonacoServices } from "./services";

export interface EditorThemeOption {
  /** The id passed to `monaco.editor.setTheme`. */
  id: string;
  label: string;
  type: "light" | "dark";
}

/**
 * Always-available standalone built-ins — the safe fallback when enumeration
 * isn't possible (service not ready, API shape drift, etc.).
 */
export const BUILTIN_EDITOR_THEMES: EditorThemeOption[] = [
  { id: "vs", label: "Light", type: "light" },
  { id: "vs-dark", label: "Dark", type: "dark" },
];

/**
 * Enumerate the color themes actually registered with the VSCode theme service,
 * so the editor-theme picker is data-driven instead of relying on guessed ids
 * (the codingame service owns the registry; `defineTheme` ids it doesn't know
 * silently no-op). Defensive by design: any failure degrades to the built-ins,
 * and the dynamic import means a bad path can't break module load.
 */
export async function getAvailableEditorThemes(): Promise<EditorThemeOption[]> {
  try {
    await initializeMonacoServices();
    const { getService, IWorkbenchThemeService } = await import(
      "@codingame/monaco-vscode-api/services"
    );
    const themeService = await getService(IWorkbenchThemeService);
    const themes = await themeService.getColorThemes();
    const mapped = themes
      .map((theme): EditorThemeOption | null => {
        const id = theme.settingsId ?? theme.id;
        if (!id) return null;
        const isDark = String(theme.type ?? "")
          .toLowerCase()
          .includes("dark");
        return {
          id,
          label: theme.label ?? id,
          type: isDark ? "dark" : "light",
        };
      })
      .filter((theme): theme is EditorThemeOption => theme !== null);
    return mapped.length > 0 ? mapped : BUILTIN_EDITOR_THEMES;
  } catch (e) {
    console.warn(
      "[sql-editor-theme] editor-theme enumeration failed; using built-ins",
      e
    );
    return BUILTIN_EDITOR_THEMES;
  }
}
