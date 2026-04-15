import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Tooltip } from "./tooltip";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Tooltip", () => {
  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  test("mounts tooltip content into the overlay layer root", async () => {
    vi.useFakeTimers();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <Tooltip content="Tip content">
          <button type="button">Trigger</button>
        </Tooltip>
      );
    });

    const trigger = container.querySelector("button");
    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });

    const overlayRoot = document.getElementById("bb-react-layer-overlay");
    expect(overlayRoot).toBeInstanceOf(HTMLDivElement);
    expect(overlayRoot?.textContent).toContain("Tip content");

    act(() => {
      root.unmount();
    });
  });
});
