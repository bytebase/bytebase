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

    unmount();
  });

  test("only exports the alert primitive", () => {
    expect("AlertTitle" in alertModule).toBe(false);
    expect("AlertDescription" in alertModule).toBe(false);
  });
});
