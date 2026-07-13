import { describe, expect, test } from "vitest";
import { ColumnResizeHandle } from "./column-resize-handle";
import { TableBody, TableEmptyView, TableRow } from "./table";

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

  test("TableEmptyView renders a full-span centered placeholder", () => {
    const element = TableEmptyView({
      colSpan: 4,
      children: "No data",
      contentClassName: "sticky left-0",
      contentStyle: { width: 640 },
      contentTestId: "table-empty",
    });

    expect(element.type).toBe(TableRow);
    const cell = element.props.children;
    expect(cell.props.colSpan).toBe(4);
    expect(cell.props.className).toContain("text-center");
    expect(cell.props.className).toContain("text-control-placeholder");

    const content = cell.props.children;
    expect(content.props["data-testid"]).toBe("table-empty");
    expect(content.props.className).toContain("flex");
    expect(content.props.className).toContain("items-center");
    expect(content.props.className).toContain("justify-center");
    expect(content.props.className).toContain("sticky left-0");
    expect(content.props.style).toEqual({ width: 640 });
    expect(content.props.children).toBe("No data");
  });

  test("ColumnResizeHandle uses a raised 12px hitbox around a 2px visual bar", () => {
    const element = ColumnResizeHandle({ onMouseDown: () => {} });

    expect(element.props.className).toContain("right-[-6px]");
    expect(element.props.className).toContain("w-3");
    expect(element.props.className).toContain("z-10");
    expect(element.props.children.props.className).toContain("w-0.5");
  });
});
