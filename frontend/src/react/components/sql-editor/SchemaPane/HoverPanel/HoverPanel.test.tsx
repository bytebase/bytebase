import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { HoverState } from "../hover-state";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// jsdom doesn't ship ResizeObserver. Stub it before HoverPanel is imported
// so its `useLayoutEffect` can wire up without exploding. The stub
// matches the real-DOM signature so TS type-check is happy.
class StubResizeObserver implements ResizeObserver {
  constructor(_cb: ResizeObserverCallback) {}
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
(
  globalThis as unknown as { ResizeObserver: typeof ResizeObserver }
).ResizeObserver = StubResizeObserver as unknown as typeof ResizeObserver;

const hoverState: {
  state: HoverState | undefined;
  position: { x: number; y: number };
  update: ReturnType<typeof vi.fn>;
} = {
  state: undefined,
  position: { x: 0, y: 0 },
  update: vi.fn(),
};

vi.mock("../hover-state", () => ({
  useHoverState: () => ({
    state: hoverState.state,
    position: hoverState.position,
    update: hoverState.update,
    setPosition: vi.fn(),
    cancel: vi.fn(),
  }),
}));

vi.mock("./ColumnInfo", () => ({
  ColumnInfo: () => <div data-testid="ColumnInfo" />,
}));
vi.mock("./TableInfo", () => ({
  TableInfo: () => <div data-testid="TableInfo" />,
}));
vi.mock("./TablePartitionInfo", () => ({
  TablePartitionInfo: () => <div data-testid="TablePartitionInfo" />,
}));
vi.mock("./ExternalTableInfo", () => ({
  ExternalTableInfo: () => <div data-testid="ExternalTableInfo" />,
}));
vi.mock("./ViewInfo", () => ({
  ViewInfo: () => <div data-testid="ViewInfo" />,
}));

vi.mock("@/react/components/ui/layer", () => ({
  LAYER_SURFACE_CLASS: "test-layer",
  getLayerRoot: () => document.body,
}));

vi.mock("@/utils", () => ({
  minmax: (value: number, min: number, max: number) =>
    Math.max(min, Math.min(max, value)),
}));

let HoverPanel: typeof import("./HoverPanel").HoverPanel;

const renderInto = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  hoverState.state = undefined;
  hoverState.position = { x: 0, y: 0 };
  hoverState.update.mockReset();
  ({ HoverPanel } = await import("./HoverPanel"));
});

afterEach(() => {
  // Each render mounts the panel into document.body via the layer-root
  // mock. Sweep any leftover panels so the next test sees a clean DOM.
  document.body.innerHTML = "";
});

describe("HoverPanel dispatch", () => {
  test("renders nothing when state is undefined", () => {
    const { container } = renderInto(
      <HoverPanel offsetX={0} offsetY={0} margin={4} />
    );
    expect(container.querySelector("[data-testid]")).toBeNull();
    expect(document.querySelector("[data-testid]")).toBeNull();
  });

  test("column + table → ColumnInfo", () => {
    hoverState.state = { database: "db", table: "users", column: "id" };
    hoverState.position = { x: 10, y: 20 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    expect(document.querySelector('[data-testid="ColumnInfo"]')).not.toBeNull();
  });

  test("table + partition → TablePartitionInfo", () => {
    hoverState.state = { database: "db", table: "events", partition: "p1" };
    hoverState.position = { x: 10, y: 20 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    expect(
      document.querySelector('[data-testid="TablePartitionInfo"]')
    ).not.toBeNull();
  });

  test("table only → TableInfo", () => {
    hoverState.state = { database: "db", table: "users" };
    hoverState.position = { x: 10, y: 20 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    expect(document.querySelector('[data-testid="TableInfo"]')).not.toBeNull();
  });

  test("externalTable → ExternalTableInfo", () => {
    hoverState.state = { database: "db", externalTable: "ext" };
    hoverState.position = { x: 10, y: 20 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    expect(
      document.querySelector('[data-testid="ExternalTableInfo"]')
    ).not.toBeNull();
  });

  test("view → ViewInfo", () => {
    hoverState.state = { database: "db", view: "v" };
    hoverState.position = { x: 10, y: 20 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    expect(document.querySelector('[data-testid="ViewInfo"]')).not.toBeNull();
  });
});

describe("HoverPanel clamp + invisibility", () => {
  test("offsets are applied to the cursor position", () => {
    hoverState.state = { database: "db", table: "t" };
    hoverState.position = { x: 100, y: 200 };
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 1000,
    });
    renderInto(<HoverPanel offsetX={4} offsetY={8} margin={4} />);
    const panel = document.querySelector(".test-layer") as HTMLElement | null;
    expect(panel).not.toBeNull();
    expect(panel!.style.left).toBe("104px");
    expect(panel!.style.top).toBe("208px");
  });

  test("y is clamped to (margin … innerHeight - popoverHeight - margin)", () => {
    hoverState.state = { database: "db", table: "t" };
    hoverState.position = { x: 0, y: 9999 };
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 600,
    });
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    const panel = document.querySelector(".test-layer") as HTMLElement | null;
    // popoverHeight defaults to 0 in jsdom, so the upper bound is
    // 600 - 0 - 4 = 596. The clamp pulls 9999 down to 596.
    expect(panel!.style.top).toBe("596px");
  });

  test("position (0,0) renders the panel as invisible", () => {
    hoverState.state = { database: "db", table: "t" };
    hoverState.position = { x: 0, y: 0 };
    renderInto(<HoverPanel offsetX={0} offsetY={0} margin={4} />);
    const panel = document.querySelector(".test-layer") as HTMLElement | null;
    expect(panel).not.toBeNull();
    // `show` is false → "invisible pointer-events-none" classes are on.
    expect(panel!.className).toContain("invisible");
    expect(panel!.className).toContain("pointer-events-none");
  });
});
