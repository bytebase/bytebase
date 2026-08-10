import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Combobox } from "./combobox";
import { Dialog, DialogContent } from "./dialog";
import { LAYER_BACKDROP_CLASS, LAYER_SURFACE_CLASS } from "./layer";
import { overlaySurfaceClassName } from "./styles.stylex";

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

    const dropdown = Array.from(
      overlayRoot?.querySelectorAll("div") ?? []
    ).find(
      (element) =>
        element.className.includes(overlaySurfaceClassName) &&
        element.textContent?.includes("Alpha")
    ) as HTMLDivElement | undefined;
    expect(dropdown).toBeInstanceOf(HTMLDivElement);
    expect(dropdown?.textContent).toContain("Alpha");
    expect(dropdown?.className).toContain(overlaySurfaceClassName);
    expect(dropdown?.className).toContain(LAYER_SURFACE_CLASS);
    expect(overlayRoot?.lastElementChild).toBe(dropdown);
    expect(dropdown?.querySelector("input")?.style.paddingInlineStart).toBe(
      "2rem"
    );
    const option = Array.from(dropdown?.querySelectorAll("button") ?? []).find(
      (button) => button.textContent?.includes("Alpha")
    );
    expect(option?.parentElement?.parentElement?.className).toContain("px-2");

    act(() => {
      root.unmount();
    });
  });

  test("renders and invokes the load-more action", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onLoadMore = vi.fn();

    act(() => {
      root.render(
        <Combobox
          value=""
          onChange={() => {}}
          options={[{ value: "alpha", label: "Alpha" }]}
          hasMore
          onLoadMore={onLoadMore}
        />
      );
    });

    act(() => {
      container.firstElementChild?.firstElementChild
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.textContent).toContain("Load more");

    const loadMoreButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "Load more"
    );
    act(() => loadMoreButton?.click());
    expect(onLoadMore).toHaveBeenCalledOnce();

    act(() => {
      root.unmount();
    });
  });

  test("passes the search query to custom option renderers", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <Combobox
          value=""
          onChange={() => {}}
          options={[
            {
              value: "bytebase",
              label: "bytebase",
              render: (keyword) => (
                <span data-testid="custom-option">{keyword}</span>
              ),
            },
          ]}
        />
      );
    });

    act(() => {
      container.firstElementChild?.firstElementChild
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    const input = container.querySelector("input");
    act(() => {
      if (!input) return;
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "byte");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(
      container.querySelector('[data-testid="custom-option"]')?.textContent
    ).toBe("byte");

    act(() => root.unmount());
  });
});
