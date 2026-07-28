import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { usePlanPolling } from "./usePlanPolling";

describe("usePlanPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0.5);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  test("backs off the idle base interval from 1s to 16s", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "idle", refreshState })
    );

    for (const intervalMs of [1000, 2000, 4000, 8000, 16000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(intervalMs - 1);
      });
      expect(refreshState).toHaveBeenCalledTimes(
        [1000, 2000, 4000, 8000, 16000].indexOf(intervalMs)
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });
    }
    expect(refreshState).toHaveBeenCalledTimes(5);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(16000);
    });
    expect(refreshState).toHaveBeenCalledTimes(6);
  });

  test("backs off the active base interval from 500ms to 4s", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    for (const intervalMs of [500, 1000, 2000, 4000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(intervalMs - 1);
      });
      expect(refreshState).toHaveBeenCalledTimes(
        [500, 1000, 2000, 4000].indexOf(intervalMs)
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });
    }
    expect(refreshState).toHaveBeenCalledTimes(4);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4000);
    });
    expect(refreshState).toHaveBeenCalledTimes(5);
  });

  test("applies scaled jitter to the active 500ms floor", async () => {
    vi.mocked(Math.random).mockReturnValue(0);
    const refreshState = vi.fn().mockResolvedValue(undefined);
    const { unmount } = renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(374);
    });
    expect(refreshState).not.toHaveBeenCalled();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    unmount();

    vi.mocked(Math.random).mockReturnValue(0.999999);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(624);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("does not feed a jittered wait back into the next base interval", async () => {
    vi.mocked(Math.random)
      .mockReturnValueOnce(0.999999)
      .mockReturnValueOnce(0)
      .mockReturnValue(0.5);
    const refreshState = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    // 500ms base + 125ms jitter.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(625);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    // The next base is still 1s, independently jittered down by 250ms. If the
    // first 625ms wait fed back into backoff, this tick would arrive at 1s.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(749);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("restarts at the 500ms floor when active mode begins", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ mode }: { mode: "active" | "idle" }) =>
        usePlanPolling({ enabled: true, mode, refreshState }),
      {
        initialProps: { mode: "idle" } as { mode: "active" | "idle" },
      }
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    rerender({ mode: "active" });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("restarts active polling when observed state changes", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ resetKey }: { resetKey: string }) =>
        usePlanPolling({
          enabled: true,
          mode: "active",
          refreshState,
          resetKey,
        }),
      { initialProps: { resetKey: "pending" } }
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    rerender({ resetKey: "running" });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("does not overlap slow refreshes", async () => {
    let resolveRefresh: () => void = () => {};
    const refreshState = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            resolveRefresh = resolve;
          })
      )
      .mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveRefresh();
      await Promise.resolve();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("restart resets idle backoff to the 1s floor", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      usePlanPolling({ enabled: true, mode: "idle", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    act(() => result.current.restart());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("continues polling after a transient refresh failure", async () => {
    const refreshState = vi
      .fn()
      .mockRejectedValueOnce(new Error("refresh failed"))
      .mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("pauses while hidden and restarts at the active floor when visible", async () => {
    const hidden = vi
      .spyOn(document, "hidden", "get")
      .mockReturnValue(true);
    const refreshState = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "active", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(refreshState).not.toHaveBeenCalled();

    hidden.mockReturnValue(false);
    act(() => document.dispatchEvent(new Event("visibilitychange")));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499);
    });
    expect(refreshState).not.toHaveBeenCalled();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
  });

  test("restarts idle backoff at the floor after reconnecting", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      usePlanPolling({ enabled: true, mode: "idle", refreshState })
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    act(() => window.dispatchEvent(new Event("online")));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("defers a state-change restart until the in-flight refresh settles", async () => {
    let resolveRefresh: () => void = () => {};
    const refreshState = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            resolveRefresh = resolve;
          })
      )
      .mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ resetKey }: { resetKey: string }) =>
        usePlanPolling({
          enabled: true,
          mode: "active",
          refreshState,
          resetKey,
        }),
      { initialProps: { resetKey: "pending" } }
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    rerender({ resetKey: "running" });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveRefresh();
      await Promise.resolve();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(2);
  });

  test("returns to the 1s idle floor when active work finishes", async () => {
    const refreshState = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ mode, resetKey }: { mode: "active" | "idle"; resetKey?: string }) =>
        usePlanPolling({ enabled: true, mode, refreshState, resetKey }),
      {
        initialProps: {
          mode: "active",
          resetKey: "running",
        } as { mode: "active" | "idle"; resetKey?: string },
      }
    );

    for (const intervalMs of [500, 1000, 2000, 4000]) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(intervalMs);
      });
    }
    expect(refreshState).toHaveBeenCalledTimes(4);

    rerender({ mode: "idle", resetKey: undefined });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(refreshState).toHaveBeenCalledTimes(4);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(refreshState).toHaveBeenCalledTimes(5);
  });

  test("does not reschedule when disabled during an in-flight refresh", async () => {
    let resolveRefresh: () => void = () => {};
    const refreshState = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveRefresh = resolve;
        })
    );
    const { rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) =>
        usePlanPolling({ enabled, mode: "active", refreshState }),
      { initialProps: { enabled: true } }
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);

    rerender({ enabled: false });
    await act(async () => {
      resolveRefresh();
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(refreshState).toHaveBeenCalledTimes(1);
  });

  test("uses the latest refresh callback without resetting backoff", async () => {
    const firstRefresh = vi.fn().mockResolvedValue(undefined);
    const latestRefresh = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ refreshState }: { refreshState: () => Promise<void> }) =>
        usePlanPolling({ enabled: true, mode: "active", refreshState }),
      { initialProps: { refreshState: firstRefresh } }
    );

    rerender({ refreshState: latestRefresh });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(firstRefresh).not.toHaveBeenCalled();
    expect(latestRefresh).toHaveBeenCalledTimes(1);

    rerender({ refreshState: firstRefresh });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(firstRefresh).not.toHaveBeenCalled();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(firstRefresh).toHaveBeenCalledTimes(1);
  });
});
