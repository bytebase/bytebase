import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("@/utils", () => ({
  minmax: (value: number, min: number, max: number) =>
    Math.max(min, Math.min(max, value)),
}));

import { usePolling } from "./usePolling";

describe("usePolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    // Math.random = 0.5 makes the jitter term exactly zero, so tick intervals
    // are the deterministic backoff sequence 1000, 2000, 4000, ...
    vi.spyOn(Math, "random").mockReturnValue(0.5);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  test("polls with a growing backoff", async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    renderHook(() => usePolling({ enabled: true, refreshState: refresh }));

    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(refresh).toHaveBeenCalledTimes(1);

    await act(() => vi.advanceTimersByTimeAsync(2000));
    expect(refresh).toHaveBeenCalledTimes(2);

    await act(() => vi.advanceTimersByTimeAsync(4000));
    expect(refresh).toHaveBeenCalledTimes(3);
  });

  test("restart resets the backoff to the minimum interval", async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      usePolling({ enabled: true, refreshState: refresh })
    );

    // Two ticks so the next scheduled interval has grown to 4000.
    await act(() => vi.advanceTimersByTimeAsync(1000));
    await act(() => vi.advanceTimersByTimeAsync(2000));
    expect(refresh).toHaveBeenCalledTimes(2);

    act(() => result.current.restart());

    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(refresh).toHaveBeenCalledTimes(3);
  });

  test("a tick in flight during restart does not clobber the restarted interval", async () => {
    // Regression: after a user action (e.g. task rerun) restart() schedules the
    // next poll at the minimum interval. A tick whose refresh was still in
    // flight at that moment must not reschedule with its grown backoff when it
    // resolves — that would silently replace the ~1s follow-up with a multi-
    // second wait for the status transition.
    let resolveRefresh: () => void = () => undefined;
    const refresh = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveRefresh = resolve;
        })
    );
    const { result } = renderHook(() =>
      usePolling({ enabled: true, refreshState: refresh })
    );

    // First tick fires; its refresh stays in flight.
    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(refresh).toHaveBeenCalledTimes(1);

    // User action: restart the poller, then let the stale tick finish.
    act(() => result.current.restart());
    await act(async () => {
      resolveRefresh();
    });

    // The restarted minimum-interval tick must still fire ~1s after restart.
    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(refresh).toHaveBeenCalledTimes(2);
  });
});
