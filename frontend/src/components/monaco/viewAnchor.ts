import type * as monaco from "monaco-editor";
import type { IStandaloneCodeEditor } from "./types";

// A place to land in an editor, recorded by line rather than scroll pixels so
// it carries over to an editor showing the same text with different view
// zones, such as the statement editor without its inline threads.
export type EditorViewAnchor = {
  selection: monaco.ISelection;
  revealLineNumber: number;
  revealAt: "top" | "center";
};

const collapsedSelectionAt = (lineNumber: number): monaco.ISelection => ({
  selectionStartLineNumber: lineNumber,
  selectionStartColumn: 1,
  positionLineNumber: lineNumber,
  positionColumn: 1,
});

// Keeps the reader's place: the top visible line stays on top.
export const captureEditorViewAnchor = (
  editor: Pick<
    IStandaloneCodeEditor,
    "getModel" | "getSelection" | "getVisibleRanges"
  >
): EditorViewAnchor | undefined => {
  if (!editor.getModel()) return undefined;
  const visibleRanges = editor.getVisibleRanges();
  const topLineNumber = visibleRanges[0]?.startLineNumber ?? 1;
  const selection = editor.getSelection();
  // A reader who never placed a cursor still has one at the top of the
  // document; only keep a cursor they can see.
  const selectionIsVisible =
    selection !== null &&
    visibleRanges.some(
      (range) =>
        range.startLineNumber <= selection.positionLineNumber &&
        selection.positionLineNumber <= range.endLineNumber
    );
  return {
    selection: selectionIsVisible
      ? {
          selectionStartLineNumber: selection.selectionStartLineNumber,
          selectionStartColumn: selection.selectionStartColumn,
          positionLineNumber: selection.positionLineNumber,
          positionColumn: selection.positionColumn,
        }
      : collapsedSelectionAt(topLineNumber),
    revealLineNumber: topLineNumber,
    revealAt: "top",
  };
};

// Jumps to a line from elsewhere on the page, centered for context.
export const lineViewAnchor = (lineNumber: number): EditorViewAnchor => ({
  selection: collapsedSelectionAt(lineNumber),
  revealLineNumber: lineNumber,
  revealAt: "center",
});

// Lines past the end land on the last line: the anchor may come from text
// that has since been edited shorter.
export const focusEditorAtAnchor = (
  editor: Pick<
    IStandaloneCodeEditor,
    | "focus"
    | "getModel"
    | "getTopForLineNumber"
    | "revealLineInCenter"
    | "setScrollTop"
    | "setSelection"
  >,
  anchor: EditorViewAnchor
) => {
  const lineCount = editor.getModel()?.getLineCount();
  if (!lineCount) return;
  const clampLine = (lineNumber: number) => Math.min(lineNumber, lineCount);
  const { selection } = anchor;
  editor.setSelection({
    ...selection,
    selectionStartLineNumber: clampLine(selection.selectionStartLineNumber),
    positionLineNumber: clampLine(selection.positionLineNumber),
  });
  const revealLineNumber = clampLine(anchor.revealLineNumber);
  if (anchor.revealAt === "top") {
    editor.setScrollTop(editor.getTopForLineNumber(revealLineNumber));
  } else {
    editor.revealLineInCenter(revealLineNumber);
  }
  editor.focus();
};
