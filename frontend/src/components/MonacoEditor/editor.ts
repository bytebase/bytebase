import { buildWorkerDefinition } from "monaco-editor-workers";
import "monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution.js";
import "monaco-editor/esm/vs/basic-languages/redis/redis.contribution.js";
import "monaco-editor/esm/vs/basic-languages/sql/sql.contribution.js";
import "monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution.js";
import "monaco-editor/esm/vs/editor/editor.all.js";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import "monaco-editor/esm/vs/editor/standalone/browser/standaloneCodeEditorService";
import "monaco-editor/esm/vs/language/typescript/monaco.contribution.js";
import { defer } from "@/utils";
import { initializeMonacoServices } from "./services";
import { getBBTheme } from "./themes/bb";
import { getBBDarkTheme } from "./themes/bb-dark";
import type { MonacoModule } from "./types";

export default monaco as MonacoModule;

const state = {
  themeInitialized: false,
};

buildWorkerDefinition("/monaco-workers", import.meta.url, true);

const MonacoEditorReadyDefer = defer<void>();

export const MonacoEditorReady = MonacoEditorReadyDefer.promise;

const initializeTheme = () => {
  if (state.themeInitialized) return;

  monaco.editor.defineTheme("bb", getBBTheme());
  monaco.editor.defineTheme("bb-dark", getBBDarkTheme());

  state.themeInitialized = true;
};

const initialize = async () => {
  await initializeMonacoServices();

  try {
    const { initializeLSPClient } = await import("./lsp-client");
    await initializeLSPClient();
  } catch (err) {
    console.error("[MonacoEditor] initialize", err);
  }
  initializeTheme();
};

export const createMonacoEditor = async (config: {
  container: HTMLElement;
  options?: monaco.editor.IStandaloneEditorConstructionOptions;
}): Promise<monaco.editor.IStandaloneCodeEditor> => {
  await initialize();

  // create monaco editor
  const editor = monaco.editor.create(config.container, {
    ...defaultEditorOptions(),
    ...config.options,
  });

  MonacoEditorReadyDefer.resolve(undefined);

  return editor;
};

export const createMonacoDiffEditor = async (config: {
  container: HTMLElement;
  options?: monaco.editor.IStandaloneDiffEditorConstructionOptions;
}): Promise<monaco.editor.IStandaloneDiffEditor> => {
  await initialize();

  // create monaco editor
  const editor = monaco.editor.createDiffEditor(config.container, {
    ...defaultDiffEditorOptions(),
    ...config.options,
  });

  MonacoEditorReadyDefer.resolve();

  return editor;
};

export const defaultEditorOptions =
  (): monaco.editor.IStandaloneEditorConstructionOptions => {
    return {
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      renderValidationDecorations: "on",
      // Learn more: https://github.com/microsoft/monaco-editor/issues/4270
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
    };
  };

export const defaultDiffEditorOptions =
  (): monaco.editor.IStandaloneDiffEditorConstructionOptions => {
    return {
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      enableSplitViewResizing: false,
      // Learn more: https://github.com/microsoft/monaco-editor/issues/4270
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
