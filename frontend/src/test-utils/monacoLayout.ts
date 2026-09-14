import { vi } from "vitest";

// Vertical layout for a fake Monaco editor: 24px lines under 8px padding,
// with view zones stacked after their line in insertion order.
export const LINE_HEIGHT = 24;
export const PADDING_TOP = 8;

export const lineTop = (lineNumber: number) =>
  PADDING_TOP + (lineNumber - 1) * LINE_HEIGHT;

export type FakeWhitespace = {
  id: string;
  afterLineNumber: number;
  height: number;
};

// The layout getters MonacoViewZone reveals through, over the zones the
// fake editor currently holds. `scroll.top` is readable and settable.
export function fakeEditorLayout(
  whitespaces: () => FakeWhitespace[],
  { scrollTop = 0, viewportHeight = 400 } = {}
) {
  const scroll = { top: scrollTop };
  const before = (lineNumber: number) =>
    whitespaces()
      .filter((zone) => zone.afterLineNumber < lineNumber)
      .reduce((sum, zone) => sum + zone.height, 0);
  return {
    scroll,
    getLayoutInfo: () => ({
      contentLeft: 52,
      height: viewportHeight,
      width: 800,
      verticalScrollbarWidth: 14,
      minimap: { minimapWidth: 0 },
    }),
    getScrollTop: () => scroll.top,
    setScrollTop: vi.fn((top: number) => {
      scroll.top = top;
    }),
    getTopForLineNumber: (lineNumber: number) =>
      lineTop(lineNumber) + before(lineNumber),
    getBottomForLineNumber: (lineNumber: number) =>
      lineTop(lineNumber) + LINE_HEIGHT + before(lineNumber),
    getWhitespaces: whitespaces,
  };
}
