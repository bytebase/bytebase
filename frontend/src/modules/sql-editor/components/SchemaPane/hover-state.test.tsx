import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type HoverActions,
  type HoverStateContextValue,
  HoverStateProvider,
  useHoverActions,
  useHoverState,
} from "./hover-state";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

function Probe({
  handleRef,
}: {
  handleRef: { current: HoverStateContextValue | null };
}) {
  handleRef.current = useHoverState();
  return null;
}

let actionRenders: number;
let actions: HoverActions | null;

function ActionsProbe() {
  actionRenders++;
  actions = useHoverActions();
  return null;
}

let container: HTMLDivElement;
let root: Root;
let handle: { current: HoverStateContextValue | null };

function mount() {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  handle = { current: null };
  actionRenders = 0;
  actions = null;
  act(() => {
    root.render(
      <HoverStateProvider>
        <Probe handleRef={handle} />
        <ActionsProbe />
      </HoverStateProvider>
    );
  });
}

function unmount() {
  act(() => {
    root.unmount();
  });
  container.remove();
}

describe("HoverStateProvider", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mount();
  });

  afterEach(() => {
    unmount();
    vi.useRealTimers();
  });

  test("clears hover state when the cursor leaves the browser window", () => {
    act(() => {
      handle.current!.update(
        { database: "instances/i/databases/db", table: "users" },
        "before",
        0
      );
    });
    expect(handle.current!.state).toEqual({
      database: "instances/i/databases/db",
      table: "users",
    });

    act(() => {
      window.dispatchEvent(
        new MouseEvent("mouseout", { bubbles: true, relatedTarget: null })
      );
    });

    expect(handle.current!.state).toBeUndefined();
  });

  test("position and state changes do not re-render action-only consumers", () => {
    expect(actionRenders).toBe(1);
    expect(actions!.isActive()).toBe(false);

    act(() => {
      actions!.setPosition({ x: 10, y: 20 });
      actions!.update(
        { database: "instances/i/databases/db", table: "users" },
        "before",
        0
      );
    });

    expect(handle.current!.position).toEqual({ x: 10, y: 20 });
    expect(handle.current!.state).toEqual({
      database: "instances/i/databases/db",
      table: "users",
    });
    expect(actions!.isActive()).toBe(true);
    expect(actionRenders).toBe(1);
  });
});
