import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
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

  test("a second onResizeStart removes the prior drag's document listeners", () => {
    // Regression: if a second drag started without an intervening mouseup
    // (rapid sequential mousedowns, missed mouseup, multi-touch trackpad),
    // the prior drag's document listeners used to leak — only the latest
    // teardown was reachable. The behavioral end state is identical whether
    // or not the leak exists (both leaked + new listeners read from the same
    // dragRef and produce the same widths), so this test counts net
    // mousemove/mouseup registrations via spies to prove the cleanup.
    const addSpy = vi.spyOn(document, "addEventListener");
    const removeSpy = vi.spyOn(document, "removeEventListener");
    const netDragListeners = () => {
      const added = addSpy.mock.calls.filter(
        ([type]) => type === "mousemove" || type === "mouseup"
      ).length;
      const removed = removeSpy.mock.calls.filter(
        ([type]) => type === "mousemove" || type === "mouseup"
      ).length;
      return added - removed;
    };
    try {
      mount([
        { key: "a", defaultWidth: 100 },
        { key: "b", defaultWidth: 200 },
      ]);
      addSpy.mockClear();
      removeSpy.mockClear();

      startDrag(0, 0);
      expect(netDragListeners()).toBe(2); // mousemove + mouseup

      // Second drag without intervening mouseup: defensive teardown must
      // remove drag #1's pair before registering drag #2's pair.
      startDrag(1, 100);
      expect(netDragListeners()).toBe(2);

      moveMouse(120); // should only affect column b
      expect(handle.current!.widths).toEqual([100, 220]);

      releaseMouse();
      expect(netDragListeners()).toBe(0);
      expect(document.body.style.cursor).toBe("");
      expect(document.body.style.userSelect).toBe("");
    } finally {
      addSpy.mockRestore();
      removeSpy.mockRestore();
    }
  });

  test("drag uses snapshotted minWidth, not the live column constraint", () => {
    // Eager-capture contract: a drag's minWidth is fixed at gesture start.
    // If the caller swaps the column at the dragged index for one with a
    // more permissive minWidth, the drag must still clamp at the original.
    // This is the "equal-length column swap" regression.
    mount([
      { key: "a", defaultWidth: 100 },
      { key: "b", defaultWidth: 200, minWidth: 150 },
    ]);
    startDrag(1, 200); // snapshot minWidth = 150
    expect(handle.current!.widths).toEqual([100, 200]);

    // Caller swaps column at index 1 for one with a far more permissive minWidth.
    act(() => {
      root.render(
        <Harness
          columns={[
            { key: "a", defaultWidth: 100 },
            { key: "c", defaultWidth: 200, minWidth: 50 },
          ]}
          handleRef={handle}
        />
      );
    });

    // Drag the column far to the left. Live minWidth=50 would allow it, but
    // the snapshotted minWidth=150 must clamp.
    moveMouse(0); // delta = -200, raw new = 0, clamp to snapshotted 150
    expect(handle.current!.widths[1]).toBe(150);
    releaseMouse();
  });

  test("setWidths short-circuits when clamped at minWidth (no-op re-renders)", () => {
    // Without the prev[colIndex] === newWidth guard, dragging the cursor
    // past minWidth would fire setWidths with the same clamped value on
    // every pixel of motion, re-rendering every consumer. Verify the
    // widths array reference is preserved (no new allocation) across a
    // clamped tick — that's how React skips the re-render.
    mount([{ key: "a", defaultWidth: 200, minWidth: 150 }]);
    startDrag(0, 200);
    moveMouse(100); // delta=-100, raw=100, clamped to 150 — first allocation
    const widthsAfterClamp = handle.current!.widths;
    expect(widthsAfterClamp).toEqual([150]);

    // Move further past minWidth. newWidth still clamps to 150 — no-op.
    moveMouse(50); // delta=-150, raw=50, clamped to 150
    expect(handle.current!.widths).toBe(widthsAfterClamp); // same ref
    moveMouse(0); // delta=-200, raw=0, clamped to 150
    expect(handle.current!.widths).toBe(widthsAfterClamp); // still same ref

    // Drag back up. At raw=120 the clamp is still engaged (120 < 150).
    moveMouse(120); // delta=-80, raw=120, still clamped to 150 — no-op
    expect(handle.current!.widths).toBe(widthsAfterClamp); // same ref
    // Cross above minWidth — now a real change, fresh allocation.
    moveMouse(180); // delta=-20, raw=180 > 150 — real change to 180
    expect(handle.current!.widths).toEqual([180]);
    expect(handle.current!.widths).not.toBe(widthsAfterClamp);
    releaseMouse();
  });

  test("drag starts with defaultWidth fallback if widths state lags behind a newly added column", () => {
    // The widths state is sized once at mount; if a caller adds a column
    // afterwards, widthsRef.current[newIndex] is undefined. The hook must
    // fall back to the column's defaultWidth so the gesture starts from a
    // sensible value rather than producing NaN.
    mount([{ key: "a", defaultWidth: 100 }]);
    act(() => {
      root.render(
        <Harness
          columns={[
            { key: "a", defaultWidth: 100 },
            { key: "b", defaultWidth: 200 },
          ]}
          handleRef={handle}
        />
      );
    });

    // widths state is still [100]; widthsRef.current[1] is undefined.
    startDrag(1, 500);
    moveMouse(560); // delta = 60, startWidth fallback = 200, new = 260
    expect(handle.current!.widths[1]).toBe(260);
    expect(Number.isNaN(handle.current!.widths[1])).toBe(false);
    releaseMouse();
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
