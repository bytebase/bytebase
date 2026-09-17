import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en" } }),
}));

const formatters = vi.hoisted(() => ({
  // Age-based so a label that never re-renders is visibly distinguishable from
  // one that keeps up with the clock.
  formatQueueTime: (ms: number) => `queue:${Date.now() - ms}`,
  formatRelativeTime: (ms: number) => `relative:${Date.now() - ms}`,
  formatCompactDateTime: (ms: number) => `compact:${ms}`,
  formatOperationalDateTime: (ms: number) => `operational:${ms}`,
  formatAbsoluteDateTime: vi.fn((ms: number) => `absolute:${ms}`),
  // Distinct offsets, so a reading scheduled on the other reading's boundary
  // wakes at the wrong time and shows it.
  nextQueueTimeChangeAt: (ms: number) => ms + 60_000,
  nextRelativeTimeChangeAt: (ms: number) => ms + 45_000,
}));

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

    // The relative reading's own boundary is 45s after the timestamp.
    act(() => {
      vi.advanceTimersByTime(15_000);
    });
    expect(overlayText()).toContain("relative:45100");
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

    // The work-queue reading's own boundary is 60s after the timestamp.
    act(() => {
      vi.advanceTimersByTime(30_000);
    });
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
