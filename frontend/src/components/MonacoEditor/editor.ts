import type * as MonacoType from "monaco-editor";
import { defer } from "@/utils";
import { loadMonacoEditor } from "./lazy-editor";
import { initializeMonacoServices } from "./services";

const state = {
  themeInitialized: false,
};

const MonacoEditorReadyDefer = defer<void>();

export const MonacoEditorReady = MonacoEditorReadyDefer.promise;

const initializeTheme = () => {
  if (state.themeInitialized) return;

  state.themeInitialized = true;
};

const initialize = async () => {
  await initializeMonacoServices();
  initializeTheme();
};

export const createMonacoEditor = async (config: {
  container: HTMLElement;
  options?: MonacoType.editor.IStandaloneEditorConstructionOptions;
}): Promise<MonacoType.editor.IStandaloneCodeEditor> => {
  await initialize();
  const monaco = await loadMonacoEditor();

  // Create monaco editor.
  const editor = monaco.editor.create(config.container, {
    ...{
      // https://github.com/microsoft/vscode/blob/main/src/vs/monaco.d.ts#L3824
      experimentalEditContextEnabled: false,
    },
    ...defaultEditorOptions(),
    ...config.options,
  });

  // Disable "Cannot edit in read-only editor" tooltip
  // https://github.com/microsoft/monaco-editor/discussions/4156
  editor.getContribution("editor.contrib.readOnlyMessageController")?.dispose();

  MonacoEditorReadyDefer.resolve(undefined);

  return editor;
};

export const createMonacoDiffEditor = async (config: {
  container: HTMLElement;
  options?: MonacoType.editor.IStandaloneDiffEditorConstructionOptions;
}): Promise<MonacoType.editor.IStandaloneDiffEditor> => {
  await initialize();
  const monaco = await loadMonacoEditor();

  // Create monaco diff editor.
  const editor = monaco.editor.createDiffEditor(config.container, {
    ...{
      // https://github.com/microsoft/vscode/blob/main/src/vs/monaco.d.ts#L3824
      experimentalEditContextEnabled: false,
    },
    ...defaultDiffEditorOptions(),
    ...config.options,
  });

  // Disable "Cannot edit in read-only editor" tooltip
  // https://github.com/microsoft/monaco-editor/discussions/4156
  editor
    .getModifiedEditor()
    .getContribution("editor.contrib.readOnlyMessageController")
    ?.dispose();

  MonacoEditorReadyDefer.resolve();

  return editor;
};

export const defaultEditorOptions =
  (): MonacoType.editor.IStandaloneEditorConstructionOptions => {
    return {
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      renderValidationDecorations: "on",
      // Learn more: https://github.com/microsoft/monaco-editor/issues/4270
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
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      enableSplitViewResizing: false,
      // Learn more: https://github.com/microsoft/monaco-editor/issues/4270
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
