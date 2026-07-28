import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en" } }),
}));

vi.mock("@/utils", () => ({
  formatRelativeTime: (ms: number) => `relative:${ms}`,
  formatAbsoluteDateTime: (ms: number) => `absolute:${ms}`,
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

  test("renders relative time and reveals the absolute time on hover", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    // `ts` is in seconds; both labels are computed from milliseconds.
    act(() => root.render(<HumanizeTs ts={1000} />));
    expect(container.textContent).toContain("relative:1000000");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("absolute:1000000");

    act(() => root.unmount());
  });

  test("omits the tooltip when tooltip is false", async () => {
    vi.useFakeTimers();
    const { container, root } = mount();

    act(() => root.render(<HumanizeTs ts={1000} tooltip={false} />));
    expect(container.textContent).toContain("relative:1000000");

    await focus(container.querySelector("span"));

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent ?? "").not.toContain("absolute:");

    act(() => root.unmount());
  });
});
