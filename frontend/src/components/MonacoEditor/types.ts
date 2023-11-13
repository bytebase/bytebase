import type * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";

export type MonacoModule = typeof monaco;

export type IStandaloneCodeEditor = monaco.editor.IStandaloneCodeEditor;
export type IStandaloneDiffEditor = monaco.editor.IStandaloneDiffEditor;

export type AdviceOption = {
  severity: "ERROR" | "WARNING";
  message: string;
  source?: string;
  startLineNumber: number; // starts from 1
  startColumn: number; // starts from 1
  endLineNumber: number; // starts from 1
  endColumn: number; // starts from 1
};
