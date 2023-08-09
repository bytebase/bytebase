import type { editor as Editor } from "monaco-editor";
import { SQLDialect } from "@/types";
import { ExtractPromiseType } from "@/utils";
import sqlFormatter from "./sqlFormatter";
import { getBBTheme } from "./themes/bb";
import { getBBDarkTheme } from "./themes/bb-dark";

export const useMonaco = async () => {
  const [monaco, { default: EditorWorker }, { default: TSWorker }] =
    await Promise.all([
      import("monaco-editor"),
      import("monaco-editor/esm/vs/editor/editor.worker?worker"),
      import("monaco-editor/esm/vs/language/typescript/ts.worker?worker"),
    ]);

  const bbTheme = getBBTheme();
  const bbDarkTheme = getBBDarkTheme();
  monaco.editor.defineTheme("bb", bbTheme);
  monaco.editor.defineTheme("bb-dark", bbDarkTheme);

  self.MonacoEnvironment = {
    getWorker: (workerId, label) => {
      console.debug("MonacoEnvironment.getWorker", workerId, label);
      if (label === "javascript") {
        return new TSWorker();
      }
      return new EditorWorker();
    },
  };

  const dispose = () => {
    // Nothing todo
  };

  /**
   * set new content in monaco editor
   * use executeEdits API can preserve undo stack, allow user to undo/redo
   * @param editorInstance Editor.IStandaloneCodeEditor
   * @param content string
   */
  const setContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    content: string
  ) => {
    const range = editorInstance.getModel()?.getFullModelRange();
    // get the current endLineNumber, or use 100000 as the default
    const endLineNumber =
      range && range?.endLineNumber > 0 ? range.endLineNumber + 1 : 100000;
    editorInstance.executeEdits("delete-content", [
      {
        range: new monaco.Range(1, 1, endLineNumber, 1),
        text: "",
        forceMoveMarkers: true,
      },
    ]);
    // set the new content
    editorInstance.executeEdits("insert-content", [
      {
        range: new monaco.Range(1, 1, 1, 1),
        text: content,
        forceMoveMarkers: true,
      },
    ]);
  };

  const formatContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    dialect: SQLDialect
  ) => {
    const sql = editorInstance.getValue();
    const { data, error } = sqlFormatter(sql, dialect);
    if (error) {
      return;
    }
    setContent(editorInstance, data);
  };

  const setPositionAtEndOfLine = (
    editorInstance: Editor.IStandaloneCodeEditor
  ) => {
    const range = editorInstance.getModel()?.getFullModelRange();
    if (range) {
      editorInstance.setPosition({
        lineNumber: range?.endLineNumber,
        column: range?.endColumn,
      });
    }
  };

  return {
    dispose,
    monaco,
    setContent,
    formatContent,
    setPositionAtEndOfLine,
  };
};

export type MonacoHelper = ExtractPromiseType<ReturnType<typeof useMonaco>>;
