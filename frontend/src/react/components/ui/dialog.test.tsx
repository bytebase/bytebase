import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Dialog, DialogContent } from "./dialog";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "./layer";

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

  test("pads the popup by default and lets callers override sizing", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <>
          <Dialog open>
            <DialogContent className="default-surface">Default</DialogContent>
          </Dialog>
          <Dialog open>
            <DialogContent className="custom-surface max-w-md p-0">
              Custom
            </DialogContent>
          </Dialog>
        </>
      );
    });

    const overlayRoot = document.getElementById("bb-react-layer-overlay");
    const surfaces = Array.from(
      overlayRoot?.querySelectorAll(".default-surface, .custom-surface") ?? []
    ) as HTMLElement[];
    const defaultSurface = surfaces.find((el) =>
      el.className.includes("default-surface")
    );
    const customSurface = surfaces.find((el) =>
      el.className.includes("custom-surface")
    );

    // Safe defaults: bare <DialogContent> must be padded.
    expect(defaultSurface?.className).toContain("p-6");

    // Caller overrides must fully replace the defaults via tailwind-merge.
    // The default max-w must be a single non-responsive utility — a 2xl:
    // variant would survive the merge and clobber the caller's max-w-* on
    // wide screens (BYT-9699).
    expect(customSurface?.className).toContain("max-w-md");
    expect(customSurface?.className).toContain("p-0");
    expect(customSurface?.className).not.toContain("p-6");
    expect(customSurface?.className).not.toMatch(/2xl:max-w/);
    expect(customSurface?.className).not.toContain("max-w-[max(");

    await act(async () => {
      root.unmount();
    });
  });

  test("keeps the agent layer visible to assistive tech when an app dialog opens", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const agentRoot = getLayerRoot("agent");
    const agentButton = document.createElement("button");

    agentButton.textContent = "Agent action";
    agentRoot.appendChild(agentButton);

    await act(async () => {
      root.render(
        <Dialog open>
          <DialogContent>App dialog</DialogContent>
        </Dialog>
      );
    });

    await vi.waitFor(() => {
      expect(agentRoot.getAttribute("aria-hidden")).toBeNull();
      expect(agentRoot.getAttribute("inert")).toBeNull();
      expect(agentRoot.getAttribute("data-base-ui-inert")).toBeNull();
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("keeps the agent layer visible when watermark content is mounted", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const agentRoot = getLayerRoot("agent");
    const watermarkRoot = getLayerRoot("watermark");
    const agentButton = document.createElement("button");
    const watermarkLayer = document.createElement("div");

    agentButton.textContent = "Agent action";
    watermarkLayer.setAttribute("aria-hidden", "true");
    agentRoot.appendChild(agentButton);
    watermarkRoot.appendChild(watermarkLayer);

    await act(async () => {
      root.render(
        <Dialog open>
          <DialogContent>App dialog</DialogContent>
        </Dialog>
      );
    });

    await vi.waitFor(() => {
      expect(agentRoot.getAttribute("aria-hidden")).toBeNull();
      expect(agentRoot.getAttribute("inert")).toBeNull();
      expect(agentRoot.getAttribute("data-base-ui-inert")).toBeNull();
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("keeps the critical layer visible when watermark content is mounted", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const criticalRoot = getLayerRoot("critical");
    const watermarkRoot = getLayerRoot("watermark");
    const criticalButton = document.createElement("button");
    const watermarkLayer = document.createElement("div");

    criticalButton.textContent = "Critical action";
    watermarkLayer.setAttribute("aria-hidden", "true");
    criticalRoot.appendChild(criticalButton);
    watermarkRoot.appendChild(watermarkLayer);

    await act(async () => {
      root.render(
        <Dialog open>
          <DialogContent>App dialog</DialogContent>
        </Dialog>
      );
    });

    await vi.waitFor(() => {
      expect(criticalRoot.getAttribute("aria-hidden")).toBeNull();
      expect(criticalRoot.getAttribute("inert")).toBeNull();
      expect(criticalRoot.getAttribute("data-base-ui-inert")).toBeNull();
    });

    await act(async () => {
      root.unmount();
    });
  });
});
