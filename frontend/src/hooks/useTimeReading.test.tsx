import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useTimeReading } from "./useTimeReading";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const DAY_MS = 86_400_000;

type ProbeSpec = {
  changesAtMs: () => number;
  // An absent input has no reading.
  input?: number;
};

function Probe({
  spec,
  onRender,
}: {
  spec: ProbeSpec;
  onRender: (value: string | undefined) => void;
}) {
  const value = useTimeReading(
    { read: (input: number) => `read:${input}`, nextChangeAt: spec.changesAtMs },
    "input" in spec ? spec.input : 0
  );
  onRender(value);
  return null;
}

// Unmounted in afterEach so a failed assertion cannot leave a subscriber on
// the module-level clock for the next test.
const roots: ReturnType<typeof createRoot>[] = [];

const mount = (probes: ProbeSpec[]) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  const renders = probes.map(() => 0);
  const values: (string | undefined)[] = probes.map(() => undefined);
  act(() =>
    root.render(
      <>
        {probes.map((spec, index) => (
          <Probe
            // biome-ignore lint/suspicious/noArrayIndexKey: fixed probe list
            key={index}
            spec={spec}
            onRender={(value) => {
              renders[index] += 1;
              values[index] = value;
            }}
          />
        ))}
      </>
    )
  );
  return { root, renders, values };
};

describe("useTimeReading", () => {
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
    const { renders } = mount([{ changesAtMs: () => deadlineMs }]);

    act(() => {
      vi.advanceTimersByTime(999);
    });
    expect(renders).toEqual([1]);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(renders).toEqual([2]);
  });

  test("waits quietly for a deadline beyond the platform timer ceiling", () => {
    // setTimeout holds its delay in 32 bits; anything past ~24.8 days fires at
    // once. A date waiting for the year to turn is ordinary input.
    const deadlineMs = Date.now() + 90 * DAY_MS;
    const { renders } = mount([{ changesAtMs: () => deadlineMs }]);
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
  });

  test("wakes each due subscriber once and re-arms once for the lot", () => {
    const firstMs = Date.now() + 1_000;
    const secondMs = firstMs + 60_000;
    // Each subscriber, once woken, declares the next deadline — as a relative
    // label does when its bucket turns over.
    const probes = Array.from({ length: 50 }, () => ({
      changesAtMs: () => (Date.now() < firstMs ? firstMs : secondMs),
    }));
    const { renders } = mount(probes);
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");

    act(() => {
      vi.advanceTimersByTime(5_000);
    });
    expect(renders.every((count) => count === 2)).toBe(true);
    expect(setTimeoutSpy.mock.calls.length).toBeLessThanOrEqual(2);
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

  test("re-checks a woken subscriber whose boundary did not advance", () => {
    // A wall clock stepped back between the wake and its render, or a boundary
    // held stale by a memo, names the instant that just passed a second time.
    const deadlineMs = Date.now() + 1_000;
    const { renders } = mount([
      {
        changesAtMs: () =>
          Date.now() < deadlineMs ? deadlineMs : deadlineMs + 60_000,
      },
    ]);

    act(() => {
      vi.advanceTimersByTime(1_000);
      vi.setSystemTime(Date.now() - 2);
    });
    expect(renders).toEqual([2]);

    // Re-checked once a resync interval has passed on the wall clock.
    act(() => {
      vi.advanceTimersByTime(61_000);
    });
    expect(renders[0]).toBeGreaterThan(2);
  });

  test("does not re-wake a stuck subscriber on every neighbour's tick", () => {
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

  test("holds nothing for an absent input, and reads nothing", () => {
    const { values } = mount([
      { changesAtMs: () => Date.now() + 1_000, input: undefined },
    ]);
    expect(values).toEqual([undefined]);
    expect(vi.getTimerCount()).toBe(0);
  });

  test("evaluates the boundary before the value it schedules", () => {
    const calls: string[] = [];
    function OrderProbe() {
      useTimeReading(
        {
          read: () => calls.push("read"),
          nextChangeAt: () => {
            calls.push("boundary");
            return Number.POSITIVE_INFINITY;
          },
        },
        0
      );
      return null;
    }
    const root = createRoot(document.createElement("div"));
    roots.push(root);
    act(() => root.render(<OrderProbe />));
    expect(calls).toEqual(["boundary", "read"]);
  });

  test("returns the reading's current value, and keeps it current", () => {
    let value = "before";
    const deadlineMs = Date.now() + 1_000;
    const { values } = mount([
      {
        changesAtMs: () => {
          value = Date.now() < deadlineMs ? "before" : "after";
          return Date.now() < deadlineMs ? deadlineMs : Number.POSITIVE_INFINITY;
        },
      },
    ]);
    expect(values).toEqual(["read:0"]);
    expect(value).toBe("before");

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(value).toBe("after");
  });

  test("wakes every display when the wall clock steps backward", () => {
    // Each boundary was computed on the later clock; on the earlier one it can
    // be far off in either direction.
    const deadlineMs = Date.now() + 5 * 60_000;
    const { renders } = mount([{ changesAtMs: () => deadlineMs }]);

    vi.setSystemTime(Date.now() - 10 * 60_000);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders[0]).toBe(2);
  });

  test("never stretches the wake gap past its bound after a clock step", () => {
    const firstMs = Date.now() + 1_000;
    mount([{ changesAtMs: () => (Date.now() < firstMs ? firstMs : Number.POSITIVE_INFINITY) }]);
    act(() => {
      vi.advanceTimersByTime(1_000);
    });

    vi.setSystemTime(Date.now() - 3_600_000);
    const secondMs = Date.now() + 100;
    const { renders } = mount([
      { changesAtMs: () => (Date.now() < secondMs ? secondMs : Number.POSITIVE_INFINITY) },
    ]);
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(renders).toEqual([2]);
  });

  test("starts no wake gap from a re-check that woke nobody", () => {
    const farMs = Date.now() + 2 * 60_000;
    mount([{ changesAtMs: () => farMs }]);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    const soonMs = Date.now() + 10;
    const { renders } = mount([
      { changesAtMs: () => (Date.now() < soonMs ? soonMs : Number.POSITIVE_INFINITY) },
    ]);
    act(() => {
      vi.advanceTimersByTime(20);
    });
    expect(renders).toEqual([2]);
  });
});
