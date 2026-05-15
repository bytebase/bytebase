import type * as MonacoType from "monaco-editor";
import type { Language } from "@/types";
import { defer } from "@/utils";
import { initializeMonacoServices } from "./services";
import { getBBTheme } from "./themes/bb";
import { getBBDarkTheme } from "./themes/bb-dark";

let monacoModule: typeof MonacoType | undefined;
const monacoLoadDefer = defer<typeof MonacoType>();
const monacoEditorReadyDefer = defer<void>();

export const MonacoEditorReady = monacoEditorReadyDefer.promise;

const state = {
  themeInitialized: false,
  registeredThemes: new Set<string>(["vs", "vs-dark", "hc-black", "hc-light"]),
};

const initializeTheme = () => {
  if (state.themeInitialized) return;
  state.themeInitialized = true;
  if (!monacoModule) return;
  try {
    monacoModule.editor.defineTheme("bb", getBBTheme());
    state.registeredThemes.add("bb");
    monacoModule.editor.defineTheme("bb-dark", getBBDarkTheme());
    state.registeredThemes.add("bb-dark");
  } catch {
    // The VSCode theme service override owns themes in some runtime modes.
    // Whichever theme failed stays out of `state.registeredThemes`, so
    // `getResolvedTheme` will fall back to the built-in `vs` for it.
  }
};

/**
 * Returns the requested theme if it's known to be registered with
 * Monaco, otherwise falls back to the always-available built-in `vs`.
 *
 * Use this anywhere `monaco.editor.setTheme(...)` is called from
 * application code — calling `setTheme` with an unregistered theme
 * name is a silent no-op, which leaves the global theme stuck on
 * whatever was last applied (and Monaco's theme is global, not
 * per-instance, so a stale `vs-dark` from a recently-disposed
 * terminal editor can bleed into a freshly-mounted worksheet editor).
 */
export const getResolvedTheme = (requested?: string): string => {
  const fallback = "vs";
  if (!requested) {
    return state.registeredThemes.has("bb") ? "bb" : fallback;
  }
  return state.registeredThemes.has(requested) ? requested : fallback;
};

const initialize = async () => {
  await initializeMonacoServices();
  await loadMonacoEditor();
  initializeTheme();
};

export const loadMonacoEditor = async (): Promise<typeof MonacoType> => {
  if (monacoModule) {
    return monacoModule;
  }

  monacoModule = await import("monaco-editor");
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
      theme: "bb",
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
      theme: "bb",
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
