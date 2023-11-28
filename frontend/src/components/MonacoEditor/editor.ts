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
import { shouldUseNewLSP } from "./dev";
import { initializeMonacoServices } from "./services";
import { getBBTheme } from "./themes/bb";
import { getBBDarkTheme } from "./themes/bb-dark";
import type { MonacoModule } from "./types";

export default monaco as MonacoModule;

const state = {
  themeInitialized: false,
};

buildWorkerDefinition(
  new URL("@public/monaco-workers", import.meta.url).href,
  import.meta.url,
  true
);

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
  if (shouldUseNewLSP()) {
    const { initializeLSPClient } = await import("./lsp-client");
    await initializeLSPClient();
  } else {
    const { useLanguageClient } = await import("@/plugins/sql-lsp/client");
    const { start } = useLanguageClient();
    start();
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
      theme: "bb",
      tabSize: 2,
      insertSpaces: true,
      autoClosingQuotes: "always",
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
      padding: {
        top: 8,
        bottom: 8,
      },
      renderLineHighlight: "none",
      codeLens: false,
      scrollbar: {
        alwaysConsumeMouseWheel: false,
      },
    };
  };

export const defaultDiffEditorOptions =
  (): monaco.editor.IStandaloneDiffEditorConstructionOptions => {
    return {
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      enableSplitViewResizing: false,
      renderValidationDecorations: "on",
      theme: "bb",
      autoClosingQuotes: "always",
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
      padding: {
        top: 8,
        bottom: 8,
      },
      renderLineHighlight: "none",
      codeLens: false,
      scrollbar: {
        alwaysConsumeMouseWheel: false,
      },
    };
  };
