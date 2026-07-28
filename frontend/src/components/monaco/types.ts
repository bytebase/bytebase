import type * as monaco from "monaco-editor";

export type MonacoModule = typeof monaco;

export type IStandaloneCodeEditor = monaco.editor.IStandaloneCodeEditor;
export type IStandaloneEditorConstructionOptions =
  monaco.editor.IStandaloneEditorConstructionOptions;
export type IStandaloneDiffEditor = monaco.editor.IStandaloneDiffEditor;
export type IStandaloneDiffEditorConstructionOptions =
  monaco.editor.IStandaloneDiffEditorConstructionOptions;

export type ITextModel = monaco.editor.ITextModel;
export type Selection = monaco.Selection;

export type AdviceOption = {
  severity: "ERROR" | "WARNING";
  message: string;
  source?: string;
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
};

export type LineHighlightOption = {
  startLineNumber: number;
  endLineNumber: number;
  options: monaco.editor.IModelDecorationOptions;
};

export const SupportedLanguages: monaco.languages.ILanguageExtensionPoint[] = [
  {
    id: "sql",
    extensions: [".sql"],
    aliases: ["SQL", "sql"],
    mimetypes: ["application/x-sql"],
  },
  {
    id: "javascript",
    extensions: [".js"],
    aliases: ["JS", "js"],
    mimetypes: ["application/javascript"],
  },
  {
    id: "redis",
    extensions: [".redis"],
    aliases: ["REDIS", "redis"],
    mimetypes: ["application/redis"],
  },
];

export type AutoCompleteContextScene = "query" | "all";

export type AutoCompleteContext = {
  instance: string;
  database?: string;
  schema?: string;
  scene?: AutoCompleteContextScene;
};

export type FormatContentOptions = {
  disabled: boolean;
  callback?: (
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ) => void;
};
