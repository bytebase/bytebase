import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type HoverStateContextValue,
  HoverStateProvider,
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

let container: HTMLDivElement;
let root: Root;
let handle: { current: HoverStateContextValue | null };

function mount() {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  handle = { current: null };
  act(() => {
    root.render(
      <HoverStateProvider>
        <Probe handleRef={handle} />
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
});
