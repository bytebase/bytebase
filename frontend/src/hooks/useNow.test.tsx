import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useNow } from "./useNow";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const DAY_MS = 86_400_000;

function Probe({
  changesAtMs,
  onRender,
}: {
  changesAtMs: () => number | undefined;
  onRender: () => void;
}) {
  useNow(changesAtMs());
  onRender();
  return null;
}

// Unmounted in afterEach so a failed assertion cannot leave a subscriber on
// the module-level clock for the next test.
const roots: ReturnType<typeof createRoot>[] = [];

const mount = (probes: { changesAtMs: () => number | undefined }[]) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  const renders = probes.map(() => 0);
  act(() =>
    root.render(
      <>
        {probes.map((probe, index) => (
          <Probe
            changesAtMs={probe.changesAtMs}
            // biome-ignore lint/suspicious/noArrayIndexKey: fixed probe list
            key={index}
            onRender={() => {
              renders[index] += 1;
            }}
          />
        ))}
      </>
    )
  );
  return { root, renders };
};

describe("useNow", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    for (const root of roots.splice(0)) {
      act(() => root.unmount());
    }
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  test("wakes a subscriber at its deadline and not before", () => {
    const deadlineMs = Date.now() + 1_000;
    const { root, renders } = mount([{ changesAtMs: () => deadlineMs }]);

    act(() => {
      vi.advanceTimersByTime(999);
    });
    expect(renders).toEqual([1]);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(renders).toEqual([2]);

    act(() => root.unmount());
  });

  test("waits quietly for a deadline beyond the platform timer ceiling", () => {
    // setTimeout holds its delay in 32 bits; anything past ~24.8 days fires at
    // once. A date waiting for the year to turn is ordinary input.
    const deadlineMs = Date.now() + 90 * DAY_MS;
    const { root, renders } = mount([{ changesAtMs: () => deadlineMs }]);
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(setTimeoutSpy).not.toHaveBeenCalled();
    expect(renders).toEqual([1]);

    act(() => {
      vi.advanceTimersByTime(90 * DAY_MS);
    });
    expect(renders).toEqual([2]);

    act(() => root.unmount());
  });

  test("wakes each due subscriber once and re-arms once for the lot", () => {
    const firstMs = Date.now() + 1_000;
    const secondMs = firstMs + 60_000;
    // Each subscriber, once woken, declares the next deadline — as a relative
    // label does when its bucket turns over.
    const probes = Array.from({ length: 50 }, () => ({
      changesAtMs: () => (Date.now() < firstMs ? firstMs : secondMs),
    }));
    const { root, renders } = mount(probes);
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");

    act(() => {
      vi.advanceTimersByTime(5_000);
    });
    expect(renders.every((count) => count === 2)).toBe(true);
    expect(setTimeoutSpy.mock.calls.length).toBeLessThanOrEqual(2);

    act(() => root.unmount());
  });

  test("catches up when the wall clock passes a deadline the timer has not", () => {
    // Timers do not count time the machine sleeps, so after a night asleep the
    // wall clock can be past a deadline that the timer still waits hours for.
    const deadlineMs = Date.now() + 2 * 3_600_000;
    const { renders } = mount([{ changesAtMs: () => deadlineMs }]);

    vi.setSystemTime(Date.now() + 3 * 3_600_000);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders).toEqual([2]);
  });

  test("does not spin on a boundary that keeps naming a past instant", () => {
    const { renders } = mount([{ changesAtMs: () => Date.now() - 1 }]);

    // Step as a browser would, committing between timer turns.
    for (let step = 0; step < 100; step++) {
      act(() => {
        vi.advanceTimersByTime(10);
      });
    }
    expect(renders[0]).toBeLessThanOrEqual(6);
  });

  test("leaves a subscriber whose boundary did not advance to its last reading", () => {
    const stuckMs = Date.now() + 500;
    const { renders } = mount([
      { changesAtMs: () => stuckMs },
      // A neighbour that wakes every second.
      { changesAtMs: () => Math.floor(Date.now() / 1_000) * 1_000 + 1_000 },
    ]);

    for (let second = 0; second < 10; second++) {
      act(() => {
        vi.advanceTimersByTime(1_000);
      });
    }
    expect(renders[0]).toBe(2);
    expect(renders[1]).toBeGreaterThanOrEqual(10);
  });

  test("releases the timer when the last subscriber leaves", () => {
    const { root } = mount([{ changesAtMs: () => Date.now() + 1_000 }]);
    expect(vi.getTimerCount()).toBe(1);

    act(() => root.unmount());
    expect(vi.getTimerCount()).toBe(0);
  });

  test("holds nothing for a display that does not vary with time", () => {
    const { root } = mount([{ changesAtMs: () => undefined }]);
    expect(vi.getTimerCount()).toBe(0);

    act(() => root.unmount());
  });
});
