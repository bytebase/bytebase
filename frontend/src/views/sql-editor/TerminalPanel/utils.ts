import type { editor as Editor } from "monaco-editor";

export const checkEndsWithSemicolon = (
  editor: Editor.IStandaloneCodeEditor
): boolean => {
  const value = editor.getValue();
  if (value.endsWith(";")) {
    return true;
  }
  return false;
};

export const checkCursorAtLast = (
  editor: Editor.IStandaloneCodeEditor
): boolean => {
  const model = editor.getModel();
  if (model) {
    const maxLine = model.getLineCount();
    const maxColumn = model.getLineMaxColumn(maxLine);
    const cursor = editor.getPosition();
    const isCursorAtLast = !!cursor?.equals({
      lineNumber: maxLine,
      column: maxColumn,
    });
    if (isCursorAtLast) {
      return true;
    }
  }
  return false;
};

export const checkCursorAtFirstLine = (
  editor: Editor.IStandaloneCodeEditor
): boolean => {
  const model = editor.getModel();
  if (model) {
    const cursor = editor.getPosition();
    if (cursor?.lineNumber === 1) return true;
  }
  return false;
};

export const checkCursorAtLastLine = (
  editor: Editor.IStandaloneCodeEditor
): boolean => {
  const model = editor.getModel();
  if (model) {
    const maxLine = model.getLineCount();
    const cursor = editor.getPosition();
    if (cursor?.lineNumber === maxLine) return true;
  }
  return false;
};
