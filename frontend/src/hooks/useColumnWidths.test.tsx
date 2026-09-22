import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  distributeColumnWidths,
  fillableWidth,
  useColumnWidths,
} from "./useColumnWidths";

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

describe("distributeColumnWidths", () => {
  test("a column that does not grow keeps its width at any container width", () => {
    const columns = [
      { key: "date", defaultWidth: 260, minWidth: 140, grow: false },
      { key: "title", defaultWidth: 400, minWidth: 180 },
      { key: "actions", defaultWidth: 100, resizable: false },
    ];
    for (const containerWidth of [700, 1100, 1800]) {
      const [date, title, actions] = distributeColumnWidths(
        columns,
        containerWidth
      );
      expect(date).toBe(260);
      expect(actions).toBe(100);
      expect(title).toBe(Math.max(180, containerWidth - 360));
    }
  });

  test("fills the container exactly whenever every floor fits", () => {
    // A column raised to its floor has to take that width from the others,
    // or the table overruns a container it could have filled.
    const columns = [
      { key: "status", defaultWidth: 160, minWidth: 128 },
      { key: "creator", defaultWidth: 200, minWidth: 128 },
      { key: "date", defaultWidth: 270, minWidth: 140, grow: false },
      { key: "statement", defaultWidth: 400, minWidth: 180 },
      { key: "actions", defaultWidth: 140, resizable: false },
    ];
    const floors = 128 + 128 + 270 + 180 + 140;
    for (let containerWidth = floors; containerWidth <= 1800; containerWidth++) {
      const widths = distributeColumnWidths(columns, containerWidth);
      expect(widths.reduce((sum, w) => sum + w, 0)).toBe(containerWidth);
      columns.forEach((column, i) => {
        expect(widths[i]).toBeGreaterThanOrEqual(
          column.minWidth ?? column.defaultWidth
        );
      });
      expect(widths[2]).toBe(270);
      expect(widths[4]).toBe(140);
    }
  });

  test("keeps every floor when the container is too narrow for them", () => {
    const widths = distributeColumnWidths(
      [
        { key: "a", defaultWidth: 200, minWidth: 150 },
        { key: "b", defaultWidth: 200, minWidth: 150 },
      ],
      200
    );
    expect(widths).toEqual([150, 150]);
  });

  test("its minimum floors a drag, not the width it opens at", () => {
    const columns = [
      { key: "date", defaultWidth: 260, minWidth: 140, grow: false },
      { key: "title", defaultWidth: 400 },
    ];
    expect(distributeColumnWidths(columns, 500)[0]).toBe(260);
    mount(columns);
    startDrag(0, 400);
    moveMouse(0);
    expect(handle.current!.widths[0]).toBe(140);
    releaseMouse();
    unmount();
  });
});

describe("distributeColumnWidths with a yield order", () => {
  const columns = [
    { key: "date", defaultWidth: 270, minWidth: 140, grow: false, yieldOrder: 1 },
    { key: "title", defaultWidth: 400, minWidth: 180, yieldOrder: 2 },
    { key: "owner", defaultWidth: 200, minWidth: 128, yieldOrder: 3 },
    { key: "actions", defaultWidth: 140, resizable: false },
  ];
  const preferred = 270 + 400 + 200 + 140;
  const floors = 140 + 180 + 128 + 140;

  test("gives way in order, each column to its floor before the next", () => {
    for (let width = floors; width < preferred; width++) {
      const [date, title, owner, actions] = distributeColumnWidths(
        columns,
        width
      );
      expect(date + title + owner + actions).toBe(width);
      expect(actions).toBe(140);
      if (title < 400) {
        expect(date).toBe(140);
      }
      if (owner < 200) {
        expect(title).toBe(180);
      }
    }
  });

  test("keeps every preferred width once the container holds them", () => {
    const [date, title, owner] = distributeColumnWidths(columns, preferred);
    expect([date, title, owner]).toEqual([270, 400, 200]);
    // Spare width still goes to the columns that grow, not the date.
    expect(distributeColumnWidths(columns, preferred + 300)[0]).toBe(270);
  });

  test("raises a column whose default is under its floor, and takes that from the next", () => {
    const narrowDefault = [
      { defaultWidth: 100, minWidth: 120, yieldOrder: 1 },
      { defaultWidth: 300, minWidth: 100, yieldOrder: 2 },
    ];
    expect(distributeColumnWidths(narrowDefault, 350)).toEqual([120, 230]);
  });
});

describe("fillableWidth", () => {
  // jsdom lays nothing out, so each case states what a browser reports; it
  // does report the inline border and padding back as the computed style.
  const scroller = (
    rect: number,
    offset: number,
    client: number,
    { border = "0px", padding = "0px" } = {}
  ) => {
    const node = document.createElement("div");
    node.style.borderLeftWidth = border;
    node.style.borderRightWidth = border;
    node.style.paddingLeft = padding;
    node.style.paddingRight = padding;
    document.body.appendChild(node);
    Object.defineProperty(node, "offsetWidth", { value: offset });
    Object.defineProperty(node, "clientWidth", { value: client });
    node.getBoundingClientRect = () => ({ width: rect }) as DOMRect;
    return node;
  };
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test.each([
    [
      "a whole width, inside a 1px border",
      () => scroller(800, 800, 798, { border: "1px" }),
      798,
    ],
    // 795.5 inside reads as 796; a table that wide scrolls by half a pixel.
    [
      "a fractional width clientWidth rounds up",
      () => scroller(797.5, 798, 796, { border: "1px" }),
      795,
    ],
    // At 150% a 1px border snaps to two thirds of a pixel: 798.67 inside,
    // though offsetWidth less clientWidth reads 1.
    [
      "a width whose borders the browser snapped",
      () => scroller(800, 800, 799, { border: "0.666667px" }),
      798,
    ],
    // 798.47 inside reads as 798, so the rounded widths differ by 2 and leave
    // two thirds of a pixel over the borders: rounding, not a scrollbar.
    [
      "a width whose rounding looks like a scrollbar",
      () => scroller(799.8, 800, 798, { border: "0.666667px" }),
      798,
    ],
    // 17px beside the content, known only to the pixel from two rounded widths.
    ["a width beside a vertical scrollbar", () => scroller(800, 800, 783), 782],
    [
      "a width inside padding",
      () => scroller(800, 800, 800, { padding: "8px" }),
      784,
    ],
  ])("fills %s without scrolling", (_, make, width) => {
    expect(fillableWidth(make())).toBe(width);
  });
});
