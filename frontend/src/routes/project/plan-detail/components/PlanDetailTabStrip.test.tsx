import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { PlanDetailTabItem } from "./PlanDetailTabStrip";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("PlanDetailTabItem", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    document.body.removeChild(container);
  });

  it("bounds Change tab width without making every tab equal", () => {
    act(() => {
      root.render(
        <PlanDetailTabItem
          action={<span data-testid="action">Action</span>}
          boundedWidth
          onSelect={() => {}}
          selected
        >
          <span>orders</span>
        </PlanDetailTabItem>
      );
    });

    const tab = container.firstElementChild;
    const button = tab?.querySelector("button");
    expect(tab).toHaveClass("min-w-40", "max-w-64");
    expect(button).toHaveClass("min-w-0", "flex-1");
    expect(button).not.toHaveClass("w-full");
  });
});
