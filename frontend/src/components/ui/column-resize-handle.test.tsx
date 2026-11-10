import { describe, expect, test } from "vitest";
import { ColumnResizeHandle } from "./column-resize-handle";

describe("ColumnResizeHandle", () => {
  test("keeps its hitbox inside the header edge", () => {
    const element = ColumnResizeHandle({ onMouseDown: () => {} });

    expect(element.props.className).toContain("right-0");
    expect(element.props.className).not.toContain("right-[-6px]");
    expect(element.props.className).toContain("w-3");
    expect(element.props.children.props.className).toContain("right-0");
    expect(element.props.children.props.className).toContain("w-0.5");
  });

  test("sets no z-index, so inline popups paint over it", () => {
    const element = ColumnResizeHandle({ onMouseDown: () => {} });

    expect(element.props.className).not.toMatch(/(?:^|[\s:])!?-?z-/);
  });
});
