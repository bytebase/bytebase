import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en" } }),
}));

vi.mock("@/utils", () => ({
  // Age-based so a label that never re-renders is visibly distinguishable from
  // one that keeps up with the clock.
  formatQueueTime: (ms: number) => `queue:${Date.now() - ms}`,
  nextRelativeChangeAt: (ms: number) => ms + 60_000,
  formatCompactDateTime: (ms: number) => `compact:${ms}`,
  formatOperationalDateTime: (ms: number) => `operational:${ms}`,
  formatAbsoluteDateTime: (ms: number) => `absolute:${ms}`,
  formatRelativeTime: (ms: number) => `relative:${ms}`,
}));

import { HumanizeTs } from "./HumanizeTs";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mount = () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  return { container, root: createRoot(container) };
};

const focus = async (el: Element | null) => {
  await act(async () => {
    el?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    vi.advanceTimersByTime(100);
  });
};

describe("HumanizeTs", () => {
  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  test("renders the work-queue form by default and the full time on hover", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    // `ts` is in seconds; both labels are computed from milliseconds.
    act(() => root.render(<HumanizeTs ts={1000} />));
    expect(container.textContent).toContain("queue:");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("absolute:1000000");

    act(() => root.unmount());
  });

  test("renders the compact history tier and keeps full precision on hover", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    act(() => root.render(<HumanizeTs mode="compact" ts={1000} />));
    expect(container.textContent).toContain("compact:1000000");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("absolute:1000000");

    act(() => root.unmount());
  });

  test("names the timezone inline for a time the reader will act on", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    act(() => root.render(<HumanizeTs mode="operational" ts={1000} />));
    expect(container.textContent).toContain("operational:1000000");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("absolute:1000000");

    act(() => root.unmount());
  });

  test("inverts the tooltip on a full cell, where age is the missing reading", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    act(() => root.render(<HumanizeTs mode="datetime" ts={1000} />));
    expect(container.textContent).toContain("absolute:1000000");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("relative:1000000");

    act(() => root.unmount());
  });

  test("omits the tooltip when tooltip is false", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    act(() => root.render(<HumanizeTs ts={1000} tooltip={false} />));
    expect(container.textContent).toContain("queue:");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent ?? "").not.toContain("absolute:");

    act(() => root.unmount());
  });
});

describe("HumanizeTs freshness", () => {
  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  test("keeps a mounted work-queue label up with the clock", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
    const { container, root } = mount();
    const ts = Math.floor(Date.now() / 1000) - 30;

    act(() => root.render(<HumanizeTs ts={ts} tooltip={false} />));
    expect(container.textContent).toBe("queue:30000");

    act(() => {
      vi.advanceTimersByTime(60_000);
    });
    expect(container.textContent).toBe("queue:90000");

    act(() => root.unmount());
  });

  test.each(["compact", "datetime", "operational"] as const)(
    "puts no %s display on the shared clock",
    (mode) => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
      const { root } = mount();
      const ts = Math.floor(Date.now() / 1000) - 30;

      act(() => root.render(<HumanizeTs mode={mode} ts={ts} tooltip={false} />));
      expect(vi.getTimerCount()).toBe(0);

      act(() => root.unmount());
    }
  );

  test("releases the clock when the last subscriber unmounts", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
    const { root } = mount();
    const ts = Math.floor(Date.now() / 1000) - 30;

    act(() => root.render(<HumanizeTs ts={ts} tooltip={false} />));
    expect(vi.getTimerCount()).toBe(1);

    act(() => root.unmount());
    expect(vi.getTimerCount()).toBe(0);
  });
});
