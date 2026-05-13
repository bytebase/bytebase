import { describe, expect, test } from "vitest";
import { ColumnResizeHandle } from "./column-resize-handle";
import { TableBody, TableRow } from "./table";

describe("table primitives", () => {
  test("TableBody stripes rows by default", () => {
    const element = TableBody({ children: null });

    expect(element.props.className).toContain(
      "[&_tr:nth-child(even)]:bg-control-bg/50"
    );
  });

  test("TableBody can disable striping", () => {
    const element = TableBody({ children: null, striped: false });

    expect(element.props.className).not.toContain(
      "[&_tr:nth-child(even)]:bg-control-bg/50"
    );
  });

  test("TableRow can opt out of striping", () => {
    const element = TableRow({ children: null, striped: false });

    expect(element.props["data-striped"]).toBe("false");
    expect(element.props.className).toContain("!bg-transparent");
  });

  test("ColumnResizeHandle uses a raised 12px hitbox around a 3px visual bar", () => {
    const element = ColumnResizeHandle({ onMouseDown: () => {} });

    expect(element.props.className).toContain("right-[-6px]");
    expect(element.props.className).toContain("w-3");
    expect(element.props.className).toContain("z-10");
    expect(element.props.children.props.className).toContain("w-[3px]");
  });
});
