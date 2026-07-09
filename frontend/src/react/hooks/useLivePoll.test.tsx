import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useLivePoll } from "./useLivePoll";

describe("useLivePoll", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test("does not start a new call while the previous is still in flight", async () => {
    let concurrent = 0;
    let maxConcurrent = 0;
    const resolvers: Array<() => void> = [];
    // Each call stays pending until we resolve it — simulating a fetch slower
    // than the poll interval.
    const fn = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          concurrent += 1;
          maxConcurrent = Math.max(maxConcurrent, concurrent);
          resolvers.push(() => {
            concurrent -= 1;
            resolve();
          });
        })
    );

    renderHook(() => useLivePoll(true, 5000, fn));

    // First tick fires after the interval; its call stays in flight.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(fn).toHaveBeenCalledTimes(1);

    // Long past several intervals, no second call starts while the first is
    // pending — a fixed setInterval would have fired several overlapping calls.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(20000);
    });
    expect(fn).toHaveBeenCalledTimes(1);
    expect(maxConcurrent).toBe(1);

    // Once it settles, the next tick is scheduled one interval later.
    await act(async () => {
      resolvers[0]();
      await Promise.resolve();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(fn).toHaveBeenCalledTimes(2);
  });

  test("stops the loop when disabled and on unmount", async () => {
    const fn = vi.fn().mockResolvedValue(undefined);
    const { rerender, unmount } = renderHook(
      ({ enabled }: { enabled: boolean }) => useLivePoll(enabled, 1000, fn),
      { initialProps: { enabled: true } }
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(fn).toHaveBeenCalledTimes(1);

    // Disabling clears the loop.
    rerender({ enabled: false });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(fn).toHaveBeenCalledTimes(1);
    unmount();
  });
});
