// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import {
  captureEditorViewAnchor,
  focusEditorAtAnchor,
  lineViewAnchor,
} from "./viewAnchor";

const readerAt = ({
  visibleRanges,
  cursorLine,
}: {
  visibleRanges: { startLineNumber: number; endLineNumber: number }[];
  cursorLine: number;
}) =>
  ({
    getModel: () => ({}),
    getVisibleRanges: () => visibleRanges,
    getSelection: () => ({
      selectionStartLineNumber: cursorLine,
      selectionStartColumn: 1,
      positionLineNumber: cursorLine,
      positionColumn: 4,
    }),
  }) as unknown as Parameters<typeof captureEditorViewAnchor>[0];

const editorWithLines = (lineCount: number) => ({
  focus: vi.fn(),
  getModel: () =>
    ({ getLineCount: () => lineCount }) as unknown as ReturnType<
      Parameters<typeof focusEditorAtAnchor>[0]["getModel"]
    >,
  getTopForLineNumber: (lineNumber: number) => lineNumber * 20,
  revealLineInCenter: vi.fn(),
  setScrollTop: vi.fn(),
  setSelection: vi.fn(),
});

describe("captureEditorViewAnchor", () => {
  test("keeps the top visible line and a cursor the reader can see", () => {
    expect(
      captureEditorViewAnchor(
        readerAt({
          visibleRanges: [
            { startLineNumber: 750, endLineNumber: 766 },
            { startLineNumber: 767, endLineNumber: 780 },
          ],
          cursorLine: 770,
        })
      )
    ).toEqual({
      selection: {
        selectionStartLineNumber: 770,
        selectionStartColumn: 1,
        positionLineNumber: 770,
        positionColumn: 4,
      },
      revealLineNumber: 750,
      revealAt: "top",
    });
  });

  test("moves a cursor scrolled out of view to the top visible line", () => {
    expect(
      captureEditorViewAnchor(
        readerAt({
          visibleRanges: [{ startLineNumber: 757, endLineNumber: 780 }],
          cursorLine: 1,
        })
      )?.selection
    ).toEqual({
      selectionStartLineNumber: 757,
      selectionStartColumn: 1,
      positionLineNumber: 757,
      positionColumn: 1,
    });
  });

  test("returns nothing for an editor without a model", () => {
    expect(
      captureEditorViewAnchor({
        getModel: () => null,
        getVisibleRanges: () => [],
        getSelection: () => null,
      })
    ).toBeUndefined();
  });
});

describe("focusEditorAtAnchor", () => {
  test("scrolls a top anchor's line to the top of the viewport", () => {
    const editor = editorWithLines(800);
    focusEditorAtAnchor(editor, {
      selection: {
        selectionStartLineNumber: 766,
        selectionStartColumn: 1,
        positionLineNumber: 766,
        positionColumn: 8,
      },
      revealLineNumber: 757,
      revealAt: "top",
    });
    expect(editor.setScrollTop).toHaveBeenCalledWith(757 * 20);
    expect(editor.revealLineInCenter).not.toHaveBeenCalled();
    expect(editor.focus).toHaveBeenCalled();
  });

  test("centers a line anchor and clamps it to shortened text", () => {
    const editor = editorWithLines(3);
    focusEditorAtAnchor(editor, lineViewAnchor(99));
    expect(editor.setSelection).toHaveBeenCalledWith({
      selectionStartLineNumber: 3,
      selectionStartColumn: 1,
      positionLineNumber: 3,
      positionColumn: 1,
    });
    expect(editor.revealLineInCenter).toHaveBeenCalledWith(3);
    expect(editor.setScrollTop).not.toHaveBeenCalled();
  });
});
