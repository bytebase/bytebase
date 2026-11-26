import { act, fireEvent, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { TableResizeOptions } from "./useTableResize";
import { useTableResize } from "./useTableResize";

const options: TableResizeOptions = {
  columns: [
    { id: "first", name: "first", columnType: "TEXT" },
    { id: "second", name: "second", columnType: "TEXT" },
  ],
  rows: [],
  containerWidth: 800,
  minWidth: 64,
  maxWidth: 640,
  getRowCellContent: () => "x".repeat(200),
};

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
    measureText: (text: string) => ({ width: text.length * 8 }),
  } as CanvasRenderingContext2D);
});

afterEach(() => {
  fireEvent.pointerUp(document);
  vi.restoreAllMocks();
});

function startResize(
  onResizeStart: ReturnType<typeof useTableResize>["onResizeStart"],
  columnIndex: number,
  clientX: number
) {
  act(() => {
    onResizeStart(
      columnIndex,
      new MouseEvent("pointerdown", {
        clientX,
      }) as unknown as React.PointerEvent
    );
  });
}

function moveResize(clientX: number) {
  fireEvent(document, new MouseEvent("pointermove", { clientX }));
}

describe("useTableResize", () => {
  test.each([1, 2])(
    "widens long-value column %i beyond its initial maximum",
    (columnIndex) => {
      const { result } = renderHook(() => useTableResize(options));

      expect(result.current.getColumnWidth(columnIndex)).toBe(640);

      startResize(result.current.onResizeStart, columnIndex, 640);
      moveResize(880);

      expect(result.current.getColumnWidth(columnIndex)).toBe(880);
      expect(result.current.effectiveWidth).toBe(1560);
    }
  );

  test("widens a single data column beyond the container without snapping smaller", () => {
    const singleColumnOptions = {
      ...options,
      columns: options.columns.slice(0, 1),
    };
    const { result } = renderHook(() => useTableResize(singleColumnOptions));

    expect(result.current.getColumnWidth(1)).toBe(799);

    startResize(result.current.onResizeStart, 1, 799);
    moveResize(1039);

    expect(result.current.getColumnWidth(1)).toBe(1039);
  });

  test("keeps the minimum width and ends resizing on pointer release", () => {
    const { result } = renderHook(() => useTableResize(options));

    startResize(result.current.onResizeStart, 1, 640);
    moveResize(0);

    expect(result.current.getColumnWidth(1)).toBe(64);

    fireEvent.pointerUp(document);
    moveResize(1000);

    expect(result.current.getColumnWidth(1)).toBe(64);
    expect(document.body).not.toHaveClass("cursor-col-resize");
  });
});
