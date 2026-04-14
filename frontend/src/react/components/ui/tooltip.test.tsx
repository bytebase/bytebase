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

  test("applies the overlay z-layer to the tooltip positioner", async () => {
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

    expect(document.body.textContent).toContain("Tip content");
    const positioner = document.body.querySelector('[role="presentation"]');
    expect(positioner?.className).toContain("z-50");

    act(() => {
      root.unmount();
    });
  });
});
