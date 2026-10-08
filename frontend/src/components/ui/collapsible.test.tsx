import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { Collapsible, CollapsiblePanel, CollapsibleTrigger } from "./collapsible";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    rerender: (next: ReactElement) => act(() => root.render(next)),
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const tree = (props: Record<string, unknown>) =>
  createElement(
    Collapsible,
    props,
    createElement(CollapsibleTrigger, null, "Toggle"),
    createElement(CollapsiblePanel, null, "Panel body")
  );

const trigger = (container: HTMLElement) => {
  const el = container.querySelector("button");
  if (!el) throw new Error("trigger not found");
  return el;
};

describe("Collapsible", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  // The whole point of using the primitive rather than a `useState` and a div:
  // the trigger has to announce the panel it controls, and a hand-rolled
  // disclosure routinely ships without it.
  test("the trigger carries the expanded state assistive tech reads", () => {
    const { container, rerender, unmount } = renderIntoContainer(
      tree({ open: false, onOpenChange: () => undefined })
    );
    expect(trigger(container).getAttribute("aria-expanded")).toBe("false");

    rerender(tree({ open: true, onOpenChange: () => undefined }));
    expect(trigger(container).getAttribute("aria-expanded")).toBe("true");
    // The panel only exists while open, so the association is asserted there.
    const controls = trigger(container).getAttribute("aria-controls");
    expect(controls).toBeTruthy();
    expect(container.querySelector(`#${controls}`)?.textContent).toContain(
      "Panel body"
    );
    unmount();
  });

  test("the panel is absent while closed and rendered while open", () => {
    const { container, rerender, unmount } = renderIntoContainer(
      tree({ open: false, onOpenChange: () => undefined })
    );
    expect(container.textContent).not.toContain("Panel body");

    rerender(tree({ open: true, onOpenChange: () => undefined }));
    expect(container.textContent).toContain("Panel body");
    unmount();
  });

  test("pressing the trigger asks the owner to toggle", () => {
    const seen: boolean[] = [];
    const { container, unmount } = renderIntoContainer(
      tree({ open: false, onOpenChange: (next: boolean) => seen.push(next) })
    );
    act(() => {
      trigger(container).click();
    });
    expect(seen).toEqual([true]);
    unmount();
  });

  test("the trigger keeps a visible keyboard focus state", () => {
    const { container, unmount } = renderIntoContainer(
      tree({ open: false, onOpenChange: () => undefined })
    );
    expect(trigger(container).className).toContain("focus-visible:ring-2");
    unmount();
  });
});
