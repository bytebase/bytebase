import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import * as alertModule from "./alert";
import { Alert } from "./alert";

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
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

describe("Alert", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders title and description props as structured content", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Alert, {
        title: "Update available",
        description: "A new version is ready to install.",
      })
    );

    expect(container.textContent).toContain("Update available");
    expect(container.textContent).toContain(
      "A new version is ready to install."
    );
    expect(container.querySelector("h5")?.classList.contains("font-semibold"))
      .toBe(true);

    unmount();
  });

  test("uses readable warning copy without fading rich descriptions", () => {
    let clicks = 0;
    const { container, unmount } = renderIntoContainer(
      createElement(Alert, {
        variant: "warning",
        title: "Review required",
        description: createElement(
          "button",
          { type: "button", onClick: () => clicks++ },
          "Review changes"
        ),
      })
    );

    const alert = container.querySelector('[role="alert"]');
    expect(alert?.classList.contains("text-main")).toBe(true);
    expect(alert?.classList.contains("bg-warning/10")).toBe(true);
    expect(alert?.querySelector("svg")?.classList.contains("text-warning-hover"))
      .toBe(true);
    const button = container.querySelector("button");
    expect(button?.parentElement?.classList.contains("opacity-90")).toBe(false);
    act(() => button?.click());
    expect(clicks).toBe(1);

    unmount();
  });

  test("preserves consumer foreground overrides and unstructured content", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        Alert,
        { variant: "warning", className: "text-warning-hover" },
        createElement("a", { href: "/settings" }, "Open settings")
      )
    );

    const alert = container.querySelector('[role="alert"]');
    expect(alert?.classList.contains("text-warning-hover")).toBe(true);
    expect(alert?.classList.contains("text-main")).toBe(false);
    expect(alert?.querySelector("a")?.getAttribute("href")).toBe("/settings");

    unmount();
  });

  test("renders a compact caption alert", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(Alert, { appearance: "caption", variant: "warning" }, "Risk")
    );

    const alert = container.querySelector('[role="alert"]');
    expect(alert?.className).toContain("text-xs");
    expect(alert?.className).toContain("border-0");
    expect(alert?.classList.contains("text-main")).toBe(true);
    expect(alert?.querySelector("svg")?.getAttribute("class")).toContain(
      "size-3.5"
    );

    unmount();
  });

  test("only exports the alert primitive", () => {
    expect("AlertTitle" in alertModule).toBe(false);
    expect("AlertDescription" in alertModule).toBe(false);
  });
});
