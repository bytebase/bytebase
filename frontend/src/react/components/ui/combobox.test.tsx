import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { Combobox } from "./combobox";
import { Dialog, DialogContent } from "./dialog";
import { LAYER_BACKDROP_CLASS, LAYER_SURFACE_CLASS } from "./layer";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Combobox", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("mounts a portaled dropdown into the overlay surface layer above the dialog surface", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <Dialog open>
          <DialogContent className="dialog-surface">
            <div>Dialog body</div>
            <Combobox
              className="test-combobox"
              portal
              value=""
              onChange={() => {}}
              options={[{ value: "alpha", label: "Alpha" }]}
              placeholder="Pick one"
            />
          </DialogContent>
        </Dialog>
      );
    });

    const overlayRoot = document.getElementById("bb-react-layer-overlay");
    expect(overlayRoot).toBeInstanceOf(HTMLDivElement);

    const trigger = overlayRoot?.querySelector(".test-combobox > div");
    expect(trigger).toBeInstanceOf(HTMLDivElement);

    act(() => {
      trigger?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const dialogSurface = overlayRoot?.querySelector(".dialog-surface");
    expect(dialogSurface).toBeInstanceOf(HTMLDivElement);
    expect(dialogSurface?.className).toContain(LAYER_SURFACE_CLASS);

    expect(overlayRoot?.innerHTML).toContain(LAYER_BACKDROP_CLASS);

    const dropdown = overlayRoot?.querySelector(
      "div.bg-background.border.border-control-border.rounded-sm.shadow-lg.overflow-hidden"
    ) as HTMLDivElement | null;
    expect(dropdown).toBeInstanceOf(HTMLDivElement);
    expect(dropdown?.textContent).toContain("Alpha");
    expect(dropdown?.className).toContain(LAYER_SURFACE_CLASS);
    expect(overlayRoot?.lastElementChild).toBe(dropdown);

    act(() => {
      root.unmount();
    });
  });
});
