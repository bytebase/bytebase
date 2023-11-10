import { buildWorkerDefinition } from "monaco-editor-workers";
import "monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution.js";
import "monaco-editor/esm/vs/basic-languages/sql/sql.contribution.js";
import "monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution.js";
import "monaco-editor/esm/vs/editor/editor.all.js";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import "monaco-editor/esm/vs/language/typescript/monaco.contribution.js";
import { createConfiguredEditor } from "vscode/monaco";
import { defer } from "@/utils";
import { initializeMonacoServices } from "./services";
import { getBBTheme } from "./themes/bb";
import { getBBDarkTheme } from "./themes/bb-dark";

export default monaco;
export type MonacoModule = typeof monaco;

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

export const createMonacoEditor = async (config: {
  container: HTMLElement;
  options: monaco.editor.IStandaloneEditorConstructionOptions | undefined;
}): Promise<monaco.editor.IStandaloneCodeEditor> => {
  await initializeMonacoServices();

  initializeTheme();

  // create monaco editor
  const editor = createConfiguredEditor(config.container, {
    ...defaultEditorOptions(),
    ...config.options,
  });

  MonacoEditorReadyDefer.resolve(undefined);

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
