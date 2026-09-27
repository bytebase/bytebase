import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useTimeReading } from "./useTimeReading";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const DAY_MS = 86_400_000;

// A boundary that names one instant and never changes again. Retiring to
// Infinity is the difference from a plain `() => deadlineMs`: it changes the
// effect's dependency, so the display drops its subscription and, if it was
// the last one, the clock disarms. Tests that need the clock quiet afterwards
// use this; tests that need a live subscription use the plain closure.
const onceAt = (changesAtMs: number) => () =>
  Date.now() < changesAtMs ? changesAtMs : Number.POSITIVE_INFINITY;

type ProbeSpec = {
  changesAtMs: () => number;
  read?: (input: number) => string;
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
    {
      read: spec.read ?? ((input: number) => `read:${input}`),
      nextChangeAt: spec.changesAtMs,
    },
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

// The clock's state is the page's: it has no reset, so a test that rewound the
// wall clock would hand the next one a clock stepped backward -- the very
// condition the clock exists to detect, and it would detect it. Time only
// moves forward here, as it does on a page: every test starts later than any
// clock reading in every test before it. A test that waits out the timer
// ceiling ends weeks after it began, and one that steps the clock back ends
// before it began, so the next start is taken from the later of the two.
let testStartMs = Date.UTC(2026, 2, 2, 12);

describe("useTimeReading", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(testStartMs);
  });

  afterEach(() => {
    for (const root of roots.splice(0)) {
      act(() => root.unmount());
    }
    // Read while the fake clock still holds where this test reached.
    testStartMs = Math.max(testStartMs, Date.now()) + DAY_MS;
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

  test("keeps the clock for the displays that remain, and stops for none", () => {
    const staying = mount([{ changesAtMs: () => Date.now() + 60_000 }]);
    expect(vi.getTimerCount()).toBe(1);

    // A nearer deadline re-arms: one timer for the lot, not one per arm.
    const leaving = mount([{ changesAtMs: () => Date.now() + 1_000 }]);
    expect(vi.getTimerCount()).toBe(1);

    act(() => leaving.root.unmount());
    expect(vi.getTimerCount()).toBe(1);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(staying.renders).toEqual([2]);

    act(() => staying.root.unmount());
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

  test("returns the value the reading gives now, and keeps it current", () => {
    const deadlineMs = Date.now() + 1_000;
    const { values } = mount([
      {
        read: () => (Date.now() < deadlineMs ? "before" : "after"),
        changesAtMs: onceAt(deadlineMs),
      },
    ]);
    expect(values).toEqual(["before"]);

    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(values).toEqual(["after"]);
  });

  test("still wakes a display when another mounts after the clock steps back", () => {
    // Arming re-reads the clock, so a display mounting in the window between
    // the step and the next check must not erase the evidence of the step.
    const deadlineMs = Date.now() + 5 * 60_000;
    const { renders } = mount([{ changesAtMs: onceAt(deadlineMs) }]);

    vi.setSystemTime(Date.now() - 10 * 60_000);
    mount([{ changesAtMs: onceAt(Date.now() + 30_000) }]);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders[0]).toBe(2);
  });

  test("keeps a display's own deadline through a backward step", () => {
    // The step invalidates the value, not the boundary: an absolute instant a
    // display already declared is still when it next changes.
    const deadlineMs = Date.now() + 30_000;
    const { renders } = mount([
      { changesAtMs: onceAt(deadlineMs) },
      // A neighbour on the second, so a tick lands while the clock is still
      // below the mark and the step is detected.
      { changesAtMs: () => Math.floor(Date.now() / 1_000) * 1_000 + 1_000 },
    ]);

    vi.setSystemTime(Date.now() - 5_000);
    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    const afterStep = renders[0];

    // Advance to the deadline it declared, and no further.
    act(() => {
      vi.advanceTimersByTime(deadlineMs - Date.now());
    });
    expect(renders[0]).toBe(afterStep + 1);
  });

  test("reads a clock that has not moved as forward, not backward", () => {
    // A boundary already past arms at no delay, so the tick lands in the very
    // millisecond the clock armed in. Standing still is not a step backward,
    // and waking the display that is not due for it would be a wake for
    // nothing.
    const { renders } = mount([
      { changesAtMs: () => Date.now() + 60_000 },
      { changesAtMs: () => Date.now() - 1 },
    ]);

    act(() => {
      vi.advanceTimersToNextTimer();
    });
    expect(renders[0]).toBe(1);
  });

  test("wakes the subscribed displays when the clock steps back, and only those", () => {
    // Each boundary was computed on the later clock; on the earlier one it can
    // be far off in either direction. A reading settled for good holds no
    // subscription, so the step cannot reach it -- the price of a settled
    // display costing nothing.
    const deadlineMs = Date.now() + 5 * 60_000;
    const { renders } = mount([
      { changesAtMs: () => Number.POSITIVE_INFINITY },
      { changesAtMs: () => deadlineMs },
    ]);

    vi.setSystemTime(Date.now() - 10 * 60_000);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders).toEqual([1, 2]);
  });

  test("handles one backward step once, not at every later check", () => {
    // The step lowers the mark it was detected against. Left standing, every
    // re-check until the clock climbs back reads as a fresh step, and an hour
    // of NTP correction becomes an hour of waking every display on the page.
    const deadlineMs = Date.now() + 5 * 60_000;
    const { renders } = mount([{ changesAtMs: () => deadlineMs }]);

    vi.setSystemTime(Date.now() - 10 * 60_000);
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders).toEqual([2]);

    // A second re-check, still below the clock the step was taken from.
    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(renders).toEqual([2]);
  });

  test("never stretches the wake gap past its bound after a clock step", () => {
    const firstMs = Date.now() + 1_000;
    mount([{ changesAtMs: onceAt(firstMs) }]);
    act(() => {
      vi.advanceTimersByTime(1_000);
    });

    vi.setSystemTime(Date.now() - 3_600_000);
    const secondMs = Date.now() + 100;
    const { renders } = mount([
      { changesAtMs: onceAt(secondMs) },
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
      { changesAtMs: onceAt(soonMs) },
    ]);
    act(() => {
      vi.advanceTimersByTime(20);
    });
    expect(renders).toEqual([2]);
  });
});
