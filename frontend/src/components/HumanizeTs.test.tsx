import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en" } }),
}));

const formatters = vi.hoisted(() => {
  // A reading whose label is its age, so a display that was not re-rendered
  // shows a stale age, and whose boundary is a fixed offset after the timestamp.
  // The two offsets differ, so a label scheduled on the other reading's
  // boundary re-renders at the wrong second and shows it.
  const ageReading = (prefix: string, boundaryOffsetMs: number) => ({
    read: (ms: number) => `${prefix}:${Date.now() - ms}`,
    nextChangeAt: (ms: number) =>
      Date.now() < ms + boundaryOffsetMs
        ? ms + boundaryOffsetMs
        : Number.POSITIVE_INFINITY,
  });
  return {
    queueTimeReading: ageReading("queue", 60_000),
    relativeTimeReading: ageReading("relative", 45_000),
    formatCompactDateTime: (ms: number) => `compact:${ms}`,
    formatOperationalDateTime: (ms: number) => `operational:${ms}`,
    formatAbsoluteDateTime: vi.fn((ms: number) => `absolute:${ms}`),
  };
});

vi.mock("@/utils/datetime", () => formatters);

import { HumanizeTs } from "./HumanizeTs";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const roots: ReturnType<typeof createRoot>[] = [];

const mount = () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  return { container, root };
};

const openTooltip = async (el: Element | null) => {
  await act(async () => {
    el?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    vi.advanceTimersByTime(100);
  });
};

const overlayText = () =>
  document.getElementById("bb-react-layer-overlay")?.textContent ?? "";

// One act per second, as a browser commits between timer turns; a single long
// act would commit only once, at its end, and hide when the render happened.
const advanceSeconds = (seconds: number) => {
  for (let second = 0; second < seconds; second++) {
    act(() => {
      vi.advanceTimersByTime(1_000);
    });
  }
};

const secondsAgo = (seconds: number) =>
  Math.floor(Date.now() / 1000) - seconds;

describe("HumanizeTs", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
    formatters.formatAbsoluteDateTime.mockClear();
  });

  afterEach(() => {
    for (const root of roots.splice(0)) {
      act(() => root.unmount());
    }
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  test.each([
    [undefined, "queue:"],
    ["compact", "compact:"],
    ["operational", "operational:"],
  ] as const)(
    "renders the %s reading and restores the full time on hover",
    async (mode, prefix) => {
      const { container, root } = mount();
      act(() => root.render(<HumanizeTs mode={mode} ts={1000} />));
      expect(container.textContent).toContain(prefix);

      await openTooltip(container.firstElementChild);
      expect(overlayText()).toContain("absolute:1000000");
    }
  );

  test("builds the full time only once the tooltip opens", async () => {
    const { container, root } = mount();
    act(() => root.render(<HumanizeTs ts={1000} />));
    expect(formatters.formatAbsoluteDateTime).not.toHaveBeenCalled();

    await openTooltip(container.firstElementChild);
    expect(formatters.formatAbsoluteDateTime).toHaveBeenCalledWith(1000000);
  });

  test("renders one element, so layout classes reach the box that lays out", () => {
    const { container, root } = mount();
    act(() =>
      root.render(<HumanizeTs className="block truncate" ts={1000} />)
    );
    expect(container.childElementCount).toBe(1);
    expect(container.firstElementChild?.className).toContain("block truncate");
    expect(container.firstElementChild?.childElementCount).toBe(0);
  });

  test("offers the age on a full cell, and keeps it counting while open", async () => {
    const { container, root } = mount();
    act(() => root.render(<HumanizeTs mode="datetime" ts={secondsAgo(30)} />));
    expect(container.textContent).toContain("absolute:");

    await openTooltip(container.firstElementChild);
    // The tooltip opens 100ms after focus, which the age already reflects.
    expect(overlayText()).toContain("relative:30100");

    // The relative reading's boundary is 45s after the timestamp: the age holds
    // until then and moves on at it.
    advanceSeconds(14);
    expect(overlayText()).toContain("relative:30100");
    advanceSeconds(1);
    expect(overlayText()).toContain("relative:45100");
  });

  test("shows nothing for a time that is not a time", () => {
    // An unvalidated string reaching `new Date(...)` gives NaN, and Intl throws
    // on it; the row should lose its timestamp, not the page its subtree.
    const { container, root } = mount();
    act(() => root.render(<HumanizeTs ts={Number.NaN} />));
    expect(container.textContent).toBe("");
  });

  test("omits the tooltip when tooltip is false", async () => {
    const { container, root } = mount();
    act(() => root.render(<HumanizeTs ts={1000} tooltip={false} />));
    expect(container.textContent).toContain("queue:");

    await openTooltip(container.firstElementChild);
    expect(overlayText()).not.toContain("absolute:");
  });

  test("keeps a mounted work-queue label up with the clock", () => {
    const { container, root } = mount();
    act(() => root.render(<HumanizeTs ts={secondsAgo(30)} tooltip={false} />));
    expect(container.textContent).toBe("queue:30000");

    // The work-queue reading's boundary is 60s after the timestamp.
    advanceSeconds(29);
    expect(container.textContent).toBe("queue:30000");
    advanceSeconds(1);
    expect(container.textContent).toBe("queue:60000");
  });

  test.each(["compact", "datetime", "operational"] as const)(
    "puts no %s display on the shared clock",
    (mode) => {
      const { root } = mount();
      act(() =>
        root.render(<HumanizeTs mode={mode} ts={secondsAgo(30)} tooltip={false} />)
      );
      expect(vi.getTimerCount()).toBe(0);
    }
  );
});
