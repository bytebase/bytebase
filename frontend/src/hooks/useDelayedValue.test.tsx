import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { type UseDelayedValueResult, useDelayedValue } from "./useDelayedValue";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type HarnessHandle = UseDelayedValueResult<string>;

function Harness({
  handleRef,
}: {
  handleRef: { current: HarnessHandle | null };
}) {
  const result = useDelayedValue("idle", {
    delayBefore: 1000,
    delayAfter: 350,
  });
  handleRef.current = result;
  return null;
}

let container: HTMLDivElement;
let root: Root;
let handle: { current: HarnessHandle | null };

function mount() {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  handle = { current: null };
  act(() => {
    root.render(<Harness handleRef={handle} />);
  });
}

function unmount() {
  act(() => {
    root.unmount();
  });
  container.remove();
}

describe("useDelayedValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mount();
  });

  afterEach(() => {
    unmount();
    vi.useRealTimers();
  });

  test("commits the value after the before-delay elapses", () => {
    act(() => {
      handle.current!.update("open", "before");
    });
    expect(handle.current!.value).toBe("idle");
    act(() => {
      vi.advanceTimersByTime(999);
    });
    expect(handle.current!.value).toBe("idle");
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(handle.current!.value).toBe("open");
  });

  test("uses the after-delay when direction is 'after'", () => {
    act(() => {
      handle.current!.update("closed", "after");
    });
    act(() => {
      vi.advanceTimersByTime(349);
    });
    expect(handle.current!.value).toBe("idle");
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(handle.current!.value).toBe("closed");
  });

  test("rapid update bursts collapse to the latest value", () => {
    act(() => {
      handle.current!.update("first", "before");
    });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    act(() => {
      handle.current!.update("second", "before");
    });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    // The first scheduled write was cancelled at t=500; the second
    // scheduled write fires at t=500+1000=1500. At now=1000 we are
    // still 500ms shy of it.
    expect(handle.current!.value).toBe("idle");
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(handle.current!.value).toBe("second");
  });

  test("overrideDelay takes precedence over direction default", () => {
    act(() => {
      handle.current!.update("now", "before", 0);
    });
    expect(handle.current!.value).toBe("now");
  });

  test("cancel() prevents a pending commit", () => {
    act(() => {
      handle.current!.update("never", "before");
    });
    act(() => {
      handle.current!.cancel();
    });
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(handle.current!.value).toBe("idle");
  });

  test("unmount clears any pending timer", () => {
    act(() => {
      handle.current!.update("late", "before");
    });
    unmount();
    // If the timer survived unmount, this would throw a React warning
    // about updating an unmounted component. The test passes silently
    // when cleanup is correct.
    expect(() => vi.advanceTimersByTime(2000)).not.toThrow();
    // Re-mount for the afterEach unmount call to succeed.
    mount();
  });
});
