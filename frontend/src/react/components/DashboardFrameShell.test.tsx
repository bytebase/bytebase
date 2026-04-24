import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { DashboardFrameShell } from "./DashboardFrameShell";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("DashboardFrameShell", () => {
  test("reports stable banner and body targets", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onReady = vi.fn();

    act(() => {
      root.render(<DashboardFrameShell onReady={onReady} />);
    });

    expect(onReady).toHaveBeenCalled();
    const targets = onReady.mock.lastCall?.[0];
    expect(targets.banner).toBeInstanceOf(HTMLDivElement);
    expect(targets.body).toBeInstanceOf(HTMLDivElement);
    expect(container.querySelector(".h-screen")).not.toBeNull();

    act(() => {
      root.unmount();
    });
    container.remove();
  });
});
