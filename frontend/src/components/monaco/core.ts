import type * as MonacoType from "monaco-editor";
import type { Language } from "@/types";
import { defer } from "@/utils";
import { getAvailableEditorThemes } from "./editorThemes";
import { initializeMonacoServices } from "./services";

let monacoModule: typeof MonacoType | undefined;
const monacoLoadDefer = defer<typeof MonacoType>();
const monacoEditorReadyDefer = defer<void>();

export const MonacoEditorReady = monacoEditorReadyDefer.promise;

const state = {
  themeInitialized: false,
  registeredThemes: new Set<string>(["vs", "vs-dark", "hc-black", "hc-light"]),
  // The light/dark fallback base for each known theme id. If `setTheme` is
  // called with an id that isn't registered, `getResolvedTheme` falls back to
  // the theme's OWN type (dark → `vs-dark`, not the light `vs`).
  themeBase: new Map<string, "vs" | "vs-dark">(),
};

// Add the color themes the VSCode theme service actually has registered to the
// allowlist (so `getResolvedTheme` lets `setTheme` apply them) and record each
// one's light/dark fallback. The standalone built-ins above are always present.
const registerEditorThemes = async () => {
  if (state.themeInitialized) return;
  state.themeInitialized = true;
  const themes = await getAvailableEditorThemes();
  for (const theme of themes) {
    state.registeredThemes.add(theme.id);
    state.themeBase.set(theme.id, theme.type === "dark" ? "vs-dark" : "vs");
  }
};

/**
 * Returns the requested theme if it's known to be registered with
 * Monaco, otherwise falls back to the theme's own light/dark base (or `vs`).
 *
 * Use this anywhere `monaco.editor.setTheme(...)` is called from
 * application code — calling `setTheme` with an unregistered theme
 * name is a silent no-op, which leaves the global theme stuck on
 * whatever was last applied (and Monaco's theme is global, not
 * per-instance, so a stale `vs-dark` from a recently-disposed
 * terminal editor can bleed into a freshly-mounted worksheet editor).
 */
export const getResolvedTheme = (requested = "vs"): string => {
  if (state.registeredThemes.has(requested)) return requested;
  return state.themeBase.get(requested) ?? "vs";
};

const initialize = async () => {
  await initializeMonacoServices();
  await loadMonacoEditor();
  await registerEditorThemes();
};

export const loadMonacoEditor = async (): Promise<typeof MonacoType> => {
  if (monacoModule) {
    return monacoModule;
  }

  monacoModule = await import("monaco-editor");
  // Monaco binds Cmd/Ctrl+L to `expandLineSelection` and preventDefault()s it,
  // stealing the browser's address-bar shortcut while the editor is focused.
  // Unbind the chord (the "-" prefix removes the binding) so the keypress falls
  // through to the browser; the command stays reachable via the command palette.
  monacoModule.editor.addKeybindingRule({
    keybinding: monacoModule.KeyMod.CtrlCmd | monacoModule.KeyCode.KeyL,
    command: "-expandLineSelection",
  });
  monacoLoadDefer.resolve(monacoModule);
  return monacoModule;
};

export const getMonacoEditor = async (): Promise<typeof MonacoType> => {
  return monacoLoadDefer.promise;
};

export const isMonacoLoaded = (): boolean => {
  return monacoModule !== undefined;
};

export const createMonacoEditor = async (config: {
  container: HTMLElement;
  options?: MonacoType.editor.IStandaloneEditorConstructionOptions;
}): Promise<MonacoType.editor.IStandaloneCodeEditor> => {
  await initialize();
  const monaco = await loadMonacoEditor();
  const baseOptions = {
    editContext: false,
    experimentalEditContextEnabled: false,
  } as const;

  const editor = monaco.editor.create(config.container, {
    ...(baseOptions as MonacoType.editor.IStandaloneEditorConstructionOptions),
    ...defaultEditorOptions(),
    ...config.options,
  });

  editor.getContribution("editor.contrib.readOnlyMessageController")?.dispose();
  monacoEditorReadyDefer.resolve();
  return editor;
};

export const createMonacoDiffEditor = async (config: {
  container: HTMLElement;
  options?: MonacoType.editor.IStandaloneDiffEditorConstructionOptions;
}): Promise<MonacoType.editor.IStandaloneDiffEditor> => {
  await initialize();
  const monaco = await loadMonacoEditor();
  const baseOptions = {
    editContext: false,
    experimentalEditContextEnabled: false,
  } as const;

  const editor = monaco.editor.createDiffEditor(config.container, {
    ...(baseOptions as MonacoType.editor.IStandaloneDiffEditorConstructionOptions),
    ...defaultDiffEditorOptions(),
    ...config.options,
  });

  editor
    .getModifiedEditor()
    .getContribution("editor.contrib.readOnlyMessageController")
    ?.dispose();
  monacoEditorReadyDefer.resolve();
  return editor;
};

export const setMonacoModelLanguage = async (
  model: MonacoType.editor.ITextModel,
  language: Language
): Promise<void> => {
  const monaco = await loadMonacoEditor();
  monaco.editor.setModelLanguage(model, language);
};

export const defaultEditorOptions =
  (): MonacoType.editor.IStandaloneEditorConstructionOptions => {
    return {
      renderValidationDecorations: "on",
      accessibilitySupport: "off",
      theme: "vs",
      tabSize: 2,
      insertSpaces: true,
      autoClosingQuotes: "never",
      detectIndentation: false,
      folding: false,
      automaticLayout: true,
      minimap: {
        enabled: false,
      },
      wordWrap: "on",
      fixedOverflowWidgets: true,
      fontSize: 14,
      lineHeight: 24,
      scrollBeyondLastLine: false,
      suggestFontSize: 12,
      padding: {
        top: 8,
        bottom: 8,
      },
      renderLineHighlight: "none",
      codeLens: false,
      scrollbar: {
        alwaysConsumeMouseWheel: false,
      },
      inlineSuggest: {
        showToolbar: "never",
      },
      wordBasedSuggestions: "currentDocument",
      lineNumbers: "on",
      cursorStyle: "line",
      glyphMargin: false,
    };
  };

export const defaultDiffEditorOptions =
  (): MonacoType.editor.IStandaloneDiffEditorConstructionOptions => {
    return {
      enableSplitViewResizing: false,
      accessibilitySupport: "off",
      renderValidationDecorations: "on",
      theme: "vs",
      autoClosingQuotes: "never",
      folding: false,
      automaticLayout: true,
      minimap: {
        enabled: false,
      },
      wordWrap: "off",
      fixedOverflowWidgets: true,
      fontSize: 14,
      lineHeight: 24,
      scrollBeyondLastLine: false,
      padding: {
        top: 8,
        bottom: 8,
      },
      renderLineHighlight: "none",
      codeLens: false,
      scrollbar: {
        alwaysConsumeMouseWheel: false,
      },
      inlineSuggest: {
        showToolbar: "never",
      },
    };
  };
