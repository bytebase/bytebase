import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Dialog, DialogContent } from "./dialog";
import { LAYER_BACKDROP_CLASS, LAYER_SURFACE_CLASS } from "./layer";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Dialog", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders a child backdrop above the parent surface within the overlay layer", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <>
          <Dialog open>
            <DialogContent className="parent-surface">Parent</DialogContent>
          </Dialog>
          <Dialog open>
            <DialogContent className="child-surface">Child</DialogContent>
          </Dialog>
        </>
      );
    });

    const overlayRoot = document.getElementById("bb-react-layer-overlay");
    expect(overlayRoot).toBeInstanceOf(HTMLDivElement);

    await vi.waitFor(() => {
      const overlayElements = Array.from(
        overlayRoot?.querySelectorAll("*") ?? []
      );
      const backdrops = overlayElements.filter(
        (child): child is HTMLElement =>
          child instanceof HTMLElement &&
          child.className.includes(LAYER_BACKDROP_CLASS) &&
          child.className.includes("bg-overlay/50")
      );
      expect(backdrops).toHaveLength(2);

      const parentSurface = overlayElements.find(
        (child): child is HTMLElement =>
          child instanceof HTMLElement &&
          child.className.includes("parent-surface")
      );
      const childSurface = overlayElements.find(
        (child): child is HTMLElement =>
          child instanceof HTMLElement &&
          child.className.includes("child-surface")
      );
      expect(parentSurface).toBeInstanceOf(HTMLDivElement);
      expect(childSurface).toBeInstanceOf(HTMLDivElement);
      expect(parentSurface?.className).toContain(LAYER_SURFACE_CLASS);
      expect(childSurface?.className).toContain(LAYER_SURFACE_CLASS);

      expect(
        backdrops[0].compareDocumentPosition(parentSurface as Node) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
      expect(
        (parentSurface as Node).compareDocumentPosition(backdrops[1]) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
      expect(
        backdrops[1].compareDocumentPosition(childSurface as Node) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
    });

    await act(async () => {
      root.unmount();
    });
  });
});
