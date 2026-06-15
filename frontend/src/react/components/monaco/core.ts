import type * as MonacoType from "monaco-editor";
import {
  buildMonacoTheme,
  monacoThemeName,
} from "@/react/components/sql-editor/theme/derive";
import { PRESETS } from "@/react/components/sql-editor/theme/presets";
import type { Language } from "@/types";
import { defer } from "@/utils";
import { initializeMonacoServices } from "./services";

let monacoModule: typeof MonacoType | undefined;
const monacoLoadDefer = defer<typeof MonacoType>();
const monacoEditorReadyDefer = defer<void>();

export const MonacoEditorReady = monacoEditorReadyDefer.promise;

const state = {
  themeInitialized: false,
  registeredThemes: new Set<string>(["vs", "vs-dark", "hc-black", "hc-light"]),
  // The Monaco base ("vs" | "vs-dark") for each generated theme name. When a
  // custom theme fails to register (the codingame VSCode theme service silently
  // ignores `defineTheme` in some runtime modes), `getResolvedTheme` falls back
  // to the theme's OWN base — so a dark theme falls back to `vs-dark`, not the
  // light `vs`. Recorded for every preset whether or not registration succeeds.
  themeBase: new Map<string, "vs" | "vs-dark">(),
};

const initializeTheme = () => {
  if (state.themeInitialized) return;
  state.themeInitialized = true;
  if (!monacoModule) return;
  for (const preset of PRESETS) {
    const name = monacoThemeName(preset);
    state.themeBase.set(name, preset.monacoBase);
    try {
      monacoModule.editor.defineTheme(name, buildMonacoTheme(preset));
      state.registeredThemes.add(name);
    } catch {
      // The vscode theme-service override owns themes in some runtime modes;
      // an un-registered theme falls back to its base via getResolvedTheme.
    }
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
export const getResolvedTheme = (requested = "bb-light"): string => {
  if (state.registeredThemes.has(requested)) return requested;
  // Custom theme not registered → use its own base (vs / vs-dark), so a dark
  // theme never falls back to the light `vs`.
  return state.themeBase.get(requested) ?? "vs";
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
      theme: "bb-light",
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
      theme: "bb-light",
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
