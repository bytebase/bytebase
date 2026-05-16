import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { useColumnWidths } from "./useColumnWidths";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

interface Column {
  key: string;
  defaultWidth: number;
  minWidth?: number;
}

type Handle = ReturnType<typeof useColumnWidths<Column>>;

function Harness({
  columns,
  handleRef,
}: {
  columns: Column[];
  handleRef: { current: Handle | null };
}) {
  const result = useColumnWidths(columns);
  handleRef.current = result;
  return null;
}

let container: HTMLDivElement;
let root: Root;
let handle: { current: Handle | null };

function mount(columns: Column[]) {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  handle = { current: null };
  act(() => {
    root.render(<Harness columns={columns} handleRef={handle} />);
  });
}

function unmount() {
  act(() => {
    root.unmount();
  });
  container.remove();
}

function startDrag(colIndex: number, clientX: number) {
  // Build a React.MouseEvent stand-in. The hook only reads
  // `clientX`, `preventDefault`, and `stopPropagation`.
  const event = {
    clientX,
    preventDefault: () => {},
    stopPropagation: () => {},
  } as unknown as React.MouseEvent;
  act(() => {
    handle.current!.onResizeStart(colIndex, event);
  });
}

function moveMouse(clientX: number) {
  act(() => {
    document.dispatchEvent(new MouseEvent("mousemove", { clientX }));
  });
}

function releaseMouse() {
  act(() => {
    document.dispatchEvent(new MouseEvent("mouseup"));
  });
}

describe("useColumnWidths", () => {
  afterEach(() => {
    // Defensive: make sure no test leaks listeners/body styles.
    if (root) {
      try {
        unmount();
      } catch {
        // already unmounted
      }
    }
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  });

  test("initial widths come from defaultWidth and totalWidth sums them", () => {
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200 },
      { key: "c", defaultWidth: 50 },
    ]);
    expect(handle.current!.widths).toEqual([100, 200, 50]);
    expect(handle.current!.totalWidth).toBe(350);
  });

  test("dragging a column updates that column positionally", () => {
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200 },
      { key: "c", defaultWidth: 50 },
    ]);
    startDrag(1, 500);
    moveMouse(540);
    expect(handle.current!.widths).toEqual([100, 240, 50]);
    expect(handle.current!.totalWidth).toBe(390);
    releaseMouse();
  });

  test("dragging left shrinks the column", () => {
    mount([
      { key: "a", defaultWidth: 300 },
      { key: "b", defaultWidth: 200 },
    ]);
    startDrag(0, 100);
    moveMouse(30); // delta = -70, new = 230
    expect(handle.current!.widths).toEqual([230, 200]);
    releaseMouse();
  });

  test("minWidth clamps shrinking", () => {
    mount([{ key: "a", defaultWidth: 200, minWidth: 150 }]);
    startDrag(0, 500);
    moveMouse(0); // delta = -500, raw new = -300, clamp to 150
    expect(handle.current!.widths).toEqual([150]);
    releaseMouse();
  });

  test("minWidth defaults to 40 when not provided", () => {
    mount([{ key: "a", defaultWidth: 100 }]);
    startDrag(0, 200);
    moveMouse(0); // delta = -200, raw new = -100, clamp to 40
    expect(handle.current!.widths).toEqual([40]);
    releaseMouse();
  });

  test("mouseup tears down listeners and restores body styles", () => {
    mount([{ key: "a", defaultWidth: 100 }]);
    startDrag(0, 0);
    expect(document.body.style.cursor).toBe("col-resize");
    expect(document.body.style.userSelect).toBe("none");

    moveMouse(50);
    expect(handle.current!.widths).toEqual([150]);

    releaseMouse();
    expect(document.body.style.cursor).toBe("");
    expect(document.body.style.userSelect).toBe("");

    // After release, further mousemove must be ignored.
    moveMouse(1000);
    expect(handle.current!.widths).toEqual([150]);
  });

  test("unmount mid-drag tears down listeners and restores body styles", () => {
    mount([{ key: "a", defaultWidth: 100 }]);
    startDrag(0, 0);
    expect(document.body.style.cursor).toBe("col-resize");
    expect(document.body.style.userSelect).toBe("none");

    unmount();
    expect(document.body.style.cursor).toBe("");
    expect(document.body.style.userSelect).toBe("");

    // After unmount, the document-level listener must not still
    // be alive. Dispatching a mousemove should be a no-op.
    document.dispatchEvent(new MouseEvent("mousemove", { clientX: 1000 }));
    // No assertion on widths (hook is unmounted), but if cleanup
    // is broken vitest would surface "act outside of a test" warnings.
  });

  test("only the dragged column changes; siblings stay put", () => {
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200 },
      { key: "c", defaultWidth: 50 },
    ]);
    startDrag(2, 800);
    moveMouse(900); // c grows by 100
    expect(handle.current!.widths).toEqual([100, 200, 150]);
    releaseMouse();
  });

  test("dragging successive columns updates each independently", () => {
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200 },
    ]);

    startDrag(0, 50);
    moveMouse(80); // a: 130
    releaseMouse();
    expect(handle.current!.widths).toEqual([130, 200]);

    startDrag(1, 300);
    moveMouse(250); // b: 150
    releaseMouse();
    expect(handle.current!.widths).toEqual([130, 150]);
  });

  test("schedule-then-teardown invariant: setWidths updater must not deref the drag ref", () => {
    // Regression: mousemove + mouseup in one batch used to crash because the
    // updater closure read dragRef.current after teardown nulled it.
    mount([{ key: "a", defaultWidth: 100 }]);
    startDrag(0, 0);
    expect(() => {
      act(() => {
        document.dispatchEvent(new MouseEvent("mousemove", { clientX: 50 }));
        document.dispatchEvent(new MouseEvent("mouseup"));
      });
    }).not.toThrow();
    expect(handle.current!.widths).toEqual([150]);
  });

  test("onResizeStart identity is stable across width changes", () => {
    // Regression: putting `widths` in onResizeStart's deps caused it to rebind
    // on every mousemove tick, re-rendering every header / row consumer.
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200 },
    ]);
    const originalOnResizeStart = handle.current!.onResizeStart;
    startDrag(0, 50);
    moveMouse(80);
    moveMouse(120);
    releaseMouse();
    expect(handle.current!.widths).toEqual([170, 200]);
    expect(handle.current!.onResizeStart).toBe(originalOnResizeStart);
  });
});
